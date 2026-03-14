# Task 002: Operator Stats Row and Filter Tabs

## Goal

Add stats row (3 stat cards) and filter tabs above the queue table. Stats derived client-side from queue response; filters integrate with `GET /operator/queue` params.

## Deliverables

- [ ] Stats row: 3 cards — e.g., "Pending" (queue count), "Today" (created today), "Total" (optional)
- [ ] Stats derived client-side: Pending = `response.count`; Today = filter transfers by `created_at` date
- [ ] Filter tabs: "All", "Pending", or similar; map to query params (e.g., status filter)
- [ ] Tab styling: pill container; active tab highlighted
- [ ] Integrate with existing filters (date, account) — can be in filter bar or modal
- [ ] Top bar: title ("Review Queue" or similar), action buttons if needed
- [ ] API bar (API base URL, status) — retain for dev; can move to settings or keep

## Notes

- Design ref: `statsRow` (stat1, stat2, stat3), `filterRow` (tab1–tab4) in frame `S4MSh`
- Stats: Pending = transfers in queue; Today = count where created_at is today (client filter)
- Optional: add `GET /operator/stats` later if server-side stats needed
- Existing filters: date, account — keep in header or collapsible

## Verification

- Stats row shows Pending count; Today count updates based on queue data
- Filter tabs change visible data (or query params) when clicked
- Layout matches design
