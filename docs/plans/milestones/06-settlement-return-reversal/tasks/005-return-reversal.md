# Task 005: Return Reversal

## Goal

Implement return processing: debit investor, $30 fee, credit omnibus, state → Returned.

## Deliverables

- [ ] `internal/return/reversal.go` — process return
- [ ] Accept transfer_id or transaction_id
- [ ] Verify transfer in FundsPosted or Completed
- [ ] Create reversal ledger entries: debit investor (amount + $30), credit omnibus
- [ ] Transition → Returned
- [ ] Idempotent: duplicate return for same transfer returns 200 with existing result

## Notes

- $30 fee hard-coded for MVP

## Verification

```bash
go test ./internal/return/...
```
