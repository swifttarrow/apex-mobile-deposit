# Task 001: Mobile App Scaffold

## Goal

Create the mobile app structure. PWA (served from Go) or separate Expo/React Native project.

## Deliverables

- [ ] Choose approach: PWA under `cmd/server/web/mobile/` or `mobile/` Expo project
- [ ] Minimal app shell: HTML/JS (PWA) or Expo entry (React Native)
- [ ] Responsive layout for mobile viewport
- [ ] Route or entry point for check capture flow

## Notes

- PWA: simpler, served with Go; add route `/mobile` similar to `/scenarios`
- Expo: better native feel; separate `mobile/` dir; `npm run start` for dev
- Document choice in README

## Verification

- App loads; blank or placeholder screen visible on mobile or in dev tools mobile emulation
