# Task 002: Orchestration Flow

## Goal

Implement full deposit flow: create transfer → Vendor stub → Funding → ledger + state updates.

## Deliverables

- [ ] Create transfer in Requested
- [ ] Call Vendor stub → get response
- [ ] If fail/reject → transition to Rejected, return
- [ ] If flagged → transition to Analyzing, return
- [ ] If pass → call Funding (limits, duplicates)
- [ ] If Funding rejects → Rejected
- [ ] If Funding approves → Approved, post ledger, FundsPosted
- [ ] Return transfer with state
- [ ] Use transaction for multi-step writes (ledger + transfer state)

## Notes

- Wire Vendor, Funding, ledger in main.go

## Verification

- Happy path: clean pass, under limit → FundsPosted, ledger entry
