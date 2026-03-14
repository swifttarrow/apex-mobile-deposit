# Checkstream MVP → Full Deliverable: Gap Analysis

**Purpose:** Identify what the current MVP plan (`2025-03-10-checkstream-mvp.md`) does not cover relative to the PRD (`docs/prd.md`) and requirements (`docs/requirements.md`) so the full deliverable is complete.

**Reference:** PRD MVP gate (24 hrs), Early Submission (4 days), Final (Sunday).

---

## 1. Funding Service Gaps


| Gap                       | PRD/Requirements                                                               | Plan Coverage | Action                                                                                        |
| ------------------------- | ------------------------------------------------------------------------------ | ------------- | --------------------------------------------------------------------------------------------- |
| **Contribution defaults** | "Contribution type defaults" (individual for retirement-type accounts)         | Not mentioned | Add to Phase 3: default contribution type when account type is retirement; document in config |
| **Contribution override** | "Ability to override contribution type defaults if needed" (Operator Workflow) | Not mentioned | Add to Phase 5: optional `contribution_type` in approve payload; store in transfer/ledger     |


---

## 2. Operator Workflow Gaps


| Gap                               | PRD/Requirements                                           | Plan Coverage                           | Action                                                                          |
| --------------------------------- | ---------------------------------------------------------- | --------------------------------------- | ------------------------------------------------------------------------------- |
| **Risk scores**                   | "Risk scores", "risk indicators and Vendor Service scores" | Plan has images, MICR, amounts only     | Add: `risk_score` or `iq_score` from Vendor response; display in queue response |
| **MICR confidence scores**        | "MICR data and confidence scores"                          | Plan has `micr_data` (JSON)             | Ensure Vendor stub returns confidence; include in queue payload                 |
| **Recognized vs. entered amount** | "Recognized vs. entered amount comparison"                 | Plan has `ocr_amount`, `entered_amount` | Already in schema; ensure queue API returns both for comparison display         |


---

## 3. Return/Reversal Gaps


| Gap                       | PRD/Requirements              | Plan Coverage | Action                                                                                    |
| ------------------------- | ----------------------------- | ------------- | ----------------------------------------------------------------------------------------- |
| **Investor notification** | "Notifies investor" on return | Not mentioned | Add: stub notification (log entry, optional webhook/event); document as simulated for MVP |


---

## 4. Settlement Gaps


| Gap                                     | PRD/Requirements                          | Plan Coverage                                     | Action                                                                                                   |
| --------------------------------------- | ----------------------------------------- | ------------------------------------------------- | -------------------------------------------------------------------------------------------------------- |
| **Settlement Bank acknowledgment**      | "Settlement Bank acknowledgment tracking" | Plan: "optional stub ack" in What We're NOT Doing | Add: minimal ack tracking (e.g. `settlement_ack_at` on transfer when file written; or stub ack endpoint) |
| **Settlement confirmation → Completed** | FundsPosted → Completed on settlement     | Plan says "Transition FundsPosted → Completed"    | Covered; verify trigger is explicit (EOD batch run)                                                      |


---

## 5. Observability & Logging Gaps


| Gap                                    | PRD/Requirements                                                                   | Plan Coverage                         | Action                                                                           |
| -------------------------------------- | ---------------------------------------------------------------------------------- | ------------------------------------- | -------------------------------------------------------------------------------- |
| **Per-deposit decision trace**         | "Inputs → Vendor response → business rules → operator actions → settlement status" | Not mentioned                         | Add: structured log per deposit (or `deposit_traces` table) for debugging        |
| **Differentiate deposit sources**      | "Differentiate between deposit sources in logs"                                    | Not mentioned                         | Add: `source` field (e.g. mobile, api) on transfer; include in logs              |
| **Monitor missing/delayed settlement** | "Monitor for missing or delayed settlement files"                                  | Not in scope (observability deferred) | Document as follow-up; optional: simple health check for "unsettled FundsPosted" |


---

## 6. Demo UI / CLI Gaps


| Gap                             | PRD/Requirements                                                       | Plan Coverage             | Action                                                                                    |
| ------------------------------- | ---------------------------------------------------------------------- | ------------------------- | ----------------------------------------------------------------------------------------- |
| **Cap-table / ledger view**     | "Cap-table / ledger view showing account balances and posted deposits" | Plan: "Minimal UI or CLI" | Add: GET /ledger or /accounts/:id/balance; CLI or UI to show balances and posted deposits |
| **Transfer status tracking UI** | "Transfer status tracking through all states"                          | GET /deposits/:id exists  | Ensure response includes full state history or at least current state; document in demo   |


