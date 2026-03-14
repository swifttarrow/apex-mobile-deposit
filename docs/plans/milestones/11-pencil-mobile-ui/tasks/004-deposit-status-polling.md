# Task 004: Deposit Status Polling

## Goal

When a deposit returns 202 (flagged for review), poll `GET /deposits/:id` until terminal state and update the stepper in real time.

## Deliverables

- [ ] After 202 response: store transfer ID; start polling `GET /deposits/:id` every 2–3 seconds
- [ ] On each response: update details card and stepper with latest state
- [ ] Stop polling when state is terminal (FundsPosted, Rejected, etc.)
- [ ] Show loading/updating indicator during poll (optional)
- [ ] Handle polling errors gracefully; retry or show message

## Notes

- Terminal states: FundsPosted, Rejected, Completed, Returned, SettlementIssue
- Polling interval: 2–3 seconds; consider exponential backoff on error
- Clear poll timer when user navigates away or component unmounts (if using framework)

## Verification

- Submit with ACC-MICR-FAIL (202) → Status shows "Flagged"; stepper updates when operator approves/rejects
- Submit with ACC-001 (201) → no polling needed; stepper shows completed
- Polling stops when state reaches FundsPosted or Rejected
