APP_NAME=pree-it

.PHONY: dev-up dev-down logs gateway auth fmt test

dev-up:
	docker compose up -d --build

dev-down:
	docker compose down

logs:
	docker compose logs -f

gateway:
	cd services/gateway-service && air

auth:
	cd services/auth-service && air

fmt:
	go fmt ./...

test:
	go test ./...