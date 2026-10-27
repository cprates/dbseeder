SHELL := /bin/bash
LINTER=go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0

include .env
export

.PHONY: dev_up
dev_up:
	docker compose rm --force --stop -v postgres
	docker compose up --build --force-recreate -d postgres
	sleep 5

.PHONY: dev_down
dev_down:
	docker compose down
	docker compose rm --force --stop -v postgres

.PHONY: integration_tests
integration_tests: dev_up
	go test -coverpkg=./... -race --tags=integration_tests -parallel 5 -count=1 -v -cover ./...
	@make dev_down

.PHONY: coverage
coverage: integration_tests
	go tool cover -html=coverage.out

.PHONY: lint
lint:
	$(LINTER) run --allow-parallel-runners -c .golangci.yml
