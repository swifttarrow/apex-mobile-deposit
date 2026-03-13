# Task 001: Operator Page Route

## Goal

Add `/operator` route and serve the operator page HTML.

## Deliverables

- [x] `cmd/server/web/operator/index.html` (or equivalent) created
- [x] `main.go` updated to serve `/operator` and `/operator/` (similar to `/scenarios`)
- [x] Visiting `/operator` loads the operator page

## Notes

- Follow pattern from `web/scenarios` and `scenarioHandler` in `cmd/server/main.go`
- Page can be minimal placeholder initially; full UI in later tasks

## Verification

- `make dev`; open `/operator` in browser; page loads
