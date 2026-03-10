# Task 001: Ledger Schema

## Goal

Add `ledger_entries` table to schema with fields for MOVEMENT entries.

## Deliverables

- [ ] `ledger_entries` table in `internal/db/schema.sql` (or `internal/ledger/schema.sql`)
- [ ] Fields: To AccountId, From AccountId, Type, Memo, SubType, Transfer Type, Currency, Amount, SourceApplicationId (TransferID)

## Notes

- Type: MOVEMENT, Memo: FREE, SubType: DEPOSIT, Transfer Type: CHECK, Currency: USD

## Verification

- Schema applies; `ledger_entries` table exists
