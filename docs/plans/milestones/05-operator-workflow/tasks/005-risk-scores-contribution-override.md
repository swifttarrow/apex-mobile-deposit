# Task 005: Risk Scores & Contribution Override

## Goal

Add risk_score/iq_score to queue response and optional contribution_type override in approve payload.

## Deliverables

- [ ] Queue response includes risk_score or iq_score from vendor_response
- [ ] Approve payload accepts optional `contribution_type`; store in transfer/ledger when provided
- [ ] MICR confidence in queue payload (from Vendor stub)

## Notes

- From gaps: "Add risk_score or iq_score from Vendor response; display in queue response"
- "Ability to override contribution type defaults if needed"

## Verification

- Queue returns risk/confidence; approve with contribution_type override stores it
