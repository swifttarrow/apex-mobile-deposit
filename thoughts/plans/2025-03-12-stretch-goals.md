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

## Stretch Goal 3: Pencil Design Implementation — Proposed

### Objective

Implement the mobile and operator UIs to match the Pencil designs in `pencil-new.pen`. This includes both visual alignment and any backend gaps required to power the UIs.

### Design Reference

| Frame | Name | Key UI Elements |
|-------|------|-----------------|
| `ayn1v` | Mobile - Deposit Capture | Status bar, header, front/back check capture areas, amount input, account select, Submit button, tab bar |
| `W6zG2` | Mobile - Deposit Status | Success alert, details card, stepper (status progression), tab bar |
| `Vpd6e` | Mobile - Deposit History | Search bar, filter button, list of deposit cards (scrollable), tab bar |
| `S4MSh` | Operator - Review Queue | Sidebar nav, stats row (3 cards), filter tabs, table (header, rows, pagination footer) |
| `fM3c1` | Operator - Deposit Detail | Breadcrumb (Review Queue / DEP-xxx), two-column layout (main + 360px right), action buttons |

---

### Phase 3.1: Backend Gaps (Power UIs)

Several APIs or schema changes are needed before the UIs can function as designed.

#### Gap 1: Deposit History API — Required for Mobile Deposit History

**Current state:** `GET /deposits/:id` exists. No endpoint to list deposits for an account.

**Required:** `GET /deposits?account_id=...` or `GET /accounts/:id/deposits`

| Item | Details |
|------|---------|
| Endpoint | `GET /deposits?account_id=ACC-001&limit=20&offset=0` or `GET /accounts/{id}/deposits` |
| Query params | `account_id` (required for mobile context), `limit`, `offset`, optional `search` (amount/date substring), `status` filter |
| Response | `{ transfers: [...], count: N, total?: N }` — transfers with id, amount, state, created_at |
| Repository | Add `ListTransfersByAccount(accountID string, limit, offset int, status string)` to `internal/transfer/repository.go` |

**Success criteria:**

- [ ] `GET /deposits?account_id=ACC-001` returns transfers for that account, ordered by `created_at DESC`
- [ ] Pagination params work; `count` reflects returned items

**Verification:**

- Automated: `go test ./internal/api/...` — add test for `GET /deposits?account_id=...`
- Manual: Call endpoint; verify response shape and ordering

---

#### Gap 2: Account List (Optional — for Account Selector)

**Current state:** No `GET /accounts` endpoint. Mobile design has account select dropdown.

**Options:**

- **A) Mock accounts:** Use hardcoded test accounts (ACC-001, ACC-IQA-BLUR, etc.) in the dropdown — no API change.
- **B) GET /accounts:** If multi-account is required, add `GET /accounts` returning `[{ id, display_name? }]` (from funding config or new table).

**Recommendation:** Option A for MVP; add Option B if product requires dynamic account list.

---

#### Gap 3: Operator Queue Stats

**Current state:** `GET /operator/queue` returns `transfers` and `count`. No aggregate stats.

**Design needs:** Stats row (e.g., "Pending", "Today", "Total") — typically counts.

**Options:**

- **A) Client-side:** Derive "Pending" from queue response `count`; "Today" by filtering `created_at` client-side.
- **B) Server-side:** Add `GET /operator/stats` or include `stats: { pending, today }` in queue response.

**Recommendation:** Option A initially; add Option B if stats need to reflect data beyond current queue.

---

#### Gap 4: Operator Queue Pagination

**Current state:** `GET /operator/queue` returns all flagged transfers. No `limit`/`offset`.

**Design needs:** Table with pagination footer.

**Required:** Add `limit` and `offset` (or `page`, `page_size`) to `GET /operator/queue`. Operator repo `ListFlaggedTransfers` already supports filters; extend signature for pagination.

---

### Phase 3.2: Mobile UI Implementation

