# ── Stage 1: deps ─────────────────────────────────────────────────────────────
FROM golang:1.26-alpine AS deps

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

# ── Stage 2: build ────────────────────────────────────────────────────────────
FROM deps AS builder

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-w -s -extldflags '-static'" \
    -trimpath \
    -o /bin/auth \
    ./cmd/auth

# ── Stage 3: dev (air hot-reload) ─────────────────────────────────────────────
FROM deps AS dev

RUN go install github.com/air-verse/air@latest

WORKDIR /app
COPY . .

EXPOSE 8081 9091
CMD ["air", "-c", ".air.toml"]

# ── Stage 4: production ────────────────────────────────────────────────────────
# Alpine instead of distroless — sh is required to read Docker secrets in the
# entrypoint before exec-ing the Go binary. Alpine is ~8MB, has no package
# manager cache, and runs as a non-root user.
FROM alpine:3.22.4 AS production

# Install only what is needed at runtime
RUN apk add --no-cache ca-certificates tzdata wget

# Create a non-root user with a fixed uid/gid
RUN addgroup -g 10001 -S app \
 && adduser  -u 10001 -S app -G app

COPY --from=builder /bin/auth /auth

# Drop to non-root before the process starts
USER app

EXPOSE 8081 9091

ENTRYPOINT ["/auth"]