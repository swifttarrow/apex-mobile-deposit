# Task 002: Vendor Types

## Goal

Define request/response structs for Vendor API stub (IQA, MICR, OCR, duplicate, amount_mismatch, clean_pass).

## Deliverables

- [ ] `internal/vendor/types.go` with Request and Response structs
- [ ] Response includes `confidence` or `iqScore` for MICR/IQA scenarios (for operator queue)
- [ ] JSON tags for all fields

## Notes

- `clean_pass` returns `micr`, `amount`, `transactionId`
- `amount_mismatch` returns `ocrAmount`, `enteredAmount`
- `iqapass` returns `iqScore`

## Verification

- Structs marshal/unmarshal to expected JSON shapes
