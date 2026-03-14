# Task 001: Deposit Capture Single-Screen Layout

## Goal

Refactor the mobile deposit flow from a multi-step wizard to a single-screen layout matching the Pencil design.

## Deliverables

- [ ] Replace wizard (Setup → Front → Back → Review → Result) with single scrollable page
- [ ] Status bar: time, connectivity icons (static/mock)
- [ ] Header: "Deposit Check" + back icon
- [ ] Front of Check capture area (tap → show mock image preview)
- [ ] Back of Check capture area (tap → show mock image preview)
- [ ] Deposit Amount input
- [ ] Submit Deposit button
- [ ] Retain capture flow: tap areas → show preview → enable submit when both captured
- [ ] Keep mock check scenario selection (or integrate into account/flow)

## Notes

- Location: `cmd/server/web/mobile/index.html`
- Design ref: frame `ayn1v` (Mobile - Deposit Capture)
- Use existing check images from `cmd/server/web/mobile/checks/`
- Styling: align with `$--background`, `$--card`, `$--primary`, `$--radius-pill` (or existing vars)
- Fonts: Geist/Inter per design

## Verification

- Open `/mobile`; single page shows all capture elements
- Tap front/back areas → previews appear; Submit enabled when both captured
- Submit still posts to API (existing logic)
