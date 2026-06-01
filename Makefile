APP_NAME=pree-it

.PHONY: up infra down clean logs build test secrets

include .env
export

# Generate secret files (run once on a fresh environment)
secrets:
	./scripts/generate-secrets.sh

# Start all services
up:
	docker compose up -d

# Start only infrastructure: db, cache, messaging, observability
infra:
	docker compose up -d postgres redis nats mimir loki tempo alloy grafana

# Stop all containers
down:
	docker compose down

# Stop and delete all volumes (full reset)
clean:
	docker compose down -v

logs:
	docker compose logs -f

logs-gateway:
	docker compose logs -f gateway

logs-auth:
	docker compose logs -f auth

logs-alloy:
	docker compose logs -f alloy

build:
	docker compose build

test:
	go test ./... -v

run-gateway:
	go run ./services/gateway

run-auth:
	go run ./services/auth

db-shell:
	docker compose exec postgres psql -U $$(cat secrets/postgres_user) -d $$(cat secrets/postgres_db)

db-reset:
	docker compose rm -sf postgres
	docker volume rm $$(docker volume ls -q | grep postgres) 2>/dev/null || true
	docker compose up -d postgres