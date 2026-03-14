# Milestone 1: Project Setup & Transfer State Machine

## Overview

Initialize Go module, directory layout, SQLite schema, and transfer state machine with valid transitions. Foundation for all subsequent phases.

**Source:** [MVP Plan Phase 1](../../2025-03-10-checkstream-mvp.md#phase-1-project-setup--transfer-state-machine)

## Dependencies

- [ ] None (greenfield)

## Changes Required

- Go module (`go.mod`), Makefile, `cmd/server/main.go`
- SQLite schema: `transfers`, `ledger_entries`, `operator_actions`, `idempotency_keys`, `check_images`
- Transfer state machine: 8 states, valid transitions only
- Config placeholder: `config/scenarios.json`

**Valid transitions:** Requested → Validating; Validating → Rejected | Analyzing; Analyzing → Rejected | Approved; Approved → FundsPosted; FundsPosted → Completed | Returned

## Success Criteria

### Automated Verification

- [ ] `go build ./...` succeeds
- [ ] `go test ./internal/transfer/...` — state transition validation tests pass
- [ ] `make dev` starts server (or fails gracefully if DB not ready)

### Manual Verification

- [ ] Schema applied; `transfers` table exists
- [ ] Invalid transition (e.g. Rejected → Approved) is rejected in code

## Tasks

- [001-go-module-makefile](./tasks/001-go-module-makefile.md)
- [002-sqlite-schema](./tasks/002-sqlite-schema.md)
- [003-transfer-state-machine](./tasks/003-transfer-state-machine.md)
- [004-http-server-stub](./tasks/004-http-server-stub.md)
