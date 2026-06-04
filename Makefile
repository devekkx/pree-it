SQLC  := sqlc
GOOSE := goose
PG_DSN   := "host=localhost user=preeit_admin password=$$(cat ../../preeit-infra/secrets/postgres_password.txt) dbname=preeit sslmode=disable"

.PHONY: sqlc-gen sqlc-vet migrate-up migrate-down migrate-status

sqlc-gen:
	$(SQLC) generate

# sqlc vet catches query/schema drift at codegen time - run in CI.
sqlc-vet:
	$(SQLC) vet

migrate-up:
	$(GOOSE) -dir internal/db/migrations postgres $(PG_DSN) up

migrate-down:
	$(GOOSE) -dir internal/db/migrations postgres $(PG_DSN) down

migrate-status:
	$(GOOSE) -dir internal/db/migrations postgres $(PG_DSN) status