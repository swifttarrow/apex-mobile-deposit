# Task 003: Transfer State Machine

## Goal

Implement the transfer state machine with 8 states and valid transitions; invalid transitions must be rejected.

## Deliverables

- [ ] `internal/transfer/state.go` — state enum, valid transitions map
- [ ] `internal/transfer/transfer.go` — Transfer model, transition validation logic
- [ ] `internal/transfer/repository.go` — CRUD, state updates
- [ ] Unit tests: valid transitions pass; invalid (e.g. Rejected → Approved) return error

## Notes

**States:** Requested, Validating, Analyzing, Approved, FundsPosted, Completed, Rejected, Returned

**Transitions:**
- Requested → Validating
- Validating → Rejected | Analyzing
- Analyzing → Rejected | Approved
- Approved → FundsPosted
- FundsPosted → Completed | Returned

## Verification

```bash
go test ./internal/transfer/...
```
