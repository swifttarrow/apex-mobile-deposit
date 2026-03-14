# Task 003: Operator Table and Pagination

## Goal

Replace queue cards with a table layout (columns: ID, Account, Amount, Status, Date) and add pagination footer. Row click opens Deposit Detail.

## Deliverables

- [ ] Table layout: header row (ID, Account, Amount, Status, Date)
- [ ] Data rows from `GET /operator/queue`; use `limit` and `offset` for pagination
- [ ] Pagination footer: "Prev" / "Next" or page numbers; show "Page X of Y"
- [ ] Fetch with `?limit=10&offset=0`; increment offset on Next
- [ ] Row click: navigate to Deposit Detail (hash route `#detail?id=xxx` or new path)
- [ ] Selected row highlight on hover
- [ ] Empty state: "No flagged deposits" when queue empty
- [ ] Retain existing queue data structure; adapt rendering to table

## Notes

- **Dependency:** Milestone 10 (Operator Queue Pagination) for limit/offset
- Design ref: `tableWrap`, `headerRow`, `row1`–`row5`, `footer` in frame `S4MSh`
- Table styling: `$--card`, `$--border`; striped or hover rows
- Pagination: use `response.total` for page count; `response.count` for current page size

## Verification

- Table shows queue data; columns align
- Pagination: Next/Prev fetches new page; URL or state updates
- Row click opens Detail view with correct transfer
