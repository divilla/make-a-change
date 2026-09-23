# Repository Helper Scripts

This directory contains maintained, user-owned helpers for working with the mch repository. These files are intentional project tooling, not generated output or repository-hygiene candidates. Do not delete or relocate them without explicit user approval.

## Git authentication

Before running a workflow that fetches or pushes, load your SSH key into the
SSH agent available in your terminal:

```shell
bash scripts/git-auth.sh
```

This uses `$HOME/.ssh/divilla-github/id_ed25519` by default. Set
`MCH_GIT_SSH_KEY` to use another private-key path; its matching `.pub` file must
also be available. An SSH agent must already be running, with `SSH_AUTH_SOCK`
available in the terminal where you run these commands.

The merge, promotion, commit, branch-creation, branch-renaming, implementation,
and review workflows check that the current agent can use this specific key
before remote Git access or local changes. The Bash merge wrappers inherit
the check from the workflows they invoke. A missing agent or key stops the
workflow with exit `1` and an actionable error:

```text
error: Git authentication is not ready. Run 'bash /path/to/scripts/git-auth.sh' in this terminal to load your SSH key, then retry this command.
```

The check verifies current authentication readiness, not a persistent marker
that could outlive an agent. Run `git-auth.sh` again when the key is removed or
the agent is restarted. `bash scripts/git-auth.sh --check` performs the same
check without loading a key or prompting for its passphrase. Local Git merge
operations do not themselves require a password; the merge workflow needs this
check because it also fetches and pushes.

`perl scripts/test-git-auth.pl` tests setup with disposable keys and an isolated
SSH agent, and verifies that every guarded workflow stops before remote access
or repository changes. Other workflow tests use an SSH check stub and local Git
remotes, so they do not require your real key or contact an external service.

## Uncommitted work

Merge, promotion, branch-creation, branch-renaming, implementation, and review
workflows require a clean repository before authentication or remote Git access.
Staged changes, unstaged changes, deleted files, untracked files, and dirty
submodules all stop the workflow with exit `1`:

```text
error: Uncommitted work detected. Commit or stash your changes, then retry.
```

The check covers the entire repository, including when invoked from a
subdirectory. Promotions check this even when there is nothing to promote.
The wrappers report the error from their first failing workflow and stop.
`commit-user.pl` and `commit-agent.pl` remain exempt because their purpose is
to commit your pending work; both still require Git authentication.
Authentication setup and the interactive `codex-resume.sh` shortcut do not
require a clean repository.

## Implement a specification

Run the branch-creation, implementation, and review workflow with:

```shell
scripts/branch-create.sh agent/specs/000-domain-types.md &&
scripts/codex-code-spec.pl agent/specs/000-domain-types.md &&
scripts/codex-review-loop.pl agent/specs/000-domain-types.md
```

This creates and checks out the specification's change branch, uses
`$change-code` to implement the specification, commits that result, and then
starts the review loop. Each findings pass uses `$change-fix-findings` before
the loop commits and pushes the resulting fixes.

## `commit-user.pl` and `commit-agent.pl`

Both scripts stage all repository changes, create a commit, and push the current branch to `origin`. Use `commit-user.pl` for user-authored changes; with no argument, its commit message defaults to `User commit`:

```shell
scripts/commit-user.pl
```

Use `commit-agent.pl` for agent-authored changes; with no argument, its commit message defaults to `Agent commit`:

```shell
scripts/commit-agent.pl
```

Pass either script one argument to provide a different commit message:

```shell
scripts/commit-user.pl "Initial commit"
scripts/commit-agent.pl "Implement runner"
```

Both scripts resolve the repository root from their own location, so they can be invoked from any working directory. They refuse to run from a detached `HEAD`.

## `branch-create.sh`

Create and check out a change branch for a specification:

```shell
scripts/branch-create.sh agent/specs/000-domain-types.md
```

The specification path must have the form `agent/specs/<spec-slug>.md`. The
script requires a clean working tree, fetches `origin`, and initializes missing
dev and stage branches as described below. It then creates and checks out
`change/<spec-slug>` directly from freshly fetched `origin/dev`,
without setting an upstream to dev. It stops if the change branch already
exists locally or as an `origin` remote-tracking ref. Local dev may be stale
or absent. On success, it prints the new branch name.

## Merge and promote changes

Run these helpers from the repository checkout. They require Git. The release
path is `dev` → `stage` → `master`. Branch creation, merging, and promotion
automatically ensure that **dev and stage exist both locally and on origin**:

When setup is needed, the initializer pulls `origin/master` into local `master`
with `--ff-only` (creating local master if needed). It then initializes **stage
from master, followed by dev from stage**. On a fresh setup, all three local and
remote branches start at the same latest production commit. The original branch
or detached checkout is restored after pulling master, including if the pull
fails. Unpublished or divergent local master commits must be resolved first.

