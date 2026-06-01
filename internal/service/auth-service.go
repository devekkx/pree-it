package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/devekkx/pree-it-auth/internal/db"
	"github.com/devekkx/pree-it-auth/internal/token"
	"github.com/devekkx/pree-it-auth/pkg/hash"
	"github.com/devekkx/pree-it-auth/pkg/password"
	"github.com/google/uuid"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailTaken         = errors.New("email already registered")
	ErrTokenReuse         = errors.New("refresh token reuse detected")
	ErrTokenExpired       = errors.New("refresh token expired")
	ErrTokenNotFound      = errors.New("refresh token not found")
)

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

type AuthService struct {
	store  *db.Store
	tokens *token.Manager
}

func New(store *db.Store, tm *token.Manager) *AuthService {
	return &AuthService{store: store, tokens: tm}
}

//  Register

func (s *AuthService) Register(ctx context.Context, email, plaintext string) (db.AuthSchemaUser, error) {
	_, err := s.store.GetUserByEmail(ctx, email)
	if err == nil {
		return db.AuthSchemaUser{}, ErrEmailTaken
	}

	hashed, err := password.Hash(plaintext)
	if err != nil {
		return db.AuthSchemaUser{}, fmt.Errorf("register: hash password: %w", err)
	}

	user, err := s.store.CreateUser(ctx, db.CreateUserParams{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: hashed,
		IsVerified:   false,
	})
	if err != nil {
		return db.AuthSchemaUser{}, fmt.Errorf("register: create user: %w", err)
	}

	return user, nil
}

//  Login

func (s *AuthService) Login(ctx context.Context, email, plaintext string) (db.AuthSchemaUser, *TokenPair, error) {
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		// User not found. Run a dummy Argon2id verification to maintain
		// constant-time response and prevent timing-based email enumeration.
		// The dummy hash is a valid encoded string so decodeHash succeeds
		// and the full KDF runs — not just a fast-path return.
		_ = password.Verify(plaintext, dummyHash)
		return db.AuthSchemaUser{}, nil, ErrInvalidCredentials
	}

	if err := password.Verify(plaintext, user.PasswordHash); err != nil {
		return db.AuthSchemaUser{}, nil, ErrInvalidCredentials
	}

	// Transparent rehash: if the stored hash used old parameters, upgrade
	// it now that we have the plaintext in hand.
	if password.NeedsRehash(user.PasswordHash) {
		if newHash, err := password.Hash(plaintext); err == nil {
			_, _ = s.store.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
				ID:           user.ID,
				PasswordHash: newHash,
			})
		}
	}

	pair, err := s.issuePair(ctx, user, uuid.New())
	if err != nil {
		return db.AuthSchemaUser{}, nil, err
	}

	return user, pair, nil
}

//  Refresh

func (s *AuthService) Refresh(ctx context.Context, rawToken string) (*TokenPair, error) {
	tokenHash := hash.Token(rawToken)

	existing, err := s.store.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		return nil, ErrTokenNotFound
	}

	if existing.Used {
		_ = s.store.InvalidateTokenFamily(ctx, existing.Family)
		return nil, ErrTokenReuse
	}

	if time.Now().After(existing.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	user, err := s.store.GetUserByID(ctx, existing.UserID)
	if err != nil {
		return nil, fmt.Errorf("refresh: user not found: %w", err)
	}

	var pair *TokenPair

	err = s.store.ExecTx(ctx, func(q *db.Queries) error {
		if _, err := q.MarkRefreshTokenUsed(ctx, existing.ID); err != nil {
			return fmt.Errorf("mark used: %w", err)
		}
		newPair, err := s.issuePairTx(ctx, q, user, existing.Family)
		if err != nil {
			return err
		}
		pair = newPair
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("refresh: transaction: %w", err)
	}

	return pair, nil
}

//  Logout

func (s *AuthService) Logout(ctx context.Context, rawToken string) error {
	tokenHash := hash.Token(rawToken)
	existing, err := s.store.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		return nil // already gone — idempotent
	}
	if err := s.store.InvalidateTokenFamily(ctx, existing.Family); err != nil {
		return fmt.Errorf("logout: invalidate family: %w", err)
	}
	return nil
}

//	Private
//
// issuePair issues a new token pair using the pool-backed Queries.
// It accesses store.Queries — the anonymous embedded *Queries field —
// to call issuePairTx outside of a transaction context.
func (s *AuthService) issuePair(
	ctx context.Context,
	user db.AuthSchemaUser,
	family uuid.UUID,
) (*TokenPair, error) {
	// store.Queries is the anonymous embedded *Queries.
	// With anonymous embedding, Go creates a field whose name IS the type name,
	// so store.Queries gives you the *Queries pointer directly.
	return s.issuePairTx(ctx, s.store.Queries, user, family)
}

// issuePairTx accepts any *Queries — pool-backed or transaction-backed.
// This is the single place that signs the access token and stores the refresh token.
func (s *AuthService) issuePairTx(
	ctx context.Context,
	q *db.Queries,
	user db.AuthSchemaUser,
	family uuid.UUID,
) (*TokenPair, error) {
	accessToken, err := s.tokens.NewAccessToken(user.ID.String(), user.Email)
	if err != nil {
		return nil, fmt.Errorf("issue pair: sign access token: %w", err)
	}

	rawRefresh, expiresAt := s.tokens.NewRefreshToken()

	_, err = q.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: hash.Token(rawRefresh),
		Family:    family,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return nil, fmt.Errorf("issue pair: store refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		ExpiresAt:    expiresAt,
	}, nil
}

// dummyHash is a pre-computed valid Argon2id hash of the string "dummy".
// It is used in Login's not-found path so the full KDF always runs,
// preventing timing-based email enumeration.
// Regenerate with: password.Hash("dummy") and paste the output here.
const dummyHash = "$argon2id$v=19$m=65536,t=3,p=2$c29tZXJhbmRvbXNhbHQ$RdescudvJCsgt3ub+b+dWRWJTmaaJObG"
