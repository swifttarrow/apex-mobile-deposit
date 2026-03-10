# Task 006: Returns API

## Goal

Expose POST /returns for return notification.

## Deliverables

- [ ] `internal/api/returns.go` — POST /returns
- [ ] Body: {transfer_id} or {transaction_id}
- [ ] Calls reversal logic; returns 200 with result
- [ ] Idempotent for duplicate returns

## Notes

- Simulated return for stub

## Verification

- POST /returns with transfer_id → reversal posted, Returned
