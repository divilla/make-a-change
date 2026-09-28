# APIHydra baseline — P0 incomplete

Measured 2026-09-28 on `change/005-backend-verification-foundation`, reviewed implementation
`313702a` over base `7dc3aa0` (measured immediately before the review-fix commit). The complete campaign is `health-steps.yaml`:
both `GET /api/v1/health` and `GET /api/health` passed status 200 and the declared
stable JSON fields. APIHydra exited 0; the owned instrumented server exited 0
following SIGTERM; both covmeta and covcounters files were present before
`go tool covdata textfmt`. A private PostgreSQL cluster loaded unchanged init.sql
and seed.sql with ON_ERROR_STOP. Its shutdown succeeded and its temporary files
were removed. No external database was reset.

`make -C backend api-test` exited **2** (recipe exit **1**) because the code
coverage gate failed: **168 / 1,209 = 13.8958%**, required **at least 90%**.
The denominator includes unlinked `internal/error` (1 statement) and `pkg/db`
(4 statements), counted as zero using `go tool cover` structural block metadata.
`internal/domain` has no executable statements. Linked-package and block audits
passed. Unit/legacy counters were not included.

Operation coverage is separately **2 / 33 = 6.0606%**. The other 31 registered
pairs remain untested/blocked in [the ledger](../agents/backend-contracts.md),
not removed from the denominator. Health smoke is not a whole-backend campaign.

APIHydra provenance: installed binary `/home/vito/go/bin/apih`, module
`v0.1.1-0.20260907061551-c3947513e2a4`, build revision
`c3947513e2a4b948ce732dec60e090a5e83be744`. Its exact revision's local manual was
read via `git show c3947513e2a4:docs/user-manual/apih.md`; the checkout's HEAD is
newer. Invoked as `apih --parallelism 0 <private-complete-suite-copy>` with
`XDG_CACHE_HOME` inside the owned artifact directory. The copy changes only the
root base URL's port. No Debug directives or step selections are allowed.

APIHydra compares declared JSON fields and ignores extra object keys. An empty
expected body disables body comparison, so it cannot prove an empty 204 body.
`TestHealthAliasExactContracts` proves exact healthy/degraded JSON for both
aliases; `TestChangeAPIContracts` proves empty type-update 204 responses.

Raw current artifacts (ignored, regenerated per run) are in `../.coverage/api/`:
`runner.log`, `server.log`, `postgres.log`, `provenance.json`, `packages.json`,
`denominator.json`, `counters/`, `coverage.out`, `report.txt`, and `result.json`.
The provenance includes actual build/campaign/conversion/DB commands, tool
version and revision; denominator.json includes source SHA-256 hashes and linked
package inventory. Per-package unit and API totals are retained in
[the checkpoint](../agents/backend-refactor-checkpoint.md).

Legacy Go HTTP tests remain in `../api-tests/`. `make -C backend legacy-api-test`
uses the same owned private lifecycle but stores output under `.coverage/legacy`
and does not convert or report coverage. Those scenarios have not been replaced
or run as passing APIHydra evidence in this pass.
