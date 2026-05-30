MIGRATE=docker compose run --rm migrate

-include .env
export

ifndef POSTGRES_USER 
$(error POSTGRES_USER is not set.)
endif

ifndef POSTGRES_DB
$(error POSTGRES_USER is not set.)
endif

ifndef POSTGRES_PASSWORD
$(error POSTGRES_USER is not set.)
endif

DB_URL=postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@db:5432/$(POSTGRES_DB)?sslmode=disable

count ?= 1
name ?= migrations

up: 
	$(MIGRATE) -path /migrations -database "${DB_URL}" up

down:
	$(MIGRATE) -path /migrations -database "${DB_URL}" down $(count)

force: 
	$(MIGRATE) -path /migrations -database "${DB_URL}" force $(count)

create: 
	$(MIGRATE) create -ext sql -dir /migrations -seq $(name)
