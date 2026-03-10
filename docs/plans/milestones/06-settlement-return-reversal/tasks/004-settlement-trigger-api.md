# Task 004: Settlement Trigger API

## Goal

Expose POST /settlement/trigger for manual EOD trigger.

## Deliverables

- [ ] `internal/api/settlement.go` — POST /settlement/trigger
- [ ] Triggers settlement engine for given date or "today"
- [ ] Returns summary (count of deposits settled, file path)

## Notes

- Manual trigger for demo/testing

## Verification

- POST triggers settlement; file written; transfers → Completed
