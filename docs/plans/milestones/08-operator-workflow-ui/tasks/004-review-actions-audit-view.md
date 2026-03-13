# Task 004: Review Actions & Audit View

## Goal

Wire Approve/Reject buttons to API; show optional contribution override; display audit history per deposit.

## Deliverables

- [x] Approve button → POST /operator/approve with transfer_id; optional contribution_type
- [x] Reject button → POST /operator/reject with transfer_id
- [x] Contribution override UI (dropdown or input) when approving
- [x] Audit view: show operator_actions for selected deposit (action, operator_id, timestamp)
- [x] Success/error feedback after approve/reject
- [x] Queue refreshes or item removed after action

## Notes

- Audit data: may need GET /operator/deposits/:id/audit or include actions in queue payload
- Operator ID can be stub (e.g. "operator-1") if auth deferred

## Verification

- Approve a flagged deposit → state moves to FundsPosted; item leaves queue
- Reject → state moves to Rejected; item leaves queue
- Audit view shows past actions for a deposit
