# Task 003: Funding Config

## Goal

Define and load funding config: omnibus map, $5K limit, account type defaults.

## Deliverables

- [ ] `internal/funding/config.go` — omnibus map (account_id → omnibus_id)
- [ ] $5,000 deposit limit constant or config
- [ ] Config loadable (env or file)

## Notes

- Omnibus lookup required for ledger From AccountId

## Verification

- Config resolves account to omnibus
