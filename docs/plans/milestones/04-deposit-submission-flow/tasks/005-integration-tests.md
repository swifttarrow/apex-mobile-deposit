# Task 005: Integration Tests

## Goal

Add integration tests for deposit flow: happy path, IQA fail, duplicate, over-limit.

## Deliverables

- [ ] Test: ACC-IQA-BLUR → Rejected, no ledger
- [ ] Test: happy path → FundsPosted, ledger entry
- [ ] Test: duplicate rejected
- [ ] Test: over $5K rejected

## Notes

- Use test DB or in-memory SQLite

## Verification

```bash
go test ./internal/api/...
```
