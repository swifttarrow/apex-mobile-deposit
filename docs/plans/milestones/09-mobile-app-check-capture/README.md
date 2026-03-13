# Milestone 9: Mobile App — Check Capture Simulation

## Overview

Lightweight mobile app (PWA or Expo) that simulates taking pictures of checks (front and back). User-supplied mock check images replace "captured" photos; app sends mock payload to the Go deposit API.

**Source:** [Gaps Plan §13: Mobile App — Check Capture Simulation](../../../thoughts/plans/2025-03-10-checkstream-full-deliverable.md#13-mobile-app--check-capture-simulation-new-facet)

## Dependencies

- [x] Milestone 4: Deposit Submission Flow (API accepts multipart with front/back images)
- [ ] 6 mocked checks (6 fronts + 6 backs) supplied by user

## Changes Required

- Mobile app scaffold (PWA in `cmd/server/web/mobile/` or separate Expo app)
- Camera flow UI: prompt for front photo, then back (no actual camera storage)
- Mock replacement: map "taken" image to one of 6 mocks; user selects scenario or round-robin
- Integration: POST to Go deposit endpoint with front image, back image, MICR, amount, account
- Document mock check assets: location, naming convention (e.g. `check-01-front.png`, `check-01-back.png`)

## Success Criteria

### Automated Verification

- [x] App builds; no runtime errors
- [x] Deposit submission returns 202/201 from Go service

### Manual Verification

- [ ] Flow: "Take front" → "Take back" → submit; mock images sent to API
- [ ] Can select which mock check (1–6) or scenario to use
- [ ] Deposit appears in system; status queryable
- [ ] Works on mobile viewport (responsive) or native app

## Tasks

- [001-mobile-app-scaffold](./tasks/001-mobile-app-scaffold.md)
- [002-mock-check-assets](./tasks/002-mock-check-assets.md)
- [003-camera-flow-mock-replacement](./tasks/003-camera-flow-mock-replacement.md)
- [004-deposit-api-integration](./tasks/004-deposit-api-integration.md)
