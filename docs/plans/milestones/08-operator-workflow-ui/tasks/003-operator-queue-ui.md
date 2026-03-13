# Task 003: Operator Queue UI

## Goal

Display flagged deposits from GET /operator/queue with risk scores, MICR confidence, OCR vs entered amounts.

## Deliverables

- [ ] Operator page fetches `GET /operator/queue` (or equivalent)
- [ ] List/cards show: transfer id, account, amount, risk_score/iq_score, MICR confidence
- [ ] OCR amount vs entered amount comparison visible
- [ ] Check images (front/back) displayable (thumbnails or expandable)
- [ ] Loading and error states handled

## Notes

- API base URL configurable (e.g. from same origin or env)
- Queue response shape: see Milestone 5; ensure risk_score, ocr_amount, entered_amount present

## Verification

- Submit a flagged deposit (e.g. MICR fail); it appears in operator queue UI
- Risk scores and amount comparison display correctly
