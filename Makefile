CGO_ENABLED=1

.PHONY: dev dev-reload test test-report build build-web bench demo demo-install seed-deposits

build-web:
	cd web && npm install && npm run build

dev:
	CGO_ENABLED=1 go run ./cmd/server/...

# Hot reload: restarts server when .go, .html, .json files change
dev-reload:
	CGO_ENABLED=1 go run github.com/air-verse/air@latest

test:
	CGO_ENABLED=1 go test ./...

# Generate test report for deliverables (reports/test_report.txt)
test-report:
	@mkdir -p reports
	CGO_ENABLED=1 go test -v ./... 2>&1 | tee reports/test_report.txt

build:
	CGO_ENABLED=1 go build -o bin/checkstream ./cmd/server/...

bench:
	@mkdir -p reports
	CGO_ENABLED=1 go test -bench=. -benchmem ./... 2>&1 | tee reports/benchmark.txt

# Browser demo: runs all user scenarios in Playwright (requires server running)
demo-install:
	cd e2e && npm install && npx playwright install chromium

demo:
	cd e2e && npm run demo

# Same as demo but opens a visible browser so you can watch.
# Optional: SLOW=500 (ms between actions) to slow down, e.g. make demo-headed SLOW=500
demo-headed:
	cd e2e && DEMO_SLOW=$(SLOW) npm run demo:headed

# Insert 25 deposits (15 before 6:30 PM CT, 10 after) for settlement demo. Requires server DB.
seed-deposits:
	CGO_ENABLED=1 go run ./cmd/seed-deposits/...
