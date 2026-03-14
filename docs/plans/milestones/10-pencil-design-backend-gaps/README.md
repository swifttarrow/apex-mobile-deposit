# Milestone 10: Pencil Design — Backend Gaps

## Overview

Add backend APIs required to power the Pencil design UIs: Deposit History (for mobile) and Operator Queue pagination (for operator table).

**Source:** [Stretch Goals Plan § Phase 3.1](../../2025-03-12-stretch-goals.md#phase-31-backend-gaps-power-uis)

## Dependencies

- [x] Milestone 4: Deposit Submission Flow
- [x] Milestone 5: Operator Workflow
- [x] Milestone 8: Operator Workflow UI
- [x] Milestone 9: Mobile App Check Capture

## Changes Required

- `internal/transfer/repository.go` — Add `ListTransfersByAccount(accountID, limit, offset, status)`
- `internal/api/deposits.go` — Add handler for `GET /deposits` with query params
- `internal/operator/repository.go` — Extend `ListFlaggedTransfers` for pagination
- `internal/api/operator.go` — Add `limit`, `offset` to Queue handler
- `cmd/server/main.go` — Register `GET /deposits` route

## Success Criteria

### Automated Verification

- [ ] `go test ./internal/api/...` — tests for `GET /deposits?account_id=...`
- [ ] `make test` passes
- [ ] Operator queue pagination tests

### Manual Verification

- [ ] `GET /deposits?account_id=ACC-001` returns transfers for that account, ordered by `created_at DESC`
- [ ] Pagination params work; `count` reflects returned items
- [ ] `GET /operator/queue?limit=10&offset=0` returns paginated results

## Tasks

- [001-deposit-history-repository](./tasks/001-deposit-history-repository.md)
- [002-deposit-history-endpoint](./tasks/002-deposit-history-endpoint.md)
- [003-operator-queue-pagination](./tasks/003-operator-queue-pagination.md)
- [004-deposit-history-api-tests](./tasks/004-deposit-history-api-tests.md)
