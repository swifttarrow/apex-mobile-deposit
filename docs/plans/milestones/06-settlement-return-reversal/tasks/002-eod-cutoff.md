# Task 002: EOD Cutoff

## Goal

Implement EOD cutoff logic: 6:30 PM CT; deposits after cutoff roll to next business day.

## Deliverables

- [ ] `internal/settlement/eod.go` — cutoff logic
- [ ] Determine settlement date from deposit timestamp vs 6:30 PM CT
- [ ] Deposits after cutoff use next business day

## Notes

- 6:30 PM CT = cutoff time

## Verification

- Test: deposit after 6:30 PM CT excluded from same-day batch
