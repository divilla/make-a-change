Simplify the testcase SQL in db/init.sql. Optimize for fewer explicit locks, fewer statements, and reliance on PostgreSQL’s existing guarantees.

Use this structure:
Centralize testcase advisory locking in fn_testcase_history_insert: one transaction-level advisory lock per testcase ID. Read the latest history version; return NULL if already deleted; otherwise append the next version. Keep the existing deletion flag separate from the requested flag.
Scenario update: append history first, then update the live scenario/version only when the returned version is newer. Use UPDATE … RETURNING to return the actual updated version, or NULL when nothing changed.
Done update: use ordinary UPDATE … RETURNING, then recalculate counters. No preliminary parent lookup or explicit testcase lock.
Delete: use a plain ID predicate and append history through DELETE … RETURNING. No history-writing side effects in WHERE.

Confine explicit change/epic locking to counter recalculation.

Remove custom checks already enforced by foreign keys or unique constraints. Do not add helpers, defensive layers, isolation-level changes, or unrelated modifications.

Test each proposed lock removal in a disposable database. Check committed rows, history, return values, and counters—including changes without an epic. Test sequential operations as well as concurrent edits, done updates, and deletion.

Standard PostgreSQL errors can be acceptable outcomes of conflicting transactions. Do not add locks merely to eliminate 40P01 or transient 23505. Distinguish those from persistent failures during valid sequential operations and silent inconsistencies after successful commits.

For every explicit lock you propose retaining, show a minimal reproduction demonstrating why it is needed. Report results precisely; passing selected tests is not proof against every possible race.

Present the smallest proposed diff and wait for approval before changing production SQL.
