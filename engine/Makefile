.PHONY: build run test migrate-up migrate-down lint clean

build:
	go build -o bin/engine ./cmd/engine

run:
	go run ./cmd/engine

test:
	go test ./... -v -timeout 30s

test-unit:
	go test ./... -v -short -timeout 30s

migrate-up:
	bash scripts/migrate.sh up

migrate-down:
	bash scripts/migrate.sh down

migrate-status:
	bash scripts/migrate.sh status

lint:
	go vet ./...
	go build ./...

clean:
	rm -rf bin/ tmp/

docker-up:
	cd .. && docker compose up -d postgres redis

docker-down:
	cd .. && docker compose down