| Existing copies of dev or stage | Initialization |
| --- | --- |
| Local and remote | Preserve both branch tips. |
| Remote only | Create the local branch from the remote tip. |
| Local only | Initialize from the release baseline below; preserve a different local tip in `backup/<branch>-before-init-<sha>`. |
| Neither | Create stage from updated master; create dev from stage's origin tip. |

Starting a completely missing stage at production preserves pending dev commits
for the requested promotion mode. A missing remote stage starts from updated
master; a missing remote dev starts from stage's origin tip. Local-only release
tips are not published because they may contain unsquashed change commits.
Their backup branches retain those commits. Existing published release history
is preserved. Once both branches
exist locally and remotely, setup does not pull master or reset either branch.
Remote creation uses a
lease requiring the branch to be absent, so it cannot overwrite a branch another
writer creates concurrently. Restricted fetch configurations are handled using
explicit refspecs when needed.

Missing dev or stage branches are normal setup conditions, not errors. Worktree
and authentication checks still run first. Actual Git failures still stop the
workflow. `origin/master` remains required whenever branch initialization is
needed, and as the destination for production promotion.

### `merge-to-dev.pl`

```shell
scripts/merge-to-dev.pl
```

Takes no arguments. The current branch must be a numbered
`change/<change-slug>` branch, have a clean working tree, and match its published
branch on `origin`. The merge always targets **dev** and requires no PR or `gh`.
The fetched dev tip must already be an ancestor of the change, otherwise rebase
the change first.

The script squashes the change with message `Implement change <change-slug>`,
publishes the squash using a lease on the remote change branch,
then fast-forwards and pushes dev. It reuses an existing squash with the expected
parent and message. After verifying the published dev SHA, it deletes the remote
change branch with a lease, keeps the local change branch, and leaves dev checked
out.

### `promote-to-stage.pl` and `promote-to-prod.pl`

| Script | Source | Destination |
| --- | --- | --- |
| `promote-to-stage.pl` | `dev` | `stage` |
| `promote-to-prod.pl` | `stage` | `master` |

Both scripts fetch `origin` and accept the same optional argument:

```shell
scripts/promote-to-stage.pl             # next pending commit
scripts/promote-to-stage.pl all         # source tip
scripts/promote-to-stage.pl <commit-sha> # through this commit
scripts/promote-to-prod.pl all
```

No argument promotes the oldest pending commit on the source's first-parent
release path after the destination. A merge commit brings its side-branch
ancestors with it. `all` promotes to the fetched source tip. A full or unambiguous
abbreviated commit SHA promotes through that exact commit, including all preceding
changes. Promotions preserve commit identities and push without force.

A supplied SHA is validated against the fetched source history first. If absent,
the script exits `1` with `<sha> does not exist in dev` (stage promotion) or
`<sha> does not exist in stage` (production promotion). Branch names and other
revision expressions are not SHA arguments. A valid SHA already in destination
history, or no pending commits, is a successful no-op with exit `0`.

Every promotion invocation requires a clean working tree before fetching or
checking its target SHA. An actual promotion also requires a fast-forward to the
selected SHA. Divergence, extra local destination commits, command failures, and
rejected pushes exit `1`. On success, the destination is checked out and its
remote tip is verified against the selected SHA. These scripts promote Git
branches; any deployment automation is separate.

### `merge-to-stage.sh` and `merge-to-prod.sh`

```shell
scripts/merge-to-stage.sh [all|<commit-sha>]
scripts/merge-to-prod.sh [all|<commit-sha>]
```

Both Bash wrappers first run `merge-to-dev.pl` without arguments. They then pass
their arguments unchanged to `promote-to-stage.pl`; `merge-to-prod.sh` also runs
`promote-to-prod.pl` with those same arguments. With no argument, each promotion
advances one pending commit, which may precede the change just merged. A SHA
argument must refer to the commit to promote, not a pre-squash change commit that
will be replaced by merging.

The wrappers stop at the first failed step, exit `1`, and report its invocation
line range, wrapper filename, and captured command stderr:

```text
error: lines 25-25 of file scripts/merge-to-prod.sh: <error-captured-from-command>
```

Completed steps remain in place; there is no rollback. Resume a failed promotion
by calling the corresponding promotion script directly. A rejected push can also
leave the local destination advanced while its remote remains unchanged.

### Tests

The Perl suites exercise shared logic and run the real scripts against temporary
local Git repositories, with a failing `gh` stub to detect accidental GitHub
dependencies. They do not contact GitHub or modify the working repository.
No separate wrapper test scripts are maintained.

```shell
perl scripts/test-merge-to-dev.pl
perl scripts/test-promote-to-stage.pl
perl scripts/test-promote-to-prod.pl
perl scripts/test-git-auth.pl
perl scripts/test-worktree.pl
perl scripts/test-release-branches.pl
bash scripts/test-create-change-branch.sh
```

## `codex-code-spec.pl`

Implement a specification on its already-checked-out change branch:

