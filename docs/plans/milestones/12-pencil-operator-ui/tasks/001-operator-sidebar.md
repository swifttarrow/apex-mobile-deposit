# Task 001: Operator Sidebar

## Goal

Add left sidebar (~280px) with logo, nav items (Review Queue active), and profile footer per Pencil design.

## Deliverables

- [ ] Sidebar fixed left, ~280px width
- [ ] Sidebar header: logo / brand ("Checkstream")
- [ ] Nav section: "Review Queue" (active), optional placeholder items (e.g., "Settlement", "Reports")
- [ ] Nav item styling: pill/capsule for active; hover states
- [ ] Sidebar footer: profile area (name/avatar placeholder), dropdown icon
- [ ] Main content area: flex to fill remaining width; layout accommodates sidebar
- [ ] Link to `/scenarios` for Scenarios (if not in sidebar, ensure nav between Operator and Scenarios exists)
- [ ] Design tokens: `$--sidebar`, `$--sidebar-border`, `$--sidebar-accent`, `$--sidebar-foreground`

## Notes

- Location: `cmd/server/web/operator/index.html`
- Design ref: frame `S4MSh` sidebar (`zrZU3`), `fM3c1` sidebar (`KqcWy`)
- Existing operator page has top nav (Scenarios, Operator); can move Scenarios to sidebar or keep top
- Responsive: sidebar can collapse on narrow viewport (optional for this task)

## Verification

- Sidebar visible on left; Review Queue highlighted
- Main content (queue/table) is to the right of sidebar
- Profile footer at bottom of sidebar
