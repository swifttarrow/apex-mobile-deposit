# Task 003: Operator Queue Endpoint

## Goal

Implement GET /operator/queue returning flagged deposits with images, MICR, amounts.

## Deliverables

- [ ] `GET /operator/queue?status=Analyzing&date=...&account=...&amount=...`
- [ ] Response includes: front_image, back_image, micr_data, ocr_amount, entered_amount
- [ ] Filter by status, date, account, amount

## Notes

- Ensure ocr_amount and entered_amount both returned for comparison display

## Verification

- Flagged deposit appears in queue with all required fields
