# Stage 1: deps
FROM golang:1.22-alpine AS deps

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

# Stage 2: build
FROM deps AS builder

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-w -s -extldflags '-static'" \
    -trimpath \
    -o /bin/auth \
    ./cmd/auth

# Stage 3: dev (air hot-reload)
FROM deps AS dev

RUN go install github.com/air-verse/air@latest

WORKDIR /app
COPY . .

EXPOSE 8081 9091
CMD ["air", "-c", ".air.toml"]

# Stage 4: production
FROM gcr.io/distroless/static-debian12:nonroot AS production

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo                 /usr/share/zoneinfo
COPY --from=builder /bin/auth                           /auth

# Run as root so Docker Compose file-based secrets (0400, owned by root) are readable.
# In production Kubernetes, use mounted secrets with proper fsGroup instead.
USER root

EXPOSE 8081 9091

ENTRYPOINT ["/auth"]