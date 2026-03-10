# Task 001: Deposits Handler

## Goal

Create POST /deposits handler that parses request (images, amount, account_id).

## Deliverables

- [ ] `internal/api/deposits.go` — handler skeleton
- [ ] Parse JSON body: front_image, back_image (or paths), amount, account_id
- [ ] Validate input (Zod or manual validation)
- [ ] Return 400 on invalid input

## Notes

- Images: base64 or file path per plan

## Verification

- Handler parses valid request; returns 400 for invalid
