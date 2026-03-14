# Task 001: Deposit History Repository

## Goal

Add `ListTransfersByAccount` to the transfer repository to support listing deposits for a given account with pagination and optional status filter.

## Deliverables

- [ ] Add `ListTransfersByAccount(accountID string, limit, offset int, status string) ([]*Transfer, int, error)` to `internal/transfer/repository.go`
- [ ] Query filters by `account_id`; optional `status` filter when non-empty
- [ ] Order by `created_at DESC`
- [ ] Return `(transfers, totalCount, error)` — totalCount for pagination UI
- [ ] Default limit (e.g. 20) when limit <= 0; sensible max (e.g. 100)

## Notes

- SQL: `SELECT ... FROM transfers WHERE account_id = ? [AND state = ?] ORDER BY created_at DESC LIMIT ? OFFSET ?`
- Use `COUNT(*) OVER()` or separate count query for total
- File: `internal/transfer/repository.go`

## Verification

- Unit test or integration test: call with account_id, verify returned transfers belong to account, ordering, limit/offset behavior