**Location:** `cmd/server/web/mobile/`

#### 3.2.1 Mobile - Deposit Capture

**Current:** Multi-step wizard (Setup → Capture Front → Capture Back → Review → Result).

**Target (from design):** Single-screen layout with:

- Status bar (time, connectivity icons)
- Header: "Deposit Check" + back
- Front of Check capture area
- Back of Check capture area
- Deposit Amount input
- Account select dropdown (use test accounts for now)
- Submit Deposit button
- Tab bar (4 tabs — map to Capture, Status, History, Settings or similar)

**Implementation:**

| Task | Details |
|------|---------|
| Layout | Refactor to single scrollable page; retain capture flow (tap areas → show preview → submit) |
| Account selector | Populate from test accounts (ACC-001, etc.); store selection in state |
| Tab bar | Add bottom nav linking to Capture (active), Status, History |
| Styling | Align with Pencil variables: `$--background`, `$--card`, `$--primary`, `$--radius-pill`, Geist/Inter fonts |

**Success criteria:**

- [ ] Single-screen capture matches design structure
- [ ] Account select works; Submit calls `POST /deposits` with selected account
- [ ] Tab bar navigates between Capture, Status, History

**Verification:**

- Automated: `make test`
- Manual: Open `/mobile`; verify layout, capture flow, tab navigation, submit

---

#### 3.2.2 Mobile - Deposit Status

**Current:** Result screen after submit shows icon, title, transfer ID, JSON blob.

**Target (from design):**

- Success alert (green, check icon)
- Details card (amount, account, status, transfer ID)
- Stepper showing status progression (e.g., Submitted → Validating → Approved → Funds Posted)

**Implementation:**

| Task | Details |
|------|---------|
| Success/error alert | Replace icon+title with alert banner; style per design |
| Details card | Structured layout for amount, account, status, ID |
| Stepper | Map `state` to steps: Requested→Validating→Analyzing→(Approved)→FundsPosted; show current step |
| Polling | For 202 responses (flagged), poll `GET /deposits/:id` until terminal state; update stepper |
| Tab bar | Same as Capture |

**Success criteria:**

- [ ] Status screen shows alert + details card + stepper
- [ ] Stepper reflects current state; updates when polling detects change
- [ ] Navigate to Status after submit (or via tab)

**Verification:**

- Automated: `make test`
- Manual: Submit deposit; verify Status screen, stepper, polling for flagged deposits

---

#### 3.2.3 Mobile - Deposit History

**Current:** No deposit history screen.

**Target (from design):**

- Header "Deposits" + filter button
- Search bar
- Scrollable list of deposit cards (each: top row with key info, bottom row with metadata)

**Implementation:**

| Task | Details |
|------|---------|
| API integration | Call `GET /deposits?account_id=...` (from Phase 3.1) |
| Search | Client-side filter by amount, date, or ID; or pass `search` param when API supports it |
| Deposit cards | Card per transfer: amount, state, date, ID; tap → navigate to Status for that ID |
| Filter button | Optional: filter by status (e.g., Pending, Completed) |
| Tab bar | Same as Capture, Status |

**Dependency:** Phase 3.1 Gap 1 (Deposit History API) must be implemented first.

**Success criteria:**

- [ ] History screen fetches and displays deposits for selected account
- [ ] Cards show amount, state, date; tap opens Status for that deposit
- [ ] Search/filter works (client or server)

**Verification:**

- Automated: `make test`
- Manual: Open History tab; verify list, card tap → Status

---

### Phase 3.3: Operator UI Implementation

**Location:** `cmd/server/web/operator/`

#### 3.3.1 Operator - Review Queue

**Current:** Queue panel (cards) + detail panel; dark theme.

**Target (from design):**

- Sidebar: logo, nav items (Review Queue active), profile footer
- Main: top bar (title, actions), stats row (3 stat cards), filter tabs, table with header/rows/footer (pagination)

