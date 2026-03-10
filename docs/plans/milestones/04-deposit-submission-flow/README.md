# Milestone 4: Deposit Submission Flow

## Overview

REST API: `POST /deposits` accepts images, amount, account. Orchestrates Vendor stub → Funding → ledger + state updates. Idempotency via `X-Idempotency-Key` (optional for MVP).

**Source:** [MVP Plan Phase 4](../../../thoughts/plans/2025-03-10-checkstream-mvp.md#phase-4-deposit-submission-flow)

## Dependencies

- [ ] Milestone 1: Project Setup & Transfer State Machine
- [ ] Milestone 2: Vendor Service Stub
- [ ] Milestone 3: Funding Service & Ledger

## Changes Required

- `internal/api/deposits.go` — POST /deposits handler
- `internal/api/middleware.go` — Idempotency middleware (optional)
- `cmd/server/main.go` — wire deposits handler, Vendor, Funding, ledger
- `internal/transfer/repository.go` — create transfer, update state in transaction with ledger

## Success Criteria

### Automated Verification

- [ ] `go test ./internal/api/...` — happy path, IQA fail, duplicate, over-limit
- [ ] Integration test: ACC-IQA-BLUR → Rejected, no ledger

### Manual Verification

- [ ] Happy path: clean pass, under limit → FundsPosted, ledger entry
- [ ] IQA blur/glare → Rejected, actionable message
- [ ] Over $5K → Rejected

## Tasks

- [001-deposits-handler](./tasks/001-deposits-handler.md)
- [002-orchestration-flow](./tasks/002-orchestration-flow.md)
- [003-idempotency-middleware](./tasks/003-idempotency-middleware.md)
- [004-get-deposit-status](./tasks/004-get-deposit-status.md)
- [005-integration-tests](./tasks/005-integration-tests.md)
