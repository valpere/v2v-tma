.PHONY: check run build

check:
	@test -z "$$(gofmt -l .)" || { gofmt -l .; echo "gofmt needed"; exit 1; }
	go vet ./...
	go test -race ./...
	node --check web/assets/app.js
	go build -o bin/server ./cmd/server

build:
	go build -o bin/server ./cmd/server

run:
	DEV_AUTH=1 go run ./cmd/server
