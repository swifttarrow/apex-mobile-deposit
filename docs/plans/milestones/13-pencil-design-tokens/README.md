# Milestone 13: Pencil Design Tokens & Styling

## Overview

Extract design tokens from the Pencil design system and apply them consistently across mobile and operator UIs. Use Geist/Inter fonts per design.

**Source:** [Stretch Goals Plan § Phase 3.4](../../../thoughts/plans/2025-03-12-stretch-goals.md#phase-34-design-tokens--styling)

## Dependencies

- Can be done in parallel with Milestones 11 and 12

## Changes Required

- Mobile and Operator HTML/CSS: define `:root` CSS variables
- Font imports: Geist, Inter from Google Fonts or similar
- Apply variables consistently to new and existing components
- Document token values for future reference (optional: tokens doc)

## Success Criteria

### Automated Verification

- [ ] No CSS/HTML lint errors
- [ ] Pages load without font/asset errors

### Manual Verification

- [ ] Mobile and operator pages use consistent color palette, radii, spacing
- [ ] Geist/Inter fonts applied; typography matches design intent
- [ ] Dark theme preserved or aligned with Pencil variables
- [ ] Components (buttons, cards, inputs) use design tokens

## Tasks

- [001-design-tokens-css-variables](./tasks/001-design-tokens-css-variables.md)
- [002-fonts-typography](./tasks/002-fonts-typography.md)
