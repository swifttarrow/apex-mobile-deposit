# Task 004: Deposit API Integration

## Goal

Send mock front/back images, amount, account to Go deposit API. Match web flow payload.

## Deliverables

- [ ] POST to deposit endpoint (e.g. POST /deposits) with multipart/form-data
- [ ] Payload: front_image, back_image, amount, account_id; optional MICR if mock provides it
- [ ] API base URL configurable (same origin if PWA, or env for Expo)
- [ ] Success: show deposit ID, link to status; error handling
- [ ] Source field: include `source: mobile` (or similar) if API supports it for logging

## Notes

- Reuse same deposit handler as web scenarios; ensure multipart parsing accepts images
- Amount/account: user input or derived from mock scenario (e.g. mock 1 = clean_pass with $150)

## Verification

- Submit from mobile app; deposit appears in system; GET /deposits/:id returns it
- Go logs show source=mobile if implemented