**Implementation:**

| Task | Details |
|------|---------|
| Sidebar | Add left sidebar (~280px); nav links: Review Queue, (others TBD); profile section |
| Stats row | 3 cards: Pending (queue count), Today (count created today), Total (optional) — derive from queue or new endpoint |
| Filter tabs | Map to existing filters (All, Pending, etc.); integrate with `GET /operator/queue` params |
| Table layout | Replace queue cards with table: columns (ID, Account, Amount, Status, Date); rows from queue |
| Pagination | Footer with prev/next; use `limit`/`offset` from Phase 3.1 Gap 4 |
| Detail navigation | Row click → navigate to Deposit Detail page (or expand inline) |

**Success criteria:**

- [ ] Sidebar, stats, filter tabs, table match design
- [ ] Table populated from `GET /operator/queue`; pagination works
- [ ] Row click opens Deposit Detail

**Verification:**

- Automated: `make test`
- Manual: Open `/operator`; verify sidebar, stats, table, pagination, row click

---

#### 3.3.2 Operator - Deposit Detail

**Current:** Inline detail panel with amounts, risk scores, images, approve/reject.

**Target (from design):**

- Breadcrumb: "← Review Queue / DEP-xxxxx"
- Two-column layout: main (full detail) + right column (~360px) for summary or audit
- Action buttons in top section

**Implementation:**

| Task | Details |
|------|---------|
| Dedicated route | `/operator/detail?id=xxx` or `/operator/:id` — enable deep link and back navigation |
| Breadcrumb | Link back to Review Queue; show transfer ID |
| Two columns | Left: amounts, risk, images, actions (current content); Right: audit history or summary card |
| Action buttons | Approve, Reject in top section; keep contribution override in actions area |

**Success criteria:**

- [ ] Deposit Detail is a distinct view with breadcrumb
- [ ] Two-column layout; right column shows audit history
- [ ] Approve/Reject work; redirect or refresh queue on success

**Verification:**

- Automated: `make test`
- Manual: Open Deposit Detail; verify layout, audit, approve/reject flow

---

### Phase 3.4: Design Tokens & Styling

**Pencil design system variables (from schema):**

- `$--background`, `$--foreground`, `$--card`, `$--primary`, `$--primary-foreground`
- `$--muted-foreground`, `$--border`, `$--input`, `$--radius-pill`, `$--radius-m`
- `$--sidebar`, `$--sidebar-border`, `$--sidebar-accent`, `$--sidebar-foreground`
- `$--color-success`, `$--font-primary`, `$--font-secondary`

**Tasks:**

- Extract values from Pencil design (or use existing dark theme as base)
- Define CSS variables in `:root` for mobile and operator pages
- Apply consistently across new components
- Use Geist/Inter fonts per design

---

### Recommended Implementation Order

1. **Phase 3.1** — Backend gaps (Deposit History API, Operator pagination)
2. **Phase 3.2** — Mobile UIs (Capture → Status → History)
3. **Phase 3.3** — Operator UIs (Review Queue → Deposit Detail)
4. **Phase 3.4** — Design tokens (can be done in parallel with 3.2/3.3)

---

### Out of Scope (for this stretch goal)

- Real camera capture (keep mock check images)
- Authentication / session (use test account IDs)
- GET /accounts (use mock list unless product requires it)
- Operator stats API (use client-side derivation initially)
- Responsive breakpoints beyond mobile (402px) and operator (1440px) as designed

---

## References

- Scenario Showcase: `cmd/server/web/scenarios/index.html`
- MVP Plan: `thoughts/plans/2025-03-10-checkstream-mvp.md`
- Architecture: `docs/architecture.md`
- User Stories: `docs/user-stories.md`
- Pencil designs: `pencil-new.pen` (Mobile Deposit Capture, Status, History; Operator Review Queue, Deposit Detail)
