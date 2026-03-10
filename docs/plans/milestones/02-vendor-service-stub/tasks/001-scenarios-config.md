# Task 001: Scenarios Config

## Goal

Define and populate `config/scenarios.json` mapping account prefixes to scenario names.

## Deliverables

- [ ] `config/scenarios.json` with mappings for all 7+ scenarios
- [ ] Example: `ACC-IQA-BLUR` → `iqafail_blur`, `ACC-MICR-FAIL` → `micr_fail`, etc.
- [ ] Config loadable at runtime

## Notes

Scenarios: `iqafail_blur`, `iqafail_glare`, `micr_fail`, `duplicate`, `amount_mismatch`, `clean_pass`, `iqapass`

## Verification

- Config file parses; mappings resolve correctly
