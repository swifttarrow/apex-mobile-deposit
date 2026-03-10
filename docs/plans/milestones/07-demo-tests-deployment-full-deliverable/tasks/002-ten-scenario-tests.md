# Task 002: Ten Scenario Tests

## Goal

Add integration tests for all 10 PRD testing scenarios.

## Deliverables

- [ ] 1. Happy path end-to-end
- [ ] 2. IQA fail blur
- [ ] 3. IQA fail glare
- [ ] 4. MICR fail → operator approve → ledger
- [ ] 5. Duplicate rejected
- [ ] 6. Amount mismatch → operator approve/reject
- [ ] 7. Over-limit rejected
- [ ] 8. Return/reversal with fee
- [ ] 9. EOD cutoff
- [ ] 10. Stub configurability (different inputs → different responses)
- [ ] Invalid state transition tests (e.g. Rejected → Approved)
- [ ] Contribution defaults test

## Notes

- From gaps: contribution defaults test, invalid transition tests

## Verification

```bash
make test
```
