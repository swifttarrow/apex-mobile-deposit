# Task 002: SQLite Schema

## Goal

Define and apply the SQLite schema for transfers, ledger entries, operator actions, idempotency keys, and check images.

## Deliverables

- [ ] `internal/db/schema.sql` with DDL for all tables
- [ ] `internal/db/db.go` — SQLite connection and migration/apply logic
- [ ] Schema includes: `id`, `account_id`, `amount`, `state`, `vendor_response` (JSON), `front_image_path`, `back_image_path`, `micr_data` (JSON), `ocr_amount`, `entered_amount`, `transaction_id`, `created_at`, `updated_at` on transfers

## Notes

- Tables: `transfers`, `ledger_entries`, `operator_actions`, `idempotency_keys`, `check_images`
- Use SQLite with file path from env or default

## Verification

- Run migrations; `transfers` table exists
- `SELECT * FROM transfers` returns empty result
