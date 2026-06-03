# Stage 1: deps
FROM golang:1.26-alpine AS deps

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

# Stage 4: production (distroless — no shell, no package manager)
FROM gcr.io/distroless/static-debian12 AS production

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo                 /usr/share/zoneinfo
COPY --from=builder /bin/auth                           /auth

USER nonroot:nonroot

EXPOSE 8081 9091

ENTRYPOINT ["/auth"]