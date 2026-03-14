# Milestone 12: Pencil Design — Operator UI

## Overview

Refactor the operator UI to match Pencil designs: sidebar nav, stats row, filter tabs, table layout with pagination, and a dedicated Deposit Detail view with breadcrumb and two-column layout.

**Source:** [Stretch Goals Plan § Phase 3.3](../../2025-03-12-stretch-goals.md#phase-33-operator-ui-implementation)

## Dependencies

- [ ] Milestone 10: Pencil Design Backend Gaps (Operator Queue Pagination)
- [x] Milestone 8: Operator Workflow UI
- [ ] Milestone 13: Design Tokens (can be done in parallel)

## Changes Required

- `cmd/server/web/operator/index.html` — Sidebar, stats, filter tabs, table, detail route
- Possibly split into multiple HTML files or use hash routing for `/operator` vs `/operator/detail?id=xxx`

## Success Criteria

### Automated Verification

- [ ] `make test` passes
- [ ] No console errors on operator page load

### Manual Verification

- [ ] Sidebar, stats row, filter tabs, table match design; table populated from queue; pagination works
- [ ] Row click opens Deposit Detail
- [ ] Deposit Detail has breadcrumb, two-column layout, audit in right column; Approve/Reject work
- [ ] Back from Detail returns to Review Queue

## Tasks

- [001-operator-sidebar](./tasks/001-operator-sidebar.md)
- [002-operator-stats-filter-tabs](./tasks/002-operator-stats-filter-tabs.md)
- [003-operator-table-pagination](./tasks/003-operator-table-pagination.md)
- [004-operator-deposit-detail-view](./tasks/004-operator-deposit-detail-view.md)
