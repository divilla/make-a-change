# PIV Loop — Plan Refinement

The user is reviewing the implementation plan and providing feedback.

**User's feedback**: $LOOP_PREV.refine-plan-review.output.text

---

## Step 1: Read the Plan

Read `$ARTIFACTS_DIR/plan.md` and CLAUDE.md for conventions.

## Step 2: Process Feedback

**If there is no user feedback yet** (first iteration, $LOOP_PREV.refine-plan-review.output.text is empty):
- Read the plan carefully
- Present a summary of the plan's key decisions and task list
- Ask the user to review and provide feedback
- Present the result for the following approval gate.

**If the user provided specific feedback:**
- Parse each piece of feedback
- Edit the plan file directly:
  - Add/remove/modify tasks as requested
  - Update success criteria if needed
  - Adjust testing strategy if needed
  - Re-verify file paths and patterns after changes


## Step 3: Show Changes

```
## Plan Revised

Changes made:
- {change 1}
- {change 2}

Updated stats:
- Tasks: {count}
- Files to change: {count}

Review the updated plan and provide more feedback, then use the approval gate to proceed or request changes.
```

Only the following approval gate decides whether to proceed. Treat its previous feedback as requests to address, never as implicit approval.
