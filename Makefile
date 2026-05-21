APP_NAME=pree-it


.PHONY: up infra down clean logs build test
 
include .env
export
 
up:
	docker compose up -d
 
infra:
	docker compose up -d postgres redis nats prometheus loki grafana
 
down:
	docker compose down
 
clean:
	docker compose down -v
 
logs:
	docker compose logs -f
 
logs-gateway:
	docker compose logs -f gateway
 
logs-auth:
	docker compose logs -f auth
 
build:
	docker compose build
 
test:
	go test ./... -v
 
run-gateway:
	go run ./services/gateway
 
run-auth:
	go run ./services/auth
 
db-shell:
	docker compose exec postgres psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)
 
db-reset:
	docker compose rm -sf postgres
	docker volume rm $$(docker volume ls -q | grep postgres) 2>/dev/null || true
	docker compose up -d postgres
 