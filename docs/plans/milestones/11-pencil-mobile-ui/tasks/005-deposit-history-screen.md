# Task 005: Deposit History Screen

## Goal

Add Deposit History screen: fetch deposits via `GET /deposits?account_id=...`, display as scrollable cards, support search/filter and tap-to-Status.

## Deliverables

- [ ] Header: "Deposits" + filter button
- [ ] Search bar: client-side filter by amount, date, or ID (or pass search param when API supports)
- [ ] Fetch `GET /deposits?account_id={selected}` on History tab load
- [ ] Deposit cards: each card shows amount, state, date, ID; top row (key info) + bottom row (metadata)
- [ ] Scrollable list of cards
- [ ] Tap card → set selected deposit; switch to Status tab and load that deposit
- [ ] Filter button: optional filter by status (All, Pending, Completed) — client-side or query param
- [ ] Empty state: "No deposits" when list is empty
- [ ] Use account from account selector (same as Capture)

## Notes

- **Dependency:** Milestone 10 (Deposit History API) must be done first
- Design ref: frame `Vpd6e` (Mobile - Deposit History)
- Card layout: two-row structure per design (dep1top, dep1bot)
- Status filter: e.g., Pending = Analyzing, Completed = FundsPosted|Completed|Rejected

## Verification

- History tab fetches and displays deposits for selected account
- Cards show amount, state, date; tap opens Status for that deposit
- Search/filter works (client-side as fallback)
