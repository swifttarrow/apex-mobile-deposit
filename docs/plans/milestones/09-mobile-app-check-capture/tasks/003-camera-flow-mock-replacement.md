# Task 003: Camera Flow & Mock Replacement

## Goal

Implement the capture flow: prompt for front, then back. Replace "taken" image with selected mock.

## Deliverables

- [x] Step 1: "Capture front" — UI shows camera placeholder or "Take front photo" button
- [x] Step 2: "Capture back" — same for back
- [x] No actual camera/camera API; on tap, assign mock image for that step
- [x] User selects which mock check (1–6) before or during flow; or round-robin
- [x] Preview of selected mock (front/back) before submit
- [x] State: front_selected, back_selected, ready to submit

## Notes

- Camera permission not required; purely simulated
- Mock selection: dropdown, scenario picker, or "Use check #N" control

## Verification

- Complete flow: select mock 1 → "take front" → "take back" → see previews; ready to submit
