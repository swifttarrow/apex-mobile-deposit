# Task 004: Approve/Reject Endpoints

## Goal

Implement POST /operator/approve and POST /operator/reject with state transitions and audit logging.

## Deliverables

- [ ] `POST /operator/approve` — body: {transfer_id, operator_id}; transition Analyzing → Approved; post ledger; log action
- [ ] `POST /operator/reject` — body: {transfer_id, operator_id}; transition Analyzing → Rejected; log action
- [ ] Every approve/reject writes to operator_actions

## Notes

- Approve triggers ledger posting and FundsPosted

## Verification

- Approve → FundsPosted, ledger entry; Reject → Rejected; both logged
