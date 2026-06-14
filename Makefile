.PHONY: dev build test vet migrate-up migrate-down seed css docker-up docker-down

dev:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

test:
	go test ./...

vet:
	go vet ./...

migrate-up:
	go run ./cmd/server -migrate-only

migrate-down:
ifdef MIGRATE_DOWN
	go run ./scripts/migrate_down.go || true
endif

seed:
	go run ./scripts/seed.go

css:
	npx tailwindcss -i ./web/static/css/input.css -o ./web/static/css/app.css --minify

css-watch:
	npx tailwindcss -i ./web/static/css/input.css -o ./web/static/css/app.css --watch

docker-up:
	docker compose up --build

docker-down:
	docker compose down

.PHONY: tidy
tidy:
	go mod tidy