---

## 7. Performance Targets (Not in Plan)


| Gap                        | PRD/Requirements                                        | Plan Coverage | Action                                                                      |
| -------------------------- | ------------------------------------------------------- | ------------- | --------------------------------------------------------------------------- |
| **Vendor stub < 1s**       | Vendor stub response < 1 second                         | Not tested    | Add assertion in integration test or document as non-functional requirement |
| **Ledger posting < 5s**    | Ledger posting latency < 5 seconds from approval        | Not tested    | Same                                                                        |
| **Settlement file < 5s**   | Settlement file generation < 5 seconds from EOD trigger | Not tested    | Same                                                                        |
| **Operator queue < 1s**    | Flagged items visible within 1 second                   | Not tested    | Same                                                                        |
| **State propagation < 1s** | State changes queryable within 1 second                 | Not tested    | Same                                                                        |


**Recommendation:** Document in README as targets; add simple latency assertions in critical-path tests if time permits.

---

## 8. Documentation Gaps


| Gap                           | PRD/Requirements                                                                                                                | Plan Coverage                     | Action                                                          |
| ----------------------------- | ------------------------------------------------------------------------------------------------------------------------------- | --------------------------------- | --------------------------------------------------------------- |
| **Risks & limitations**       | Required: "Risks & limitations" (no compliance claims)                                                                          | Not in Phase 7                    | Add: `docs/risks_limitations.md` or section in README           |
| **State machine diagram**     | Architecture: "state machine diagram"                                                                                           | Plan: "System diagram, data flow" | Ensure architecture doc includes explicit state machine diagram |
| **Operational Cost Analysis** | Required deliverable (dev + production projection)                                                                              | Not in plan                       | Add: `docs/operational_cost.md` with table and assumptions      |
| **Test report**               | `/reports` — test results and scenario coverage                                                                                 | Plan: `make test`                 | Add: `make test-report` or script to output to `reports/`       |
| **SUBMISSION.md format**      | README or SUBMISSION.md with: Project name, Summary, How to run, Test results, "With one more week", Risks, Evaluation guidance | Not specified                     | Add template to Phase 7                                         |


---

## 9. Submission & Final Deliverable Gaps


| Gap                     | PRD/Requirements                                                        | Plan Coverage                               | Action                                                                                                    |
| ----------------------- | ----------------------------------------------------------------------- | ------------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| **Demo Video**          | 3–5 min; happy path, stub scenarios, operator workflow, return handling | Plan: demo script only                      | Add: record demo video as Final deliverable                                                               |
| **Pre-Search document** | Required; saved AI conversation or distilled output                     | Research exists; not explicitly deliverable | Ensure `docs/research/checkstream-prd-research.md` is referenced in README and included in submission |
| **Social Post**         | Tag @GauntletAI                                                         | Not in plan                                 | Add to submission checklist                                                                               |


---

## 10. Testing Gaps


| Gap                                     | PRD/Requirements                                    | Plan Coverage                                    | Action                                                                   |
| --------------------------------------- | --------------------------------------------------- | ------------------------------------------------ | ------------------------------------------------------------------------ |
| **Contribution defaults test**          | "Business rule enforcement (contribution defaults)" | Plan tests limits, duplicates                    | Add test: contribution type default applied                              |
| **Invalid state transition tests**      | "State machine transitions (valid and invalid)"     | Plan: "Invalid transition rejected" in Phase 1   | Ensure explicit tests for invalid transitions (e.g. Rejected → Approved) |
| **Settlement file contents validation** | "Settlement file contents validation"               | Plan: "X9 file generated with correct structure" | Add: assert MICR, images, amounts present in generated file              |


---

## 11. Idempotency (Deferred but Documented)


| Gap                          | PRD/Requirements                                             | Plan Coverage                                   | Action                                                                                   |
| ---------------------------- | ------------------------------------------------------------ | ----------------------------------------------- | ---------------------------------------------------------------------------------------- |
| **Idempotency for deposits** | Interview prep: "How would you add idempotency?"             | Plan: "optional for MVP; document as follow-up" | Document in decision log; implement if time permits (Phase 4 has middleware placeholder) |
| **Idempotency for returns**  | "Idempotent: duplicate return for same transfer returns 200" | Plan Phase 6: covered                           | ✓                                                                                        |


