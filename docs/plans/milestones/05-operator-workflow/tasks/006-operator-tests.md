# Task 006: Operator Tests

## Goal

Add tests for operator workflow: MICR fail flagged, operator approve → ledger posted, actions logged.

## Deliverables

- [ ] Test: MICR fail deposit flagged; operator approve → ledger posted
- [ ] Test: operator actions logged
- [ ] Test: reject → Rejected

## Notes

- Integration test with test DB

## Verification

```bash
go test ./internal/api/...
```
