# Task 003: Idempotency Middleware

## Goal

Add optional idempotency middleware for X-Idempotency-Key header.

## Deliverables

- [ ] `internal/api/middleware.go` — idempotency middleware
- [ ] Check X-Idempotency-Key; if present and seen, return cached response
- [ ] Document as optional for MVP; implement if time permits

## Notes

- Plan: "optional for MVP; document as follow-up"
- Use idempotency_keys table

## Verification

- Same idempotency key returns same response (200, no duplicate ledger)
