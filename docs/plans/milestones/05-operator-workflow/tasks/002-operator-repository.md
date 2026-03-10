# Task 002: Operator Repository

## Goal

Implement repository for querying flagged transfers and recording operator actions.

## Deliverables

- [ ] `internal/operator/repository.go` — query flagged transfers (status=Analyzing)
- [ ] Search/filter by date, status, account, amount
- [ ] Record operator action (approve/reject) with operator_id

## Notes

- Flagged = state Analyzing

## Verification

- Query returns flagged deposits; actions logged
