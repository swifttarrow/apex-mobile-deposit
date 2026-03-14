# Task 002: Tab Bar and Account Selector

## Goal

Add bottom tab bar navigation (Capture, Status, History) and account select dropdown populated with test accounts.

## Deliverables

- [ ] Tab bar at bottom: 4 tabs (Capture active, Status, History, optional 4th e.g. Settings)
- [ ] Tab bar styling: pill container, active tab highlighted
- [ ] Clicking tab switches visible screen (Capture / Status / History)
- [ ] Account select dropdown: populate with test accounts (ACC-001, ACC-IQA-BLUR, ACC-IQA-GLARE, ACC-MICR-FAIL, ACC-DUP-001, ACC-MISMATCH, ACC-OVER-LIMIT, ACC-RETIRE-001)
- [ ] Selected account stored in state; used when submitting deposit
- [ ] API base URL: retain settings (can live in Settings tab or header) for /mobile context

## Notes

- Design ref: tab bar in frames `ayn1v`, `W6zG2`, `Vpd6e` (pill with 4 segments)
- Use sessionStorage or in-memory state for selected account across tab switches
- Capture tab = single-screen capture; Status = last submitted or selected deposit; History = list

## Verification

- Tab bar visible; switching tabs shows correct screen
- Account selector shows test accounts; selection persists for submit
- Submit uses selected account
