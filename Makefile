.PHONY: check run build css

check:
	@test -z "$$(gofmt -l .)" || { gofmt -l .; echo "gofmt needed"; exit 1; }
	go vet ./...
	go test -race ./...
	for f in web/assets/*.mjs; do node --check $$f || exit 1; done
	node --test web/test/*.test.mjs
	go build -o bin/server ./cmd/server

build:
	go build -o bin/server ./cmd/server

# web/assets/tailwind.css is generated and committed (the runtime CDN is gone). Rerun after
# adding or changing Tailwind classes in web/index.html or web/assets/*.mjs; needs npx + network.
css:
	npx --yes tailwindcss@3.4.19 -c tools/tailwind.config.cjs -i tools/input.css -o web/assets/tailwind.css --minify

# Needs a running engine with WEB_ADDR (v2v-demo): by default http://127.0.0.1:8091.
# WEB_DIR=web serves the client from disk, so edits show without a rebuild.
run:
	ENGINE_URL=$${ENGINE_URL:-http://127.0.0.1:8091} WEB_DIR=web go run ./cmd/server
