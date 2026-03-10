# Task 004: Funding Rules

## Goal

Implement business rules: reject if amount > $5K, reject if duplicate (same check/transaction ID).

## Deliverables

- [ ] `internal/funding/funding.go` — limit check, duplicate check
- [ ] Account resolution (account_id → omnibus_id)
- [ ] Unit tests: over-limit rejected, duplicate rejected

## Notes

- Duplicate detection: same transaction_id or check identifier

## Verification

```bash
go test ./internal/funding/...
```
