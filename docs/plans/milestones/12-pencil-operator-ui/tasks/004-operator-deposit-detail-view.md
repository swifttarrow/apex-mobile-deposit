# Task 004: Operator Deposit Detail View

## Goal

Create a dedicated Deposit Detail view with breadcrumb, two-column layout (main + right audit column), and action buttons. Enable deep link and back navigation.

## Deliverables

- [ ] Dedicated view: `/operator/detail?id=xxx` or hash `#detail?id=xxx` — enable deep link
- [ ] Breadcrumb: "← Review Queue / DEP-xxxxx" — link back to queue
- [ ] Two-column layout: left (main) = amounts, risk scores, images, approve/reject; right (~360px) = audit history
- [ ] Action buttons in top section: Approve, Reject
- [ ] Contribution override in actions area (existing)
- [ ] Audit history in right column (from `GET /operator/actions/:id`)
- [ ] On Approve/Reject success: redirect to queue or refresh; remove item from queue
- [ ] Server: add route for `/operator/detail` or `/operator/index.html` with hash — SPA-style routing in single HTML

## Notes

- Design ref: frame `fM3c1` (Operator - Deposit Detail)
- Right column: audit list; can add summary card
- Use hash routing: `#queue` (default), `#detail?id=xxx` — no server changes if single-page
- Or: `/operator` = queue, `/operator/detail.html?id=xxx` = detail — requires server to serve same HTML for both
- Simplest: single HTML; show queue or detail based on `location.hash` / `?id=`
- Existing detail content: amounts, risk, images, approve/reject, audit — reorganize into two columns

## Verification

- Navigate to Detail (row click or direct URL); breadcrumb shows
- Two columns: left = detail content; right = audit
- Approve/Reject work; redirect to queue on success
- Back link returns to queue