---

## 12. Operator Workflow UI (New Facet)


| Component          | Description                                                                        | Action                                                                                                 |
| ------------------ | ---------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------ |
| **Operator page**  | Separate page for manual review, approval/rejection, and audit of flagged deposits | Add `/operator` route; list flagged deposits with risk scores, MICR confidence, OCR vs entered amounts |
| **Review actions** | Approve, reject, optionally override contribution type                             | Wire to existing approve/reject API; show audit trail                                                  |
| **Audit view**     | View decision history per deposit                                                  | Display deposit traces, operator actions, timestamps                                                   |
| **Navigation**     | Cross-page navigation                                                              | Add nav structure (e.g. header/sidebar) to move between `/scenarios` and `/operator`                   |


**Implementation notes:**

- Operator page lives alongside existing scenarios UI; share auth/session if applicable
- Ensure operator queue API (`GET /operator/queue` or equivalent) returns flagged items with full context
- Document as new phase or extend Phase 5 (Operator Queue) in MVP plan

---

## 13. Mobile App — Check Capture Simulation (New Facet)


| Component            | Description                                          | Action                                                                                                                |
| -------------------- | ---------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------- |
| **Mobile app**       | Simulates taking pictures of checks (front and back) | Build lightweight mobile app (React Native, Expo, or PWA)                                                             |
| **Camera flow**      | Capture front, then back of check                    | UI prompts for front photo, then back; no actual camera storage required                                              |
| **Mock replacement** | Replace captured image with one of 6 supplied mocks  | User supplies 6 mocked checks (each with front + back); app selects mock based on flow/choice and sends to Go service |
| **Integration**      | Send mock images to Go deposit API                   | Same payload as web flow: front image, back image, MICR, amount, etc.                                                 |


**Mock check set:**

- 6 mocked checks total (6 fronts + 6 backs)
- App maps "taken" picture to a mock (e.g. user selects which scenario, or round-robin)
- No persistence of actual camera capture; mock images are the payload

**Implementation notes:**

- PWA may suffice for "mobile" demo (responsive, runs on phone)
- Alternatively: Expo/React Native for native feel
- Reuse Go service deposit endpoint; ensure it accepts multipart with front/back images
- Document mock check assets location and naming convention

---

## Summary: Recommended Additions to MVP Plan

### Must-add for full deliverable

1. **Phase 3:** Contribution defaults (config-driven); document
2. **Phase 5:** Risk/confidence scores in queue; contribution override in approve
3. **Phase 6:** Stub investor notification (log) on return
4. **Phase 7:** `docs/risks_limitations.md`, `docs/operational_cost.md`, state machine diagram in architecture, SUBMISSION.md template, test report output
5. **Operator Workflow UI:** `/operator` page for review, approval/rejection, audit; nav between `/scenarios` and `/operator`
6. **Mobile App:** Check capture simulation (front + back); 6 mocked checks; send mock images to Go service
7. **Final:** Demo video, Pre-Search reference, Social post

### Nice-to-have (if time permits)

- Settlement Bank ack tracking (minimal)
- Per-deposit decision trace (structured log)
- Performance assertions in tests
- Cap-table / ledger view endpoint

### Out of scope (document only)

- Production observability
- Operator authentication
- Full idempotency for deposits

---

## Checklist: Full Deliverable vs MVP


| Category                                  | MVP (24h)   | Full (Sunday) |
| ----------------------------------------- | ----------- | ------------- |
| Core pipeline                             | ✓           | ✓             |
| Operator queue (risk scores, override)    | Partial     | ✓             |
| Operator Workflow UI (`/operator`, nav)   | —           | ✓             |
| Mobile app (check capture simulation)     | —           | ✓             |
| Return notification                       | Stub/log    | ✓             |
| Documentation (risks, cost, architecture) | Minimal     | Complete      |
| Demo video                                | —           | Required      |
| Test report                               | `make test` | `reports/`    |
| Deployed + public                         | ✓           | ✓             |
| Pre-Search, Social post                   | —           | Required      |


