# Task 003: Operator Queue Pagination

## Goal

Add `limit` and `offset` (or `page`, `page_size`) query params to `GET /operator/queue` to support table pagination.

## Deliverables

- [ ] Extend `ListFlaggedTransfers` in `internal/operator/repository.go` to accept `limit`, `offset int`
- [ ] Add `total` count to response (total flagged matching filters, before pagination)
- [ ] Update `OperatorHandler.Queue` to parse `limit`, `offset` from query; pass to repo
- [ ] Response shape: `{ "transfers": [...], "count": N, "total": N }` — count = returned, total = matching
- [ ] Default limit 20, max 100 when limit <= 0 or > 100

## Notes

- Existing filters: `date`, `account`, `amount_min`, `amount_max` — preserve
- SQL: add `LIMIT ? OFFSET ?` to query; use `COUNT(*)` subquery or separate query for total
- File: `internal/operator/repository.go`, `internal/api/operator.go`

## Verification

- `GET /operator/queue?limit=5&offset=0` returns at most 5 items; `total` reflects full matching count
- Pagination works with existing filters
