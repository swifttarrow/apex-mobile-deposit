# Task 004: Deposit History API Tests

## Goal

Add automated tests for the Deposit History endpoint to ensure correct behavior and prevent regressions.

## Deliverables

- [ ] Add test in `internal/api/deposits_test.go` (or new file) for `GET /deposits?account_id=...`
- [ ] Test: returns 400 when account_id missing
- [ ] Test: returns empty array for account with no transfers
- [ ] Test: returns transfers for account; ordering by created_at DESC
- [ ] Test: limit/offset pagination works
- [ ] Test: optional status filter works when provided

## Notes

- Use existing test patterns from `deposits_test.go` (database setup, handler wiring)
- Create transfers via `POST /deposits` or directly in DB for test fixtures
- File: `internal/api/deposits_test.go`

## Verification

- `go test ./internal/api/... -run Deposit` passes
- `make test` passes
