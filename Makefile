CGO_ENABLED=1

.PHONY: dev test build build-web bench

build-web:
	cd web && npm install && npm run build

dev:
	CGO_ENABLED=1 go run ./cmd/server/...

test:
	CGO_ENABLED=1 go test ./...

build:
	CGO_ENABLED=1 go build -o bin/checkstream ./cmd/server/...

bench:
	@mkdir -p reports
	CGO_ENABLED=1 go test -bench=. -benchmem ./... 2>&1 | tee reports/benchmark.txt
