# Task 002: Fonts and Typography

## Goal

Add Geist and Inter fonts per Pencil design; apply font families to headings, body, and secondary text. Ensure consistent typography across mobile and operator.

## Deliverables

- [ ] Add font imports: Geist (or fallback), Inter — via Google Fonts or local/cdn
- [ ] Apply `--font-primary` (e.g., Geist) for headings and primary text
- [ ] Apply `--font-secondary` (e.g., Inter) for body/secondary text
- [ ] Set base font-size, line-height for readability
- [ ] Use font-weight (400, 500, 600, 700) per design for headings vs body
- [ ] Update mobile and operator pages to use font variables

## Notes

- Design specifies Geist, Inter
- Geist: https://fonts.google.com/specimen/Geist (or Geist Sans from Vercel)
- Inter: https://fonts.google.com/specimen/Inter
- Existing operator uses DM Sans, JetBrains Mono — can migrate to Geist/Inter or keep mono for IDs
- Mobile uses DM Sans — migrate to design fonts
- Preconnect to font CDN for performance

## Verification

- Fonts load; no FOUT (flash of unstyled text) or minimize
- Headings and body text use correct families
- Typography matches design intent (sizes, weights)
