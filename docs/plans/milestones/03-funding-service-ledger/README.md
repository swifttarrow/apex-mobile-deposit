# Milestone 3: Funding Service & Ledger

## Overview

Business rules ($5K limit, duplicate detection, contribution defaults), omnibus lookup, ledger posting. MOVEMENT entries with To/From, SubType DEPOSIT, Transfer Type CHECK.

**Source:** [MVP Plan Phase 3](../../../thoughts/plans/2025-03-10-checkstream-mvp.md#phase-3-funding-service--ledger) | [Gaps: Contribution defaults](../../../thoughts/plans/2025-03-10-checkstream-full-deliverable.md#1-funding-service-gaps)

## Dependencies

- [ ] Milestone 1: Project Setup & Transfer State Machine

## Changes Required

- `internal/funding/funding.go` — session validation, account eligibility, limit check, duplicate check, account resolution, contribution defaults
- `internal/funding/config.go` — omnibus map, $5K limit, contribution type defaults (e.g. individual for retirement)
- `internal/ledger/ledger.go` — create MOVEMENT entry
- `internal/db/schema.sql` — `ledger_entries` table if not in Phase 1

## Success Criteria

### Automated Verification

- [ ] `go test ./internal/funding/...` — session/eligibility rejected for unknown account, over-limit rejected, duplicate rejected, contribution default applied
- [ ] `go test ./internal/ledger/...` — MOVEMENT entry created with correct fields

### Manual Verification

- [ ] Ledger entry matches PRD spec for a valid deposit

## Tasks

- [001-ledger-schema](./tasks/001-ledger-schema.md)
- [002-ledger-posting](./tasks/002-ledger-posting.md)
- [003-funding-config](./tasks/003-funding-config.md)
- [004-funding-rules](./tasks/004-funding-rules.md)
- [005-contribution-defaults](./tasks/005-contribution-defaults.md)
- [006-session-account-eligibility](./tasks/006-session-account-eligibility.md)