```shell
scripts/codex-code-spec.pl agent/specs/000-domain-types.md
```

The specification path must have the form `agent/specs/<spec-slug>.md`, and the
current branch must be `change/<spec-slug>`. The script requires a clean working
tree so the automated commit cannot absorb unrelated changes.

The startup context is printed in `Repository`, `Specification`, `Branch`
order, followed by an `=== Implementation ===` heading and the rendered Codex
command. In a color-capable terminal, labels remain white while repository,
specification, and branch values are blue, magenta, and green respectively.

The implementation runs as
`codex exec --json -o <temporary-result> '$change-code <specification>'` with
the same elapsed-time, output-marker, activity-marker, success, failure, and
interrupt behavior as `codex-review-loop.pl`. Raw JSON output is suppressed on
success and printed on failure. When Codex succeeds, the script requires both a
final response and repository changes. It prints the final response between the
progress line and the changed-file list, then commits and pushes the changes as
`Implement change <spec-slug>`. Temporary output stays outside the repository
and is removed on exit.

## `codex-review-loop.pl`

Run Codex review-and-fix passes until the native review returns no comments:

```shell
scripts/codex-review-loop.pl agent/specs/010-executor-service.md
```

The required first positional argument is the specification file. The script
always reviews a branch range so every pass includes fixes committed by earlier
passes. It uses the default remote branch from `origin/HEAD` unless
`--base BRANCH` is supplied; arguments after the specification are forwarded
to Codex. Every review pass uses the native `codex exec review --base` target,
with the base resolved to a pinned commit before the loop begins. The
specification is supplied only to the subsequent `$change-fix-findings` fixer
because Codex treats a custom review prompt and `--base` as conflicting review
targets.
Custom review instructions from the caller are therefore rejected, whether
supplied as a bare positional prompt, `-` for standard input, or after `--`.
For example:

```shell
scripts/codex-review-loop.pl agent/specs/010-executor-service.md --base develop
```

The `--uncommitted` and `--commit SHA` review targets are rejected because they
cannot include the loop's later fix commits.

Each invocation creates a private temporary directory outside the repository,
preferring `TMPDIR`, `TMP`, `TEMP`, and then `/tmp`, and prints its
`findings.md` path. A candidate that resolves inside the checkout is skipped.
Every findings pass redirects that file through standard input to a fresh
`codex exec` invocation and explicitly invokes `$change-fix-findings` with the
positional specification file. The skill validates the findings against the
specification and repository contracts, implements valid fixes with tests and
verification, and preserves unrelated changes. The prompt leaves commits and
pushes to the loop. Each review and fix command has a numbered heading, and the
loop prints each captured final response after its progress line. If the fixer
makes no repository changes, the response keeps protected-contract blockers and
rejected findings visible before the loop stops without committing. Otherwise,
it prints changed files, then commits and pushes them as
`review fixes 01`, `review fixes 02`, and so on. Native Codex review rendering
adds an exact `Review comment:` or `Full review comments:` header whenever its
structured result contains findings. The loop checks those headers instead of
searching unconstrained review prose for verdict text, and rejects empty output
plus Codex's failed-response and interrupted-review fallbacks before a
headerless result can be accepted as clean. `codex exec review`
ignores `--output-schema`, so the script rejects that option. The commit
operation is built into the running review-loop process, so a fixer-modified
repository helper is never executed with the caller's permissions. The script
requires a clean working tree so automated commits cannot include unrelated
work. All generated findings, final-response, and progress-log artifacts
stay outside the repository, and their private directory is removed on every
exit path.
The default `/tmp` location works on Linux and macOS. The script requires Perl
and a POSIX environment because interruption is propagated to the active Codex
process group.

While Codex runs, its output is replaced by a progress line such as
`[ ] 00:00 •••••••••●`, containing elapsed minutes and seconds plus accumulated
output bullets, rate-limited to at most one bullet per second while output is
received. The activity marker is appended directly to the bullets without
intervening whitespace. In a terminal, Perl's timed I/O
multiplexing updates the activity marker every 250 milliseconds independently
of Codex output, without delivering asynchronous progress signals to the
process. The reusable implementation lives in `lib/mch/Progress.pm`. The
status resolves to `[✅]` on success or `[❌]` on failure or interruption; a
finished line looks like `[✅] 11:11 ••••••••••••`. Pressing Ctrl+C
terminates the active Codex command and the loop. Review startup output uses the
same repository, specification, and branch order and colors as the
implementation script, followed by the base, pinned base, review options, and
findings file. Additional values retain their established colors. All terminal
colors are disabled when `NO_COLOR` is set.
Before each run, the Codex command is printed with readable single-quoted
arguments between separator lines sized to its longest rendered line. The
fixer's stdin redirection is rendered on a separate line, so neither separator
extends into that line. Codex runs in JSON mode so progress events remain
streamable while their raw content is suppressed on success. If Codex fails,
its captured output is printed to standard error for diagnosis.
