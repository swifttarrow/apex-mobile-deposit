# Task 008: Settlement & Return Tests

## Goal

Add tests for settlement and return flows.

## Deliverables

- [ ] Test: X9 file generated with correct structure
- [ ] Test: reversal entries, fee, state → Returned
- [ ] Test: EOD cutoff excludes late deposits
- [ ] Test: settlement file contents (MICR, images, amounts)
- [ ] Test: return idempotency (duplicate returns 200)

## Notes

- From gaps: "Settlement file contents validation"

## Verification

```bash
go test ./internal/settlement/... ./internal/return/...
```
