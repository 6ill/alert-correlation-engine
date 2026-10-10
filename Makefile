-include .env

.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

.PHONY: db/migrations/up
db/migrations/up: confirm
	@echo 'Running up migrations...'
	migrate -path migrations -database "${DATABASE_URL}?sslmode=disable" up

.PHONY: db/migrations/down
db/migrations/down: confirm
	@echo 'Downgrade migrations...'
	migrate -path migrations -database "${DATABASE_URL}?sslmode=disable" down 1

.PHONY: db/migrations/new
db/migrations/new: confirm
	@echo 'Creating migration files for ${name}...'
	migrate create -seq -ext=.sql -dir=./migrations ${name}

.PHONY: docker/prod/build
docker/prod/build:
	docker compose -f docker-compose.prod.yml build

.PHONY: docker/prod/migrate
docker/prod/migrate: confirm
	docker compose -f docker-compose.prod.yml --env-file .env.production --profile migrate run --rm migrate

.PHONY: docker/prod/up
docker/prod/up:
	docker compose -f docker-compose.prod.yml --env-file .env.production up -d

.PHONY: docker/prod/down
docker/prod/down:
	docker compose -f docker-compose.prod.yml --env-file .env.production down