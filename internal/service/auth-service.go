package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/devekkx/pree-it-auth/internal/token"
	"github.com/devekkx/pree-it-auth/pkg/hash"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Sentinel errors - handlers switch on these; never expose raw DB errors.
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailTaken         = errors.New("email already registered")
	ErrTokenReuse         = errors.New("refresh token reuse detected")
	ErrTokenExpired       = errors.New("refresh token expired")
	ErrTokenNotFound      = errors.New("refresh token not found")
)

// TokenPair is the pair issued on login and refresh.
// RefreshToken is the raw token returned once to the client - never stored raw.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

// AuthService owns all authentication business logic.
// It never touches HTTP - that is the handler's concern.
type AuthService struct {
	store  *db.Store
	tokens *token.Manager
}

// New constructs an AuthService. Both arguments are required.
func New(store *db.Store, tm *token.Manager) *AuthService {
	return &AuthService{store: store, tokens: tm}
}

// Register

// Register creates a new user account.
// Returns ErrEmailTaken if the email is already in use - but note that
// handlers must NOT expose this to clients (prevents email enumeration).
func (s *AuthService) Register(ctx context.Context, email, password string) (db.AuthSchemaUser, error) {
	_, err := s.store.GetUserByEmail(ctx, email)
	if err == nil {
		// User found - email is taken.
		return db.AuthSchemaUser{}, ErrEmailTaken
	}

	// bcrypt cost 12 is ~300ms on commodity hardware - adjust per threat model.
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return db.AuthSchemaUser{}, fmt.Errorf("register: hash password: %w", err)
	}

	user, err := s.store.CreateUser(ctx, db.CreateUserParams{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hash),
		IsVerified:   false,
	})
	if err != nil {
		return db.AuthSchemaUser{}, fmt.Errorf("register: create user: %w", err)
	}

	return user, nil
}

// Login

// Login authenticates a user and returns a signed token pair.
// Always returns ErrInvalidCredentials on any failure - never reveals
// whether the email exists or the password was wrong (prevents enumeration).
func (s *AuthService) Login(ctx context.Context, email, password string) (db.AuthSchemaUser, *TokenPair, error) {
	user, err := s.store.GetUserByEmail(ctx, email)
	if err != nil {
		// User not found. Run a dummy bcrypt comparison to maintain
		// constant-time response - prevents timing-based email enumeration.
		_ = bcrypt.CompareHashAndPassword(
			[]byte("$2a$12$aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
			[]byte(password),
		)
		return db.AuthSchemaUser{}, nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return db.AuthSchemaUser{}, nil, ErrInvalidCredentials
	}

	pair, err := s.issuePair(ctx, user, uuid.New())
	if err != nil {
		return db.AuthSchemaUser{}, nil, err
	}

	return user, pair, nil
}

// Refresh

// Refresh validates an existing refresh token and atomically issues a new pair.
// If the token has already been used (reuse attack), the entire token family
// is invalidated, forcing re-authentication on all devices.
func (s *AuthService) Refresh(ctx context.Context, rawToken string) (*TokenPair, error) {
	tokenHash := hash.Token(rawToken)

	existing, err := s.store.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		return nil, ErrTokenNotFound
	}

	// Reuse detection
	if existing.Used {
		// A used token was presented again. This is either a replay attack
		// or a client bug. Invalidate the entire family to force re-login
		// on all sessions that share this login chain.
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

	// Atomic rotation
	// Mark the current token used AND insert the new token in a single
	// serializable transaction. If either fails, the client retains their
	// existing token and can retry.
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

// Logout

// Logout invalidates the entire token family for this session.
// Idempotent - safe to call if the token is already gone.
func (s *AuthService) Logout(ctx context.Context, rawToken string) error {
	tokenHash := hash.Token(rawToken)

	existing, err := s.store.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		// Already gone - treat as success.
		return nil
	}

	if err := s.store.InvalidateTokenFamily(ctx, existing.Family); err != nil {
		return fmt.Errorf("logout: invalidate family: %w", err)
	}
	return nil
}

// Private helpers

// issuePair issues a new token pair using the pool-backed Queries.
func (s *AuthService) issuePair(ctx context.Context, user db.AuthSchemaUser, family uuid.UUID) (*TokenPair, error) {
	return s.issuePairTx(ctx, s.store.Queries, user, family)
}

// issuePairTx issues a new token pair using the provided Queries (may be transactional).
func (s *AuthService) issuePairTx(ctx context.Context, q *db.Queries, user db.AuthSchemaUser, family uuid.UUID) (*TokenPair, error) {
	accessToken, err := s.tokens.NewAccessToken(user.ID.String(), user.Email)
	if err != nil {
		return nil, fmt.Errorf("issue pair: sign access token: %w", err)
	}

	rawRefresh, expiresAt := s.tokens.NewRefreshToken()
	tokenHash := hash.Token(rawRefresh)

	_, err = q.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: tokenHash,
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
