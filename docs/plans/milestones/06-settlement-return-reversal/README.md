# Milestone 6: Settlement & Return/Reversal

## Overview

Settlement file (X9 ICL via Moov) with EOD cutoff. Return handler: reversal posting, $30 fee, state → Returned. Investor notification stub; minimal settlement ack tracking (full deliverable).

**Source:** [MVP Plan Phase 6](../../../thoughts/plans/2025-03-10-checkstream-mvp.md#phase-6-settlement--returnreversal) | [Gaps: Investor notification, settlement ack](../../../thoughts/plans/2025-03-10-checkstream-mvp-to-full-deliverable.md#3-returnreversal-gaps)

## Dependencies

- [ ] Milestone 1: Project Setup & Transfer State Machine
- [ ] Milestone 3: Funding Service & Ledger
- [ ] Milestone 5: Operator Workflow

## Changes Required

- `internal/settlement/engine.go` — batch FundsPosted, generate X9 ICL
- `internal/settlement/eod.go` — EOD cutoff (6:30 PM CT)
- `internal/api/settlement.go` — POST /settlement/trigger
- `internal/return/reversal.go` — debit investor, $30 fee, credit omnibus, state → Returned
- `internal/api/returns.go` — POST /returns
- Investor notification stub (log) on return
- Minimal settlement ack tracking (e.g. settlement_ack_at on transfer)

## Success Criteria

### Automated Verification

- [ ] `go test ./internal/settlement/...` — X9 file generated with correct structure
- [ ] `go test ./internal/return/...` — reversal entries, fee, state
- [ ] EOD cutoff: deposit after 6:30 PM CT excluded from same-day batch
- [ ] Settlement file contents: assert MICR, images, amounts present

### Manual Verification

- [ ] Settlement file contains MICR, images, amounts
- [ ] Return → reversal posted, $30 fee, Returned
- [ ] Investor notification logged (stub)

## Tasks

- [001-moov-dependency](./tasks/001-moov-dependency.md)
- [002-eod-cutoff](./tasks/002-eod-cutoff.md)
- [003-settlement-engine](./tasks/003-settlement-engine.md)
- [004-settlement-trigger-api](./tasks/004-settlement-trigger-api.md)
- [005-return-reversal](./tasks/005-return-reversal.md)
- [006-returns-api](./tasks/006-returns-api.md)
- [007-investor-notification-settlement-ack](./tasks/007-investor-notification-settlement-ack.md)
- [008-settlement-tests](./tasks/008-settlement-tests.md)
