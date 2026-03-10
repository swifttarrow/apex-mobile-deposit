# Milestone 5: Operator Workflow

## Overview

Review queue for flagged deposits. Approve/reject with audit logging. Search/filter by date, status, account, amount. Include risk scores, MICR confidence, amount comparison, and contribution override (full deliverable).

**Source:** [MVP Plan Phase 5](../../../thoughts/plans/2025-03-10-checkstream-mvp.md#phase-5-operator-workflow) | [Gaps: Risk scores, contribution override](../../../thoughts/plans/2025-03-10-checkstream-mvp-to-full-deliverable.md#2-operator-workflow-gaps)

## Dependencies

- [ ] Milestone 1: Project Setup & Transfer State Machine
- [ ] Milestone 4: Deposit Submission Flow

## Changes Required

- `internal/api/operator.go` — GET /operator/queue, POST /operator/approve, POST /operator/reject
- `internal/operator/repository.go` — query flagged transfers; record operator_actions
- `operator_actions` table: transfer_id, action, operator_id, created_at
- Queue response: images, MICR, amounts, risk_score/iq_score, ocr_amount vs entered_amount
- Approve payload: optional `contribution_type` override

## Success Criteria

### Automated Verification

- [ ] `go test ./internal/api/...` — MICR fail deposit flagged; operator approve → ledger posted
- [ ] Operator actions logged

### Manual Verification

- [ ] Flagged deposit appears in queue with images, MICR, amount comparison, risk scores
- [ ] Approve → FundsPosted; Reject → Rejected
- [ ] Contribution override applied when provided

## Tasks

- [001-operator-actions-schema](./tasks/001-operator-actions-schema.md)
- [002-operator-repository](./tasks/002-operator-repository.md)
- [003-operator-queue-endpoint](./tasks/003-operator-queue-endpoint.md)
- [004-approve-reject-endpoints](./tasks/004-approve-reject-endpoints.md)
- [005-risk-scores-contribution-override](./tasks/005-risk-scores-contribution-override.md)
- [006-operator-tests](./tasks/006-operator-tests.md)
