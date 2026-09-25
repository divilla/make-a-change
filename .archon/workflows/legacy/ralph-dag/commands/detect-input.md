# Detect Ralph Input

**User input**: $ARGUMENTS

Determine what the user provided and prepare the PRD directory. Follow these steps exactly:

## Step 1: Detect worktree

Run `git worktree list --porcelain` to check if you're in a worktree.
If you see multiple entries, you ARE in a worktree. The first entry (the one without "branch" pointing to your current branch) is the **main repo root**. Save it — you'll need it to find files.

## Step 2: Classify the input

Look at the user input above. It's one of three things:

**Case A — Ralph directory path** (contains `.archon/ralph/`):
Extract the directory. Check if both `prd.json` and `prd.md` exist there (try locally first, then in the main repo root if in a worktree).

**Case B — File path** (ends in `.md`):
This is an external PRD file. Find it:
1. Try the path as-is (relative to cwd)
2. Try it as an absolute path
3. If in a worktree, try it relative to the **main repo root** from Step 1
Once found, read the file to confirm it's a PRD.

**Case C — Free text**:
Not a file path — it's a feature idea.

## Step 3: Auto-discover existing ralph PRDs

If the input didn't point to a specific path, check if `.archon/ralph/` contains any `prd.json` files:
```bash
find .archon/ralph -name "prd.json" -type f 2>/dev/null
```

## Step 4: Take action based on classification

**If Case A and both files exist** → output `ready` (no further action needed)

**If Case B (external PRD found)**:
1. Derive a kebab-case slug from the PRD filename or title (e.g., `workflow-lifecycle-overhaul`)
2. Create the ralph directory: `mkdir -p .archon/ralph/{slug}`
3. Copy the PRD content to `.archon/ralph/{slug}/prd.md`
4. Output `external_prd` with the new prd_dir

**If Case C or auto-discovered ralph dir has prd.md but no prd.json** → output `needs_generation`

## Output

Your final output MUST be exactly one JSON object:
```json
{"input_type": "ready|external_prd|needs_generation", "prd_dir": ".archon/ralph/{slug}"}
```
