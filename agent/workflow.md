# Research → Plan → Implement → Validate Workflow

A HumanLayer-inspired workflow for complex tasks.

**Greenfield projects only.** This workflow is designed for new projects and codebases. For brownfield (existing) codebases, use a different approach—the research/plan/implement flow assumes you're building from scratch, not integrating into established systems.

## Tool Split

| Phase | Tool | Why |
|-------|------|-----|
| Research | **Cursor** | Rules, context, human review of findings |
| Plan | **Cursor** | Interactive iteration, rules, human approval |
| Implement | **Claude Code** | Terminal-based execution, phase-by-phase |
| Validate | **Cursor** | Review implementation against plan |

---

## When to Use This Workflow

| Use this workflow | Skip to Implement |
|-------------------|-------------------|
| **Greenfield projects** (new codebases) | Brownfield / existing codebases |
| Complex features or refactors | Simple, single-file changes |
| Multi-file implementations | Small bug fixes |
| Need mental alignment across sessions | Obvious one-liners |

---

## Phase 1: Research

**Goal**: Understand the codebase—what exists, where, how it works. No code changes, no suggestions.

### How to Run (Cursor)

Attach `@agent/prompts/research.md`, then: "Research the codebase to answer: [your question]"

### Output

- **Artifact**: `thoughts/research/YYYY-MM-DD-description.md`
- **Rules applied**: `10-codebase-research.mdc` (when working in `thoughts/research/`)

### Key Rules

- **Document, don't evaluate**—no improvements, refactors, or critiques
- Use parallel sub-tasks for different aspects
- Include file:line references
- Synthesize findings before writing

### Human Checkpoint

Review the research doc before planning. A bad research doc → thousands of bad lines of code.

---

## Phase 2: Plan

**Goal**: Create a detailed implementation plan with phases, success criteria, and verification steps.

### Prerequisites

- Research doc (or sufficient codebase understanding)
- Clear task/ticket description

### How to Run (Cursor)

Attach `@agent/prompts/plan.md` and `@thoughts/research/[your-research].md`, then: "Create an implementation plan for [task]"

### Output

- **Artifact**: `thoughts/plans/YYYY-MM-DD-description.md`
- **Rules applied**: `20-planning.mdc` (when working in `thoughts/plans/`)

### Plan Structure

Each phase must include:

- **Automated Verification**: Commands to run (`make test`, `npm run lint`, etc.)
- **Manual Verification**: Steps requiring human testing
- **What We're NOT Doing**: Explicit out-of-scope items

### Human Checkpoint

Review and approve the plan before implementation. A bad plan → hundreds of bad lines of code.

---

## Phase 3: Implement

**Goal**: Execute the plan phase-by-phase, verifying each step before proceeding.

### Prerequisites

- Approved plan in `thoughts/plans/`

### How to Run (Claude Code)

From Cursor's terminal:

```
claude
/implement_plan thoughts/plans/[file].md
```

Have the plan file path ready. Claude Code will read the plan and execute phase-by-phase.

### Output

- Code changes
- Updated checkboxes in the plan
- Pause after each phase for human verification

### Key Rules

- Follow the plan's intent; on mismatch, STOP and explain
- Run automated verification after each phase
- Pause for manual verification unless instructed to run multiple phases
- Trust completed checkmarks when resuming

### Human Checkpoint

After each phase: run manual verification from the plan, then confirm before next phase.

---

## Phase 4: Validate

**Goal**: Verify implementation against the plan. Run checks, identify gaps, produce a report.

### When to Run

After implementation is complete (ideally after commit, before PR).

### How to Run (Cursor)

Attach `@agent/prompts/validate.md` and `@thoughts/plans/[your-plan].md`, then: "Validate the implementation against thoughts/plans/[file].md"

### Output

- Validation report (in chat or `thoughts/handoffs/`)
- Pass/fail for automated verification
- Manual testing checklist
- Deviations and recommendations

### Recommended Order

1. Implement → 2. Commit → 3. Validate → 4. PR description

---

## Handoff (Between Sessions)

When switching between Cursor and Claude Code, or to a new session:

**Cursor**: Attach `@agent/prompts/handoff.md`, then: "Create a handoff for the current work"

### Output

- **Artifact**: `thoughts/handoffs/YYYY-MM-DD_HH-MM-SS_description.md`

### Resume

In a new session: "Read thoughts/handoffs/[file].md and continue from the Action Items."

---

## Quick Reference

| Phase | Tool | Invocation |
|-------|------|------------|
| Research | Cursor | `@agent/prompts/research.md` + "Research: [question]" |
| Plan | Cursor | `@agent/prompts/plan.md` + research + "Create plan for [task]" |
| Implement | **Claude Code** | `claude` → `/implement_plan thoughts/plans/[file].md` |
| Validate | Cursor | `@agent/prompts/validate.md` + plan + "Validate..." |
| Handoff | Cursor | `@agent/prompts/handoff.md` + "Create handoff" |

---

## Context Management (HumanLayer Principle)

- **Target**: Keep context utilization in 40–60%
- **Compaction**: Distill findings into `thoughts/` artifacts between phases
- **Human leverage**: Review research and plans—highest ROI
- **Specs as source of truth**: Plans and research guide implementation

---

## Artifact Locations

| Type | Path |
|------|------|
| Research | `thoughts/research/` |
| Plans | `thoughts/plans/` |
| Handoffs | `thoughts/handoffs/` |
