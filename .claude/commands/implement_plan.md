---
description: Implement technical plans from thoughts/plans with phase-by-phase verification
---

# Implement Plan

Implement an approved technical plan from `thoughts/plans/`. Execute phase-by-phase with verification.

## Getting Started

When given a plan path:
- Read the plan completely; note existing checkmarks
- Read the original ticket and all files mentioned
- Create a todo list to track progress
- Start implementing when you understand the work

If no plan path provided, ask for one.

## Implementation Philosophy

- Follow the plan's intent while adapting to reality
- Implement each phase fully before moving to the next
- Update checkboxes in the plan as you complete sections
- When things don't match, STOP and explain clearly

## Follow the Plan's Intent

- Pause for human verification after each phase (unless instructed to run multiple phases)

## On Mismatch

If reality doesn't match the plan:
- STOP and explain the discrepancy
- Present: Expected vs Found vs Why it matters
- Ask how to proceed before deviating

```
Issue in Phase [N]:
Expected: [what the plan says]
Found: [actual situation]
Why this matters: [explanation]

How should I proceed?
```

## Verification

After each phase:
1. Run automated success criteria
2. Fix issues before proceeding
3. Update plan checkboxes
4. Inform human that manual verification is ready
5. Wait for confirmation before next phase

```
Phase [N] Complete – Ready for Manual Verification

Automated verification passed:
- [List checks that passed]

Please perform manual verification from the plan:
- [List manual steps]

Let me know when complete so I can proceed to Phase [N+1].
```

If instructed to run multiple phases consecutively, skip the pause until the last phase.

Do not check off manual verification items until confirmed by the user.

## Resuming

If the plan has existing checkmarks, trust completed work and pick up from the first unchecked item.
