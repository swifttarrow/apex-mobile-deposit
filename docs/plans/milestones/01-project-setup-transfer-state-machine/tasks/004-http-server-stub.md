# Task 004: HTTP Server Stub

## Goal

Create the HTTP server entry point so `make dev` starts a listening server.

## Deliverables

- [ ] `cmd/server/main.go` — entry point, HTTP server listening
- [ ] `config/scenarios.json` — placeholder (empty object or minimal structure)
- [ ] Server starts without panic; fails gracefully if DB not ready

## Notes

- Use `net/http` or a minimal router (e.g. chi, echo)
- Port from env or default (e.g. 8080)

## Verification

```bash
make dev
# Server starts; curl http://localhost:8080/ returns something (even 404)
```
