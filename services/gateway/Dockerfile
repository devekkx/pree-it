FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download 2>/dev/null || true

COPY pkg/ pkg/
COPY services/gateway/ services/gateway/

RUN CGO_ENABLED=0 GOOS=linux go build -o /gateway ./services/gateway

# Runtime
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /gateway /usr/local/bin/gateway

# Port injected via GATEWAY_PORT env var — never hardcoded here.
# Secrets mounted at /run/secrets/ by Docker at runtime.
CMD ["gateway"]