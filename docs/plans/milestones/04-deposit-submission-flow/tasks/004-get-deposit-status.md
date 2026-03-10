# Task 004: GET Deposit Status

## Goal

Expose GET /deposits/:id or GET /transfers/:id for status lookup.

## Deliverables

- [ ] GET endpoint returns transfer with current state
- [ ] Include state, amount, account, timestamps
- [ ] 404 if not found

## Notes

- Response includes full state for transfer status tracking UI

## Verification

- GET returns transfer; 404 for unknown id
