# Task 001: Design Tokens CSS Variables

## Goal

Define CSS variables in `:root` for mobile and operator pages, mapping Pencil design system tokens to usable values. Apply consistently across components.

## Deliverables

- [ ] Define `:root` variables in mobile and operator pages (or shared if possible):
  - `--background`, `--foreground`
  - `--card`, `--primary`, `--primary-foreground`
  - `--muted-foreground`, `--border`, `--input`
  - `--radius-pill`, `--radius-m`, `--radius-none`
  - `--sidebar`, `--sidebar-border`, `--sidebar-accent`, `--sidebar-foreground`
  - `--color-success`, `--font-primary`, `--font-secondary`
  - `--white` (if used)
- [ ] Align with existing dark theme (e.g., `--bg`, `--text`, `--accent`) or migrate to new names
- [ ] Apply to buttons, cards, inputs, alerts, tables
- [ ] Ensure contrast and accessibility (WCAG AA for text)

## Notes

- Pencil schema uses `$--*`; we use `--*` in CSS (no `$`)
- Existing operator/mobile use `--bg`, `--bg-card`, `--text`, `--accent`, etc. — either rename or map
- Reference: existing values in `cmd/server/web/operator/index.html`, `cmd/server/web/mobile/index.html`
- Optional: create `styles/tokens.css` or inline in each page

## Verification

- Inspect element; CSS variables resolve correctly
- Buttons, cards, inputs use token-based colors and radii
- No hardcoded hex values for primary/success/border (except in token defs)
