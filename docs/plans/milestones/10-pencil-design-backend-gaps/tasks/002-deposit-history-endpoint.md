# Task 002: Deposit History Endpoint

## Goal

Add `GET /deposits` endpoint that lists transfers for an account, using the new repository method.

## Deliverables

- [ ] Add `List` handler to `internal/api/deposits.go` (or new method on DepositHandler)
- [ ] Query params: `account_id` (required), `limit`, `offset`, optional `status`
- [ ] Response: `{ "transfers": [...], "count": N, "total": N }`
- [ ] Register `GET /deposits` in `cmd/server/main.go` — must not conflict with `GET /deposits/{id}`; use query-param path or route ordering
- [ ] Return 400 if `account_id` missing

## Notes

- Route ordering: `GET /deposits/{id}` matches first; ensure `GET /deposits` (no path param) is registered. Check Go 1.22+ mux behavior.
- Use `r.URL.Query().Get("account_id")`, `limit`, `offset` from query
- Call `h.transferRepo.ListTransfersByAccount(...)` from Task 001

## Verification

- `curl "http://localhost:8080/deposits?account_id=ACC-001"` returns JSON with transfers array
- Missing account_id returns 400
