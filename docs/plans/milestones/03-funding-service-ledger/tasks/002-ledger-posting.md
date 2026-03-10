# Task 002: Ledger Posting

## Goal

Implement ledger posting: create MOVEMENT entry with correct account mapping and metadata.

## Deliverables

- [ ] `internal/ledger/ledger.go` — function to create MOVEMENT entry
- [ ] To AccountId (investor), From AccountId (omnibus)
- [ ] Type: MOVEMENT, Memo: FREE, SubType: DEPOSIT, Transfer Type: CHECK, Currency: USD
- [ ] Amount, SourceApplicationId (TransferID)

## Notes

- Omnibus resolved via config (account_id → omnibus_id)

## Verification

```bash
go test ./internal/ledger/...
```
