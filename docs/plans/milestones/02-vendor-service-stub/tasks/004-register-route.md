# Task 004: Register Vendor Route

## Goal

Wire the Vendor stub into the HTTP server and expose `POST /vendor/validate`.

## Deliverables

- [ ] `cmd/server/main.go` — register `POST /vendor/validate`
- [ ] Route accepts image payload (or body); passes to stub handler
- [ ] Returns 200 with stub JSON response

## Notes

- Stub may accept minimal body (account_id, etc.)

## Verification

```bash
curl -X POST http://localhost:8080/vendor/validate -H "Content-Type: application/json" -d '{"account_id":"ACC-IQA-BLUR"}'
# Returns blur fail JSON
```
