# Task 001: Mobile App Scaffold

## Goal

Create the mobile app structure. PWA (served from Go) or separate Expo/React Native project.

## Deliverables

- [x] Choose approach: PWA under `cmd/server/web/mobile/` or `mobile/` Expo project
- [x] Minimal app shell: HTML/JS (PWA) or Expo entry (React Native)
- [x] Responsive layout for mobile viewport
- [x] Route or entry point for check capture flow

## Notes

- PWA: simpler, served with Go; add route `/mobile` similar to `/scenarios`
- Expo: better native feel; separate `mobile/` dir; `npm run start` for dev
- Document choice in README

## Verification

- App loads; blank or placeholder screen visible on mobile or in dev tools mobile emulation
