# PIV Loop — Create Structured Plan

You are creating a structured implementation plan from a completed exploration phase.
This plan will be the SOLE GUIDE for the implementation agent — it must be complete,
specific, and actionable.

**Original request**: $ARGUMENTS
**Final exploration summary**: Read `$ARTIFACTS_DIR/exploration.md`, approved at the exploration gate.

---

## Step 1: Read the Codebase (Again)

Before writing the plan, verify your understanding is current:

1. **Read CLAUDE.md** — capture all relevant conventions
2. **Read every file you plan to change** — note exact current state
3. **Read example test files** — understand testing patterns
4. **Check for any recent changes** — `git log --oneline -10`

## Step 2: Plan File Location

Save the plan to `$ARTIFACTS_DIR/plan.md`.
The directory already exists (pre-created by the workflow executor).

## Step 3: Write the Plan

Use this template. Fill EVERY section with specific, verified information.

```markdown
# Feature: {Title}

## Summary
{1-2 sentences: what changes and why}

## Mission
{The core goal in one clear statement}

## Success Criteria
- [ ] {Specific, testable criterion}
- [ ] All validation passes (`bun run validate` or equivalent)
- [ ] No regressions in existing tests

## Scope
### In Scope
- {What we ARE building}
### Out of Scope
- {What we are NOT building — and why}

## Codebase Context
### Key Files
| File | Role | Action |
|------|------|--------|
| `{path}` | {what it does} | CREATE / UPDATE |

### Patterns to Follow
{Actual code snippets from the codebase to mirror}

## Architecture
- {Decision 1 — with rationale}
- {Decision 2 — with rationale}

## Task List
Execute in order. Each task is atomic and independently verifiable.

### Task 1: {ACTION} `{file path}`
**Action**: CREATE / UPDATE
**Details**: {Exact changes — specific enough for an agent with no context}
**Pattern**: Follow `{source file}:{lines}`
**Validate**: `{command to verify this task}`

## Testing Strategy
| Test File | Test Cases | Validates |
|-----------|-----------|-----------|
| `{path}` | {cases} | {what it validates} |

## Validation Commands
1. Type check: `{command}`
2. Lint: `{command}`
3. Tests: `{command}`
4. Full validation: `{command}`

## Risks
| Risk | Impact | Mitigation |
|------|--------|------------|
| {risk} | {HIGH/MED/LOW} | {specific mitigation} |
```

## Step 4: Verify the Plan

1. Check every file path referenced — verify they exist
2. Check every pattern cited — verify the code matches
3. Check task ordering — ensure dependencies are respected
4. Check completeness — could an agent with NO context implement this?

## Step 5: Report

```
## Plan Created

**File**: `$ARTIFACTS_DIR/plan.md`
**Tasks**: {count}
**Files to change**: {count}

Key decisions:
- {decision 1}
- {decision 2}

Please review the plan and provide feedback.
```
