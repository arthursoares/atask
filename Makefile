BINARY     := atask
CMD        := ./cmd/atask
BUILD_DIR  := ./bin
DOCKER_TAG := atask:latest

.PHONY: build run test lint fmt vet migrate migrate-down sqlc \
        docker-build docker-up docker-down check

## build: compile the binary
build:
	go build -o $(BUILD_DIR)/$(BINARY) $(CMD)

## run: run the server locally
run:
	go run $(CMD)

## test: run all tests
test:
	go test -race -count=1 ./...

## lint: run golangci-lint
lint:
	golangci-lint run ./...

## fmt: format Go source files
fmt:
	gofmt -w -s .
	goimports -w .

## vet: run go vet
vet:
	go vet ./...

## migrate: apply database migrations (up)
migrate:
	goose -dir internal/store/migrations sqlite3 atask.db up

## migrate-down: roll back the last database migration
migrate-down:
	goose -dir internal/store/migrations sqlite3 atask.db down

## sqlc: generate type-safe SQL code
sqlc:
	sqlc generate

## docker-build: build the Docker image
docker-build:
	docker build -t $(DOCKER_TAG) .

## docker-up: start services with docker-compose
docker-up:
	docker compose up -d

## docker-down: stop services with docker-compose
docker-down:
	docker compose down

## check: fmt + vet + lint + test
check: fmt vet lint test
