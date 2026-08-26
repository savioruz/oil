BINARY=engine
COMPOSE=docker-compose
MIGRATION_PATH=migrations/postgres

.PHONY: help test coverage coverage.view coverage.check dev run build clean docker.build docker.start docker.stop setup.linter lint generate generate.mock migrate.up migrate.down migrate.step-up migrate.drop migrate.create setup.githook setup modules

help: ## Display this help screen
	@if [ -z "$(shell which awk)" ]; then \
		echo "awk is required to display help"; \
		exit 1; \
	fi
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_.-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

test: generate generate.mock ## Run tests
	GO_ENV=test go test -v -cover -covermode=atomic ./...

coverage: generate generate.mock ## Run tests with coverage
	GO_ENV=test go test $(shell go list ./... |  grep -v  mocks) -coverprofile=tmp/cover.out  -coverpkg=./...

coverage.view: ## View coverage report
	go tool cover -html=tmp/cover.out

coverage.check: ## Check coverage
	go tool cover -func tmp/cover.out

dev: setup.githook generate ## Run the application in development mode
	go run github.com/air-verse/air

run: generate ## Run the application
	go run .

build: generate ## Build the application
	go build -ldflags '-s -w' -o ${BINARY} ./cmd/app/main.go

clean: ## Clean up the project
	@if [ -f ${BINARY} ] ; then rm ${BINARY} ; fi
	@find . -name *mock* -delete
	@rm -rf docs/ ./di/wire_gen.go tmp/cover.out

docker.build: ## Build the docker image
	$(COMPOSE) build

docker.start: ## Start the docker container
	$(COMPOSE) compose up -d

docker.stop: ## Stop the docker container
	$(COMPOSE) compose down

setup.linter: ## Install golangci-lint
	@echo "Installing golangci-lint"
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin latest

lint: generate generate.mock ## Run linters
	golangci-lint run ./...

lint.fix: generate generate.mock ## Run linters and fix issues
	golangci-lint run ./... --fix

generate: ## Generate code
	go generate -skip="mockgen" ./...
	@if [ ! -f permissions/permissions.json ]; then \
		echo '{"skip": true, "endpoints": []}' > permissions/permissions.json; \
	fi
	@echo "Upgrading to OpenAPI 3.1..."
	@npx @scalar/cli document upgrade docs/swagger.json --output docs/openapi.json

generate.mock: ## Generate mock code
	go generate -run="mockgen" ./...

migrate.up: ## Run database migrations up
	go run cmd/migrate/main.go up

migrate.down: ## Run database migrations down
	go run cmd/migrate/main.go down

migrate.step-up: ## Run database migrations step up
	go run cmd/migrate/main.go step-up

migrate.drop: ## Drop all database migrations
	go run cmd/migrate/main.go drop

migrate.create: ## Create a new migration file. Usage: make migrate.create name=<migration name>
	@if [ -z "$(name)" ]; then \
		echo "Please set the name variable"; \
		echo "Example: make migrate.create name=add_users_table"; \
		exit 1; \
	fi
	go run github.com/pressly/goose/v3/cmd/goose@latest create -s --dir $(MIGRATION_PATH) $(name) sql

setup.githook: ## Setup git hooks
	git config core.hooksPath .githooks

setup: setup.githook setup.linter ## Setup the project (git hooks, linter, scalar CLI)
	@echo "Installing Scalar CLI"
	npm install -g @scalar/cli

modules: ## Create simple empty module file. Usage: make modules name=<modules name>
	@if [ -z "$(name)" ]; then \
    	echo "Please set the name variable"; \
    	echo "Example: make modules name=service"; \
    	exit 1; \
    fi
	@mkdir -p ./internal/modules/$(name)/model/dto
	@echo "package model" > ./internal/modules/$(name)/model/model.go
	@echo "package dto" > ./internal/modules/$(name)/model/dto/dto.go
	@mkdir -p ./internal/modules/$(name)/repository
	@echo "package repository" > ./internal/modules/$(name)/repository/repository.go
	@mkdir -p ./internal/modules/$(name)/service
	@echo "package service" > ./internal/modules/$(name)/service/service.go
	@echo "Module $(name) created successfully"
