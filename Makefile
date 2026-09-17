include .env

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