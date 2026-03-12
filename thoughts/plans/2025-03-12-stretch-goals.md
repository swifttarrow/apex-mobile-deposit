# Checkstream Stretch Goals

## Overview

Post-MVP enhancements for the Scenario Showcase and observability. This plan documents what is **already implemented** (UI workflow diagram) and what is **proposed** (audit trail).

---

## Stretch Goal 1: UI Workflow Diagram — Implemented ✓

### Summary

The Scenario Showcase at `/scenarios/` includes an interactive Mermaid state diagram that visualizes the transfer lifecycle and animates the path taken when a scenario runs.

### Implementation Details

**Location:** `cmd/server/web/scenarios/index.html`

**Components:**

| Component | Description |
|-----------|-------------|
| **Mermaid diagram** | `stateDiagram-v2` with all transfer states and transitions |
| **Diagram pane** | ~60% width, left side; dark theme (GitHub-style palette) |
| **Accordion pane** | ~380px right side; scenario list with Run/Clear buttons |
| **Animation engine** | `window.__flowDiagram.animatePath(segments, isFail)` — segments are `[from, to]` pairs |

**States and transitions (from diagram source):**

```
[*] → Requested (Submit)
Requested → Validating (Send to Vendor)
Validating → Rejected (IQA Fail | Duplicate)
Validating → ManualReview (MICR/Amount mismatch)
Validating → Analyzing (Clean pass)
ManualReview → Rejected | Analyzing
Analyzing → Rejected (Rules fail) | Approved (Pass)
Approved → FundsPosted (Ledger)
FundsPosted → SettlementQueued (EOD batch) | Returned (Return)
SettlementQueued → Settling → Completed | SettlementIssue
Completed → Returned (Return + fee)
```

**Visual feedback:**

- **flow-current** — Blue stroke, dashed animation (currently animating)
- **flow-traversed** — Green stroke (success path)
- **flow-traversed-fail** — Red stroke (failure path)

**Controls:**

- Speed slider: 200ms–2000ms delay between segments (default 1000ms)
- Pause/Resume: Halts animation mid-sequence

**Scenario integration:**

- Each scenario defines a `path` array: `[['Requested','Validating'], ['Validating','Analyzing'], ...]`
- On Run, `runScenario()` calls `animatePath(result.path, !result.ok)`
- `pathFromResp(j)` infers path from response for failure cases
- Multi-step scenarios (e.g. US-1.4a MICR approve) define full path through ManualReview → Analyzing → Approved → FundsPosted

### Runnable Scenarios (with diagram paths)

| ID | Scenario | Path summary |
|----|----------|--------------|
| US-1.1 | Happy path | Requested → Validating → Analyzing → Approved → FundsPosted |
| US-1.2 | IQA blur | Requested → Validating → Rejected |
| US-1.3 | IQA glare | Requested → Validating → Rejected |
| US-1.4a | MICR fail → approve | … → ManualReview → Analyzing → Approved → FundsPosted |
| US-1.4b | MICR fail → reject | … → ManualReview → Rejected |
| US-1.5 | Duplicate | Requested → Validating → Rejected |
| US-1.6 | Amount mismatch | Requested → Validating → ManualReview |
| US-2.1 | Over-limit | … → Analyzing → Rejected |
| US-2.2 | Under-limit | … → FundsPosted |
| US-6.1 | Return | … → FundsPosted → Returned |
| US-5.1 | Settlement | FundsPosted → SettlementQueued → Settling → Completed |
| US-Idem | Idempotency | Same as happy path |

---

## Stretch Goal 2: Audit Trail — Proposed

### Objective

When a scenario is triggered, capture an audit trail of all events (HTTP requests and responses) with full request and response payloads, and present them in a usable format.

### Options (in order of effort)

#### Option A: Client-Side Audit (Recommended First)

**Approach:** The scenario runner already has access to every `fetch()` call. Refactor `run()` to record each call as an event and return an `events` array alongside `ok`, `status`, `body`, `msg`, `path`.

**Effort:** Low (~1–2 hours)

**Changes:**

| Area | Change |
|------|--------|
| Scenario `run()` functions | Use a shared `auditedFetch(base, method, path, body, opts)` helper that records `{ method, path, request, response, status }` and returns response |
| Result shape | `{ ok, status, body, msg, path, events }` — `events` is `[{ step, method, path, request, response }]` |
| Output UI | Replace single JSON blob with timeline: "1. POST /deposits → 201" (expandable request/response) |

**Success criteria:**

- [ ] Each scenario run displays a numbered event list
- [ ] Each event shows method, path, status; expandable request/response bodies
- [ ] Multi-step scenarios (US-1.4a, US-6.1, US-Idem) show 2+ events in order

---

#### Option B: Server-Side Audit via Correlation ID

**Approach:** Middleware records all requests and responses when `X-Audit-Correlation-ID` header is present. Scenario UI generates a correlation ID per run, sends it on each request, then optionally fetches `GET /audit/trail?correlation_id=...` to display server-side trail.

**Effort:** Medium (~2–4 hours)

**Changes:**

| Area | Change |
|------|--------|
| Schema | New `audit_events` table: `id`, `correlation_id`, `seq`, `method`, `path`, `request_body`, `response_body`, `status_code`, `created_at` |
| Middleware | Wrap handlers; when header present, tee request body (buffer/restore for downstream), capture response via `httptest.ResponseRecorder` pattern |
| Endpoint | `GET /audit/trail?correlation_id=...` returns JSON array of events |
| Scenario UI | Add `X-Audit-Correlation-ID: run-{timestamp}-{random}` to each fetch; optional "View server audit" link |

**Risks:**

- Request body read-once: must wrap `r.Body` with `io.NopCloser(bytes.NewBuffer(teeBytes))` so downstream can still read
- Response capture: wrap `ResponseWriter` to capture status + body

**Success criteria:**

- [ ] Correlation ID in header triggers audit logging
- [ ] `GET /audit/trail?correlation_id=...` returns events in order
- [ ] Scenario UI can display server-side trail (optional enhancement to Option A)

---

#### Option C: Internal Event Capture

**Approach:** Emit events for internal steps (vendor call, funding checks, ledger post, state transitions) in addition to HTTP. Requires instrumentation in `deposits.go` and related services.

**Effort:** Medium–High (~4–8 hours)

**Changes:**

- Context or event collector passed through handler → vendor → funding → ledger
- Event types: `transfer_created`, `vendor_called`, `vendor_response`, `funding_check`, `ledger_posted`, `state_transition`
- Persist or attach to response/audit trail

**Use case:** When debugging, see not just "POST /deposits returned 422" but "vendor returned pass → funding CheckLimit failed → Rejected."

---

### Recommendation

1. **Implement Option A** first — client-side audit in the Scenario UI. No server changes; fast to ship; gives full request/response trail for scenario runs.
2. **Add Option B** if server-side persistence or debugging outside the UI is needed (e.g. support tickets, compliance).
3. **Defer Option C** unless internal step tracing becomes a requirement.

---

## References

- Scenario Showcase: `cmd/server/web/scenarios/index.html`
- MVP Plan: `thoughts/plans/2025-03-10-checkstream-mvp.md`
- Architecture: `docs/architecture.md`
- User Stories: `docs/user-stories.md`
