# Milestone 8: Operator Workflow UI

## Overview

Separate `/operator` page for manual review, approval/rejection, and audit of flagged deposits. Shared navigation between `/scenarios` and `/operator` so users can move between pages.

**Source:** [Gaps Plan §12: Operator Workflow UI](../../../thoughts/plans/2025-03-10-checkstream-full-deliverable.md#12-operator-workflow-ui-new-facet)

## Dependencies

- [x] Milestone 5: Operator Workflow (queue API, approve/reject endpoints)

## Changes Required

- `cmd/server/web/operator/` — operator page (index.html or similar)
- `cmd/server/main.go` — serve `/operator` route (or `/operator/`)
- Navigation component — header/sidebar with links to `/scenarios` and `/operator`
- Operator page: list flagged deposits, risk scores, MICR confidence, OCR vs entered amounts
- Review actions: Approve, Reject, optional contribution override; wire to existing API
- Audit view: decision history per deposit (deposit traces, operator actions, timestamps)
- Add nav to existing `/scenarios` page so both pages share navigation

## Success Criteria

### Automated Verification

- [x] Server serves `/operator` and `/scenarios` without error
- [x] Navigation links resolve correctly

### Manual Verification

- [ ] Nav appears on both `/scenarios` and `/operator`; clicking switches pages
- [ ] Operator page shows flagged deposits from queue API with risk scores, MICR, amount comparison
- [ ] Approve/Reject actions work; contribution override applies when provided
- [ ] Audit view shows decision history per deposit

## Tasks

- [001-operator-page-route](./tasks/001-operator-page-route.md)
- [002-navigation-structure](./tasks/002-navigation-structure.md)
- [003-operator-queue-ui](./tasks/003-operator-queue-ui.md)
- [004-review-actions-audit-view](./tasks/004-review-actions-audit-view.md)
