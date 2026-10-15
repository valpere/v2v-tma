.PHONY: check run build

check:
	@test -z "$$(gofmt -l .)" || { gofmt -l .; echo "gofmt needed"; exit 1; }
	go vet ./...
	go test -race ./...
	node --check web/assets/app.js
	go build -o bin/server ./cmd/server

build:
	go build -o bin/server ./cmd/server

# Needs a running engine with WEB_ADDR (v2v-demo): by default http://127.0.0.1:8091.
# WEB_DIR=web serves the client from disk, so edits show without a rebuild.
run:
	ENGINE_URL=$${ENGINE_URL:-http://127.0.0.1:8091} WEB_DIR=web go run ./cmd/server
