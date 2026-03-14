# Milestone 11: Pencil Design — Mobile UI

## Overview

Refactor the mobile app to match Pencil designs: single-screen Deposit Capture, redesigned Deposit Status with stepper, and new Deposit History screen. Includes tab bar navigation across Capture, Status, and History.

**Source:** [Stretch Goals Plan § Phase 3.2](../../2025-03-12-stretch-goals.md#phase-32-mobile-ui-implementation)

## Dependencies

- [ ] Milestone 10: Pencil Design Backend Gaps (Deposit History API)
- [x] Milestone 9: Mobile App Check Capture
- [ ] Milestone 13: Design Tokens (can be done in parallel)

## Changes Required

- `cmd/server/web/mobile/index.html` — Refactor layout, add tab bar, Status screen, History screen
- Design tokens from Pencil: `$--background`, `$--card`, `$--primary`, `$--radius-pill`, Geist/Inter fonts

## Success Criteria

### Automated Verification

- [ ] `make test` passes
- [ ] No console errors on page load

### Manual Verification

- [ ] Single-screen capture matches design structure; account select works; Submit calls POST /deposits
- [ ] Tab bar navigates between Capture, Status, History
- [ ] Status screen shows alert, details card, stepper; stepper updates when polling; navigate after submit
- [ ] History screen fetches and displays deposits; cards show amount, state, date; tap opens Status

## Tasks

- [001-deposit-capture-single-screen](./tasks/001-deposit-capture-single-screen.md)
- [002-tab-bar-account-selector](./tasks/002-tab-bar-account-selector.md)
- [003-deposit-status-screen](./tasks/003-deposit-status-screen.md)
- [004-deposit-status-polling](./tasks/004-deposit-status-polling.md)
- [005-deposit-history-screen](./tasks/005-deposit-history-screen.md)
