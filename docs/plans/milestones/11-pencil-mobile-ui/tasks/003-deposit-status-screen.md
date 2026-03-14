# Task 003: Deposit Status Screen

## Goal

Replace the result screen with a design-aligned Status screen: success/error alert, details card, and stepper showing state progression.

## Deliverables

- [ ] Success alert: green banner with check icon (when deposit submitted successfully)
- [ ] Error alert: red/muted styling when submission failed
- [ ] Details card: amount, account, status (state), transfer ID
- [ ] Stepper: map transfer state to steps: Requested → Validating → Analyzing → (Approved) → FundsPosted; show current step
- [ ] State-to-step mapping: Requested=1, Validating=2, Analyzing=3, Approved=4, FundsPosted=5; Rejected = show error state
- [ ] Navigate to Status tab after submit (with transfer ID in state)
- [ ] Status tab shows "Select a deposit" or last-viewed deposit when no selection

## Notes

- Design ref: frame `W6zG2` (Mobile - Deposit Status)
- Use `GET /deposits/:id` for status data when viewing specific deposit
- Stepper visual: horizontal steps with checkmarks for completed, highlight for current
- ManualReview state maps to Analyzing in stepper

## Verification

- After submit → Status tab shows alert, details, stepper
- Stepper reflects current state (e.g., FundsPosted = all steps complete)
- Failed submit shows error alert
