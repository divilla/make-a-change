begin;

truncate table
    public.test_case_history,
    public.test_case,
    public.change_history,
    public.change,
    public.epic_history,
    public.epic,
    public.project
restart identity;

insert into public.project (name) values ('demo1'), ('demo2'), ('demo3');

insert into public.epic (project_id, name)
select p.id, seed.name
from public.project p
cross join (values
    ('Echo Router'),
    ('Echo Middleware'),
    ('Echo Binder'),
    ('Echo Documentation'),
    ('Echo Maintenance')
) as seed(name)
where p.name = 'demo1';

update public.project set last_ref = 200 where name = 'demo1';

create or replace function pg_temp.fn_demo_change_insert(
    _project_id bigint,
    _change_types text[],
    _epic_id bigint,
    _title text,
    _spec text
) returns bigint
    language plpgsql
as
$$
declare
    _id bigint;
    _def text;
begin
    _def := coalesce(nullif(trim(_spec), ''), format('# %s', trim(_title)));

    select public.fn_change_insert(_project_id, gen_random_uuid(), _title, _def) into _id;
    call public.sp_change_assign_flow(_id);

    update public.change
    set
        change_types = coalesce(_change_types, array[]::text[]),
        epic_id = _epic_id
    where id = _id;

    -- Keep every eleventh demo Change's optional spec in its never-submitted state.
    if not exists (
        select 1
        from public.change c
        join public.project p on p.id = c.project_id
        where c.id = _id
          and p.name = 'demo1'
          and c.ref % 11 = 0
    ) then
        call public.sp_change_spec_update(_id, _spec, false);
    end if;

    return _id;
end;
$$;

create or replace procedure pg_temp.sp_demo_change_pr_update(
    _id bigint,
    _pr text,
    _pr_url text
)
    language plpgsql
as
$$
begin
    call public.sp_change_pr_update(_id, _pr, false);

    update public.change
    set pr_url = _pr_url
    where id = _id;
end;
$$;

do
$$
declare
    _project_id bigint;
    _change_id bigint;
begin
    select id into _project_id from public.project where name = 'demo1';

    -- Echo PR pair 1: #3028 creates the Change, #3024 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Router'),
        'Respect q=0 in gzip content negotiation',
        '## Summary

Fix gzip middleware to respect `q=0` values in the `Accept-Encoding` header.

## Problem

According to RFC 9110, an encoding with `q=0` is explicitly unacceptable. The current implementation does not respect this quality value and may enable gzip even when the client sends:

```http
Accept-Encoding: gzip;q=0
```

## Solution

Update the gzip negotiation logic to honor `q=0` quality values and skip gzip when it is explicitly marked as unacceptable.

## Testing

- added a regression test covering `Accept-Encoding: gzip;q=0`
- `go test ./middleware`'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Split out of #3023 at @aldas''s request so the router and JSON changes land as independent PRs.

## What

`findStaticChild` scanned `[]*node` and dereferenced every child just to read its one-byte label — a pointer chase per static step on the routing hot path. This adds a parallel `scLabels []byte` (kept in sync in `addStaticChild`, `newNode`, the insert split, and `Remove`) and scans that contiguous slice instead, indexing into `staticChildren` only on a hit.

It also folds the is-leaf recomputation — previously duplicated across 5 sites, with the `Remove` copy subtly using `len()==0` while the others used `== nil` — into a single `refreshLeaf()` helper that uses `len()`, so it is correct whether `staticChildren` is `nil` or an emptied-but-non-nil slice left after a removal.

## Numbers

```
benchstat (n=6):
RouterStaticRoutes-14   8.49µs -> 8.01µs   -5.67%  (p=0.002)
RouterGitHubAPI-14      16.93µs -> 16.45µs  -2.86%  (p=0.002)
RouterParseAPI / GooglePlusAPI: neutral (param-heavy)
allocations: 0, unchanged
```

The companion ServeHTTP benchmark harness lands with #3023 (the JSON PR); routing benchmarks there exercise this change once both merge.

🤖 Generated with [Claude Code](https://claude.com/claude-code)',
        'https://github.com/labstack/echo/pull/3028'
    );

    -- Echo PR pair 2: #3023 creates the Change, #3020 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Middleware'),
        'perf(json): pooled-buffer JSON deserialize',
        'Per @aldas''s request, this PR is now JSON-only. The router change moved to #3024.

## What

`DefaultJSONSerializer.Deserialize` used `json.NewDecoder(body).Decode()`, which allocates a decoder and its internal read buffer on every JSON request. It now reads the body into a capped pooled buffer and decodes with `json.Unmarshal`; `Unmarshal` does not retain the input slice, so the buffer is safe to reuse. The pool drops oversized buffers (>64 KiB) so a single large body cannot pin memory.

## Numbers

`BenchmarkBind_JSON`: 1008 → 312 B/op (**-69%**), 9 → 7 allocs, ~12% faster.

## Behavioral note

`json.Unmarshal` is stricter than streaming `Decode` — it rejects trailing data after the first top-level value and reports `"unexpected end of JSON input"` for truncated bodies (both still 400 Bad Request). Two bind tests asserting the old `"unexpected EOF"` message are updated to match. Covered by new tests in `json_test.go` (trailing-data rejection, pooled-buffer reuse/concurrency under `-race`, the >64 KiB cap path, and body-read errors).

## DoS note (from review)

The full body is read into memory before decoding. The previous `json.Decoder` path also fully materialized any *valid* large body, so the only real delta is a malformed-early huge body. The correct guard for untrusted input is `middleware.BodyLimit` / `http.MaxBytesReader`, which makes the read here fail fast — documented on `Deserialize`.

This PR also adds the general ServeHTTP/JSON benchmark harness (`perf_bench_test.go`) used to measure both this and #3024.

🤖 Generated with [Claude Code](https://claude.com/claude-code)'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'changelog + version string bump',
        'https://github.com/labstack/echo/pull/3023'
    );

    -- Echo PR pair 3: #3019 creates the Change, #3018 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Binder'),
        'backport PR 3016 from v5 to v4',
        'backport PR #3016 from v5 for https://github.com/labstack/echo/security/advisories/GHSA-vfp3-v2gw-7wfq

-----------

Make serving static file releated methods  and middleware not unescape path by default - so how the way Router interprets paths and Static methods/middleware is consistent.

Given following situation:
```go
// 0.
// given folder structure:
// private.txt
// public/
// public/index.html
// public/text.txt
// public/admin/private.txt

// 1. share `public/` folder contents from the server root. This folder actually contains subfolder `admin` which
// contents we want to forbid from downloading
e.Static("/", "public")

// 2. naively assume that everything under /admin folder is now forbidden
e.GET("/admin/*", func(c *Context) error {
    return ErrForbidden
})
```

Then requests to `/admin%2fprivate.txt` would not be matched to `GET /admin/*` route (routing does not look unescaped path) and static file serving will use unescaped path to serve the file.

Note: this way of "guarding" subfolders will never work for for paths like `/assets/../admin%2fprivate.txt` which will `path.Clean("/assets/../admin%2fprivate.txt")` to `/admin/private.txt` and are servable if static file serving is configured to unescape paths.

If you want to guard routes - use middlewares on `Static*` methods and before `Static` middleware.

 **Breaking change / migration:** If you serve files whose names contain URL-encoded characters (e.g., `/hello%20world.txt` → `hello world.txt`), you must now opt in:

```go
	e := echo.New()
	e.EnablePathUnescapingStaticFiles = true  // <-- enable old behavior
	e.Static("/", "public")
```
for static middleware
```go
	e.Use(middleware.StaticWithConfig(middleware.StaticConfig{
		EnablePathUnescaping: true, // <-- enable old behavior
	}))
```'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'remove dependency on labstack/echo v5 introduced in go.mod and go.sum in https://github.com/labstack/echo/pull/3017',
        'https://github.com/labstack/echo/pull/3019'
    );

    -- Echo PR pair 4: #3017 creates the Change, #3016 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Documentation'),
        'Update CI action versions for v4 branch',
        'Update CI action versions for v4 branch'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Revert PR #3009 changes to just disabling path escaping by default in static methods/middleware

#3009 is a little bit brute force hack to solve the problem from LLM. Claude proposed checking and fixing path used is not a maintainable solutions and there could be so many clever ways how bad actors try to manipulate the path, and the root cause is that the Router and code serving Static files are conceptionally using path differently - so more maintainable solution is to make these 2 acting consitently.

Note: disabling path escaping in static methods and static middleware is a breaking change.',
        'https://github.com/labstack/echo/pull/3017'
    );

    -- Echo PR pair 5: #3015 creates the Change, #3014 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Maintenance'),
        'fix(static): disable static path unescaping by default to prevent ACL bypass (GHSA-3pmx-cf9f-34xr)',
        '## Summary

Fixes **GHSA-3pmx-cf9f-34xr**, a bypass of the GHSA-vfp3-v2gw-7wfq fix (#3009).

The router matches the raw, still-encoded request path, so encoded separators and dot segments in a static wildcard are not treated as traversal during routing. Unescaping them in the static file resolver afterwards let an unauthenticated attacker read a file across a route-level middleware guard the encoded path never matched:

- `/public/%2E%2E/admin/secret.txt` → `admin/secret.txt` — **high severity, default router**
- `/public%2F..%2Fadmin%2Fsecret.txt` under `UseEscapedPathForMatching=true`, where the router decodes the path itself before the handler sees it — lower severity (non-default config)

Both returned `200` + the protected file on `master` at `5786024` (the exact commit the advisory tested).

## Approach

Rather than keep extending an encoding denylist (the original fix blocked `%2F`/`%5C` but missed `%2E%2E`), this addresses the root cause: **make static path unescaping opt-in.** This follows @aldas''s proposal (`e85ee8f`), rebased onto current `master`.

- **echo:** `Config.EnablePathUnescapingStaticFiles` (default `false`) controls unescaping for `Echo.Static`/`StaticFS` and `Group.Static`/`StaticFS`.
- **middleware:** `StaticConfig.EnablePathUnescaping` replaces the now-deprecated `DisablePathUnescaping`; the default is the safe, no-unescape mode.

With unescaping off, `%2F`/`%5C`/`%2E%2E` stay literal and never become separators or traversal.

As defense in depth — and to also close the `UseEscapedPathForMatching` variant, where the **router** (not the handler) does the decoding — any `..` path segment in the resolved wildcard is rejected via `pathutil.HasDotDotSegment`, mirroring the `fs.ValidPath` "no `..` element" invariant. The existing encoded-separator guard remains as a backstop on the opt-in unescaping path.

## Verification

| Variant | `master` (5786024) | this PR |
|---|---|---|
| `/public/%2E%2E/admin/secret.txt` (default router) | 200 `TOP-SECRET` | **404** |
| `/public%2F..%2Fadmin%2Fsecret.txt` (`UseEscapedPathForMatching`) | 200 `TOP-SECRET` | **404** |
| middleware equivalents (both router modes) | 200 `TOP-SECRET` | **404** |

Regression tests cover both variants in both router modes, the opt-in mode (encoded `%20` filenames serve, but `%2F`/`..` still rejected), and `pathutil.HasDotDotSegment` units. `go test -race ./...` and `go vet ./...` pass.

## ⚠️ Breaking change

Static files whose names contain URL-encoded characters (e.g. `"hello world.txt"` via `/hello%20world.txt`) are **no longer served by default**. Set `EnablePathUnescapingStaticFiles` (echo) / `EnablePathUnescaping` (middleware) to opt back in. Because this flips a default, suggest releasing as **5.3.0** with an upgrade note rather than a patch.

## Notes for reviewers

- Omitted the `RawPath`-preferring directory-redirect tweak from `e85ee8f` to keep this scoped to the vuln; happy to fold it in.
- cc @aldas — this takes your disable-by-default approach plus the `..` rejection needed to close the `UseEscapedPathForMatching` variant your patch didn''t reach.

🤖 Generated with [Claude Code](https://claude.com/claude-code)'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Replaces all occurrences of `interface{}` with `any` for Go 1.18+ compatibility.',
        'https://github.com/labstack/echo/pull/3015'
    );

    -- Echo PR pair 6: #3013 creates the Change, #3012 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Router'),
        'ci: run workflows on the v4 branch, not just master',
        '## Problem
The **v4** LTS maintenance branch receives **no CI**. Both `echo.yml` (Run Tests) and `checks.yml` (Run checks) trigger only on `master` — a stale snapshot from when v4 *was* the default branch. As a result, PRs targeting `v4` (e.g. the recent security backport #3011) get zero automated testing and must be dispatched manually.

## Fix
Add `v4` to the `push` and `pull_request` branch filters in both workflows.

## Note on this PR
Because `pull_request` workflows run from the **base** branch''s config, this PR itself won''t auto-run CI (the base `v4` doesn''t have the trigger yet — chicken-and-egg). I''ve **dispatched both workflows manually on this branch** to validate the YAML and confirm they pass. After merge, all future v4 pushes/PRs will be tested automatically.

🤖 Generated with [Claude Code](https://claude.com/claude-code)'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Release prep for **v4.15.3** (v4 LTS).

- Bumps `Version` in echo.go 4.15.2 → 4.15.3.
- Adds the v4.15.3 `CHANGELOG.md` entry.

Headline: fixes **GHSA-vfp3-v2gw-7wfq** — the encoded path separator static bypass (v4 backport of #3009, merged in #3011). After merge, tag `v4.15.3`, publish the release, and amend the advisory to add the v4 affected product.

🤖 Generated with [Claude Code](https://claude.com/claude-code)',
        'https://github.com/labstack/echo/pull/3013'
    );

    -- Echo PR pair 7: #3011 creates the Change, #3010 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Middleware'),
        'fix(static): reject encoded path separators that bypass route-level middleware (v4 backport)',
        'v4 backport of #3009 (released in v5.2.0) for **GHSA-vfp3-v2gw-7wfq**.

## Summary
An encoded path separator (`%2F` or `%5C`) in a static file URL could bypass route-level access control and disclose files. The router matches against the raw, still-encoded path, so `%2F` is not a separator — `/admin%2Fsecret.txt` skips a protected `/admin/*` group, falls through to static serving, which then unescaped `%2F`→`/` and served `admin/secret.txt`.

v4 is affected on **both** static surfaces:
- `echo_fs.go` `StaticDirectoryHandler` (used by `Static`/`StaticFS`) — vulnerable to `%2F` **and** `%5C` (it used OS-specific `filepath.Clean`).
- `middleware/static.go` — vulnerable to `%2F` (it already used `path.Clean`, so not `%5C`).

## Fix
- Both surfaces reject a wildcard containing an encoded separator (`%2F`/`%2f` or `%5C`/`%5c`) with `404` before unescaping, via a shared `internal/pathutil` helper.
- `StaticDirectoryHandler` switched from `filepath.Clean`+`ToSlash` to `path.Clean` (OS-independent; keeps backslash literal on Windows).

## Tests
- New regression tests for `%2F`, `%5C`, double-encoded `%252F`, group `StaticFS`, and the static middleware on a group; unit test for the detector.
- Updated two existing cases (`open redirect vulnerability`, `Directory redirect#01`) that asserted the old `%2f`-unescaped redirect — they now correctly assert `404` + no `Location`.
- `go test -race ./...` and `go vet ./...` pass.

Targets the **v4** branch for a **v4.15.3** release; the advisory will be amended to add the `github.com/labstack/echo` (v4) affected product once tagged.

🤖 Generated with [Claude Code](https://claude.com/claude-code)'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Release prep for **v5.2.0**.

- Bumps `version.go` 5.1.1 → 5.2.0 (minor: the diff since v5.1.1 includes `feat(middleware): RateLimiterStoreContext` #3007).
- Adds the v5.2.0 `CHANGELOG.md` entry.

Headline: fixes **GHSA-vfp3-v2gw-7wfq** (encoded path separator static bypass, #3009). After merge, tag `v5.2.0` and publish the GitHub release.

🤖 Generated with [Claude Code](https://claude.com/claude-code)',
        'https://github.com/labstack/echo/pull/3011'
    );

    -- Echo PR pair 8: #3009 creates the Change, #3008 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Binder'),
        'fix(static): reject encoded path separator to prevent route-level auth bypass',
        '## Summary

Fixes an access-control bypass / static file disclosure where an **encoded slash (`%2F`) lets a request skip a protected route group yet still resolve the file on disk** (GHSA-vfp3-v2gw-7wfq, CWE-22).

The router matches routes against the raw, still-encoded request path, so `%2F` is **not** treated as a path separator — `/admin%2Fsecret.txt` is a single segment and never matches a protected `/admin/*` group. The request falls through to the static handler, which then `url.PathUnescape`d `%2F` back to `/` and resolved `admin/secret.txt` from disk. Router and file handler disagreed on what counts as a separator.

### Reproduction (before)
```
GET /admin/secret.txt    -> 403  (protected group fires)
GET /admin%2Fsecret.txt  -> 200  body="TOP-SECRET"   ← bypass + disclosure
```

Common affected pattern:
```go
adminGroup := e.Group("/admin", authMiddleware)
e.StaticFS("/", os.DirFS("public"))
```

## Fix

`StaticDirectoryHandler` now rejects a wildcard containing an encoded path separator (`%2F`/`%2f` or `%5C`/`%5c`) with `404` **before** unescaping, keeping the router and the file handler consistent. No real filename contains a path separator, so legitimate static requests are unaffected.

## Tests
- New `static_encoded_separator_test.go`: regression test for the bypass + unit test for the `%2F`/`%5C` detector (incl. lower-case hex and a benign `%25` case).
- Updated the existing `TestEcho_StaticFS` "open redirect" case: `…%2f..` now returns `404` with **no redirect emitted at all** (was a sanitized `301`), which closes that vector harder.
- `go test ./ ./middleware/` and `go vet ./` pass.

## Scope / notes
- This addresses the **static-disclosure** vector. The related report GHSA-xr3h-5475-xgqr (percent-encoded routing bypassing protected middleware on *non-static* routes) is a broader router-level decision and is intentionally **not** included here.
- Reported by @a-tt-om and @oran-gugu.

🤖 Generated with [Claude Code](https://claude.com/claude-code)'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        '## Summary

Optimizes Echo''s per-request hot paths to remove avoidable allocations and CPU work. **No public API changes; the standard-library JSON serializer remains the default.** All numbers are `benchstat` medians (n=8, Apple M3 Max / arm64, Go 1.26).

> **Note (per @aldas review):** the opt-in sonic serializer was **removed** from this PR — it belongs in `echox/cookbook` as a runnable example, not as a submodule in core. This PR is now purely core hot-path optimizations. See "Using a faster JSON encoder" below.

## What changed

**Core**
- **Middleware chain compiled once** (`echo.go`, `buildRouterChains`) and reused, instead of re-wrapping closures on every request. Routing stores the matched handler on the `Context`.
- **Context** (`context.go`): zero-copy `String`/`HTML`/`JSONP` writes (write-only `unsafe` view), reuse of `delayedStatusWriter` (guarded against re-entrant `c.JSON`) and the store map across requests, inline `Get`/`Set` unlock, and a single-key `QueryParam` fast path proven byte-for-byte equal to `url.ParseQuery().Get` (incl. malformed escapes / `;` / `+`).
- **Binder** (`bind.go`): per-`reflect.Type` field-metadata cache so struct tags are parsed once per type, not per request. Preserves the field-name error wrapping from #3005.
- **Middleware**: precompute the HSTS header once (`secure.go`); pool the request-ID `randomString` scratch buffers (`util.go`).
- New hot-path benchmark suite + pooling/dispatch regression tests.
- `test:` de-flake `TestStartConfig_WithListenerNetwork` (ephemeral port instead of a hard-coded one) — separable commit; fixes a pre-existing CI flake.

## Performance (before → after)

| Path | Before | After | Δ time | Allocs |
|---|--:|--:|--:|:--:|
| 5-middleware request | 101 ns | **34 ns** | **−66%** | 5 → **0** |
| `Set` per request | (1 map alloc) | **0 allocs** | — | 1 → **0** |
| `QueryParam` (single key) | 199 ns | **41 ns** | **−79%** | 4 → **0** |
| `String()` response | 191 ns | 188 ns | flat | 4 → **3** |
| `JSON()` response | 347 ns | 350 ns | flat | 5 → **4** |
| `Bind` query (5 fields) | 961 ns | **688 ns** | **−28%** | 8 |
| `bindData` w/ tags | 4973 ns | **2609 ns** | **−48%** | — |
| request-ID gen | 130 ns | 122 ns | −6% | 2 → **1** (−60% B) |
| Static / Param route | 27 / 42 ns | 27 / 43 ns | flat | 0 |

Headline: the middleware path and the `Set`/`QueryParam` paths are now **allocation-free**; binding is **28–48% faster**.

## Router — profiled, intentionally untouched

`-cpuprofile` shows the router is already **0 allocs/op**, with time dominated by the irreducible LCP byte-loop (58%) and method switch (11%). I implemented the httprouter `indices`/`IndexByte` trick for `findStaticChild` and **measured a 30–37% regression** on hits — Echo''s nodes have small fan-out, where the inlined linear scan beats a non-inlined `IndexByte` call — so it was reverted. No router change.

## Using a faster JSON encoder (e.g. sonic)

This PR does not bundle sonic. The `echo.JSONSerializer` interface already lets any app swap encoders in ~10 lines:

```go
import "github.com/bytedance/sonic"

type sonicJSON struct{}
func (sonicJSON) Serialize(c *echo.Context, v any, _ string) error {
	b, err := sonic.Marshal(v); if err != nil { return err }
	_, err = c.Response().Write(b); return err
}
func (sonicJSON) Deserialize(c *echo.Context, v any) error {
	return sonic.ConfigDefault.NewDecoder(c.Request().Body).Decode(v)
}
// e.JSONSerializer = sonicJSON{}
```

Measured (this machine, arm64): sonic **decode −44%** (a clear win on any arch), **encode +43%** (arm64 is sonic''s weak arch; usually a win on amd64). A full cookbook example with these caveats will be a separate PR to labstack/echox.

## Testing

- `go test ./...` + `-race` pass; `gofmt` + `go vet` clean.
- Added: store no-leak across `Reset`, JSON status across `Reset`, nested `c.JSON`, global/pre middleware on 404/405/OPTIONS, `randomString` concurrency, query fast-path stdlib-equivalence.',
        'https://github.com/labstack/echo/pull/3009'
    );

    -- Echo PR pair 9: #3007 creates the Change, #3006 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Documentation'),
        'feat(middleware): optional RateLimiterStoreContext for response headers (#2961)',
        '## Problem (#2961)

The rate limiter sets no `Retry-After` / `X-RateLimit-*` headers, and the `RateLimiterStore` interface (`Allow(identifier) (bool, error)`) gives a store no way to set them either.

## Fix

Add an **optional** interface:

```go
type RateLimiterStoreContext interface {
	AllowContext(c *echo.Context, identifier string) (bool, error)
}
```

When the configured store implements it, the middleware calls `AllowContext` (with the request context) instead of `Allow`, so the store can set response headers on the allow/deny decision.

**Fully backward compatible** — stores implementing only `Allow` are unaffected; the existing interface and the built-in memory store are unchanged.

This is the optional-interface approach @aldas proposed in the issue thread (mirroring the pattern used in the v4 proxy middleware). It intentionally does **not** retrofit the built-in store with full metadata plumbing (the part flagged as a larger rewrite in the thread) — it just provides the hook so stores can set headers.

## Test

`TestRateLimiter_storeAllowContextIsPreferred` (written first; fails before the change): a store implementing `AllowContext` is preferred over `Allow` and can set a `Retry-After` header. gofmt/vet clean; full rate-limiter suite passes (backward compat confirmed).

Addresses #2961.

🤖 Generated with [Claude Code](https://claude.com/claude-code)'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        '## Problem (#2599)

A file whose name contains a percent sign cannot be downloaded via the static middleware:

- `100%.txt` → `GET /100%25.txt` → **`invalid URL escape "%.t"`** (500)
- `foo%20bar.txt` → `GET /foo%2520bar.txt` → serves the wrong/missing file

`http.Request.URL.Path` is **already decoded** by net/http (per its docs), but the middleware unescaped it a second time.

## Fix

Default `pathUnescape` to `false`. The non-group path comes from `URL.Path` (already decoded), so it must not be unescaped again. Only the wildcard param from a group route (`c.Param("*")`, set explicitly) may still be escaped, and that case keeps its existing `DisablePathUnescaping` toggle — so group behavior is unchanged.

```
GET /100%25.txt       → 200  "hundred percent"
GET /foo%2520bar.txt  → 200  "literal percent twenty"
```

## Test

`TestStatic_servesFileWithPercentInName` (written first; fails on master with 500) using an in-memory `fstest.MapFS`. gofmt/vet clean; all `TestStatic*` and the full middleware package pass (group/`DisablePathUnescaping` cases unaffected).

Fixes #2599.

🤖 Generated with [Claude Code](https://claude.com/claude-code)',
        'https://github.com/labstack/echo/pull/3007'
    );

    -- Echo PR pair 10: #3005 creates the Change, #3004 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Maintenance'),
        'fix(binder): include field name in bind conversion errors (#2629)',
        '## Problem (#2629)

When `c.Bind()` fails a type conversion on form/struct data, the error gives no indication of *which* field failed:

```
POST /submit  number=10a
→ code=400, message=Bad Request, err=strconv.ParseInt: parsing "10a": invalid syntax
```

This makes it hard to render a useful validation message ("the `number` field must be an integer").

## Fix

`bindData` already has the field name (`inputFieldName`) in scope at each conversion site but returned the bare error. Wrap those returns with the field name using `%w` (so `errors.Is`/`errors.As` still work):

```
→ code=400, message=Bad Request, err=number: strconv.ParseInt: parsing "10a": invalid syntax
```

This is the wrap the reporter proposed in the thread (`fmt.Errorf("%s: %w", inputFieldName, err)`), applied to all four conversion-error sites (unmarshaler, slice element, scalar).

## Test

`TestBind_formConversionErrorIncludesFieldName` (written first; fails on master — error contains no field name). Four existing tests that asserted the bare message are updated to the field-prefixed form. gofmt/vet clean; full root-package suite passes.

Fixes #2629.

🤖 Generated with [Claude Code](https://claude.com/claude-code)'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        '## Problem (#2771)

A binding error returned from a handler is serialized as `{"message":"Bad Request"}` — the field name **and** the binder message are both lost.

### Root cause
`BindingError` embeds `*HTTPError` but does not implement `json.Marshaler`. In `DefaultHTTPErrorHandler` the type switch runs on the `HTTPStatusCoder` extracted via `errors.As`, whose dynamic type is `*BindingError`. Go''s `case *HTTPError` matches only the exact type, so a `*BindingError` falls through to the `default` branch → `{"message": http.StatusText(code)}`. Regression from fbfe216 (#2456).

Verified on current `master`:
```
GET /doc?docNum=abc  →  400  {"message":"Bad Request"}
```

## Fix

Implement `MarshalJSON` on `*BindingError` so it takes the handler''s existing `case json.Marshaler` branch (which is checked *before* `*HTTPError`). Restores the v4.10.2 structured response:
```
GET /doc?docNum=abc  →  400  {"field":"docNum","message":"failed to bind field value to int"}
```

This is the approach the maintainer outlined in the issue thread ("we could make `echo.BindingError` … implement `json.Marshaler`").

## Test

`TestBindingError_serializesToStructuredJSON` (written first; fails on master with `field=<nil>`, `message="Bad Request"`). gofmt/vet clean; full root-package suite passes.

Fixes #2771.

🤖 Generated with [Claude Code](https://claude.com/claude-code)',
        'https://github.com/labstack/echo/pull/3005'
    );

    -- Echo PR pair 11: #3003 creates the Change, #3002 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Router'),
        'test: lock in v5 group route method-handling (405 + OPTIONS)',
        '## What

Adds `group_method_handling_test.go`: tests that lock in v5''s method-handling semantics for routes registered through a `Group`, and act as a regression gate for changes that would reintroduce an implicit per-group catch-all (e.g. #2996, which restores it to fix CORS-on-group preflight).

## Behavior asserted (verified on `master`)

| Request on a `GET`-only group route | v5 today |
|---|---|
| `POST /api/users` | **405** + `Allow: OPTIONS, GET` |
| `OPTIONS /api/users` | **204** + `Allow: OPTIONS, GET` (preflight-relevant) |
| `GET /api/users` | 200 |
| `GET /health` (root) | 200, unaffected by the group |

## Why these are real gates

A group-level catch-all — whether registered manually via `g.RouteNotFound("/*", …)` or automatically (as in #2996) — matches **every** method, which masks both the 405 and the automatic OPTIONS response as **404**. Verified empirically: with such a catch-all, `POST` and `OPTIONS` on `/api/users` both return 404. The first two tests would catch that regression.

## Note

This replaces an earlier version of this PR whose comments described a middleware-triggered "auto catch-all" that does not exist on `master` (v5) — the tests passed for the wrong reason. Reworked after review to assert the actual, verified behavior and drop the inert no-op middleware. All four tests pass; gofmt- and vet-clean.

🤖 Generated with [Claude Code](https://claude.com/claude-code)'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        '## Why

Echo is actively maintained and shipping (v5.1.1 + v4.15.2 on 2026-05-01, `master` commits within days), but to a casual visitor the repo can read as inactive. This PR adds low-cost, **self-updating** signals that Echo is alive and clarifies its positioning.

## Changes

- **Dynamic badges** for latest release and last commit. These pull live from GitHub, so they can never go stale the way a hand-written "last updated" line does.
- **Positioning line** under the tagline explaining what Echo adds on top of Go''s `net/http` — a gentle answer to the "do I still need a framework after Go 1.22 routing?" question.
- **"Actively maintained" note** pointing at the badges.
- **ROADMAP.md (DRAFT)** documenting the version-support policy (v5 current, v4 LTS until 2026-12-31) and a Now/Next/Later view seeded from current issues. Marked DRAFT — maintainers own the final content. Linked from the README.

No code changes. Docs only.

🤖 Generated with [Claude Code](https://claude.com/claude-code)',
        'https://github.com/labstack/echo/pull/3003'
    );

    -- Echo PR pair 12: #3000 creates the Change, #2994 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Middleware'),
        'fix(middleware): reset ContentLength after gzip decompression',
        '## Summary

Fix `Decompress` so middleware after it does not keep using the compressed request size after the body has been replaced with the decoded gzip stream.

For example:

```go
e.Use(middleware.Decompress())
e.Use(middleware.BodyLimit(4))
```

A gzipped request whose decoded body is `ok` should pass:

```bash
printf "ok" | gzip | curl -X POST \
  -H "Content-Type: text/plain" \
  -H "Content-Encoding: gzip" \
  --data-binary @- \
  http://localhost:8080/
```

Before this change, `Decompress` replaced `req.Body` but left `req.ContentLength` set to the compressed size, so `BodyLimit` could reject the request before reading it:

```text
{"message":"Request Entity Too Large"}
```

With this change, `Decompress` sets `req.ContentLength = -1` after gzip decompression is set up, so downstream middleware enforces limits while reading the decoded body:

```text
ok
```

This intentionally does not change `Content-Encoding`, `GetBody`, or multiple content-coding behavior.

## Test plan

- [x] `go test -count=1 ./...`
- [x] `go test -race -count=1 ./...`
- [x] `go vet ./...`
- [x] `staticcheck ./...`
- [x] Manual curl check with the same handler returned `ok` and HTTP 200'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        '## Description

This PR fixes https://github.com/labstack/echo/issues/2993

The proxy middleware''s WebSocket path currently sets `X-Forwarded-For` only when the header is empty, dropping the proxy''s own peer IP from the chain whenever upstream proxies had already added entries. This breaks downstream services that rely on the "rightmost untrusted" rule to extract the real client IP, including echo''s own `ExtractIPFromXFFHeader`.

The HTTP path delegates to `net/http/httputil.ReverseProxy`, which appends `RemoteAddr` to the existing `X-Forwarded-For` chain — either inline in `ServeHTTP`''s default Director path ([reverseproxy.go#L519-L531](https://github.com/golang/go/blob/go1.26.3/src/net/http/httputil/reverseproxy.go#L519-L531)) or via the explicit [`(*ProxyRequest).SetXForwarded`](https://github.com/golang/go/blob/go1.26.3/src/net/http/httputil/reverseproxy.go#L82-L96)
when a `Rewrite` callback is configured. The WebSocket path uses `proxyRaw`,
which writes the request verbatim via `r.Write(out)`, so this middleware is the only place where the appending happens for WebSocket Upgrade requests.

## Change

Replace the "set if empty" guard with always-append. Read values via map access to preserve multi-line `X-Forwarded-For` headers (RFC 9110 §5.3 allows combining them by joining values with commas).

## Test

Added TestProxyWebSocketXForwardedFor exercising 4 cases:

- no incoming X-Forwarded-For → only c.RealIP()
- single-line single-entry → preserved + c.RealIP() at the tail
- single-line comma-separated → preserved + c.RealIP() at the tail
- multi-line headers (multiple X-Forwarded-For occurrences) → joined with , + c.RealIP() at the tail

Each case captures the request header at WebSocket Upgrade time inside the upstream handler and asserts both the appended tail and the preserved prefix.

## Backwards compatibility

The change is additive: existing entries are preserved and the proxy''s own peer IP is added at the tail. Downstream readers using the standard "rightmost untrusted" rule (e.g. echo.ExtractIPFromXFFHeader) see no behavioral difference for chains where they already worked, and start returning the correct IP for chains where the proxy IP was previously
dropped.

## Background

We hit this in production with an Echo-based WebSocket reverse proxy fronting an internal service that uses echo.ExtractIPFromXFFHeader for IP-based authorization. Multi-hop deployments (customer proxy → our reverse proxy → backend) broke because the reverse proxy''s egress IP was missing from the chain reaching the backend.',
        'https://github.com/labstack/echo/pull/3000'
    );

    -- Echo PR pair 13: #2992 creates the Change, #2990 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Binder'),
        'fix(middleware): correct documented KeyAuth KeyLookup default',
        'The `KeyLookup` field comment in `KeyAuthConfig` documents the default as `"header:Authorization"`, but the default applied by `DefaultKeyAuthConfig` is:

```go
KeyLookup: "header:" + echo.HeaderAuthorization + ":Bearer ",
```

which evaluates to `"header:Authorization:Bearer "` (also the value `ToMiddleware` falls back to when `KeyLookup` is empty). The `:Bearer ` cut-prefix trims the scheme from the header value, exactly as the same comment block describes.

The comment was correct in v4, where `AuthScheme: "Bearer"` was a separate field. In v5 `AuthScheme` was folded into `KeyLookup`, but this comment was left showing the old default.

Doc comment only — no behavior change.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Fixes #2853.

When Echo CORS middleware is run in a chained proxy setup (or in front of any handler copying upstream headers using Add), headers like Access-Control-Allow-Origin and Vary get duplicated in the response.

Changes:
- Run simple request CORS header writing inside a Before hook on the response. This allows the proxy''s CORS config to cleanly Set the headers, overriding duplicated upstream headers from the proxy or downstream response copy.
- Implement addVaryHeader helper to merge and deduplicate Vary values case-insensitively.
- Add unit test simulating ReverseProxy behavior to verify headers are not duplicated.',
        'https://github.com/labstack/echo/pull/2992'
    );

    -- Echo PR pair 14: #2989 creates the Change, #2988 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Documentation'),
        'chore: improve echo maintenance path',
        'Summary:
- Add or tighten focused edge-case tests or type assertions in ip_test.go, json_test.go related to CI, Docker, Kubernetes, build tooling, release automation; avoid docs-only changes and broad refactors.
- Keep the change narrow so it is straightforward to review.

Notes:
- I kept this scoped to the relevant implementation and tests.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Spotted while reading binder.go. The doc comments on `MustUnixTime`, `MustUnixTimeMilli`, `MustUnixTimeNano` all say "bind to time.Duration variable" but the function signature is `dest *time.Time` and the non-Must variants directly above each one correctly say "binds parameter to time.Time variable". Looks like a copy-paste from the actual `MustDuration` doc that never got updated.

While there, dropped a stray double space and changed "nano second" to "nanosecond" on the Nano variant. No code change.',
        'https://github.com/labstack/echo/pull/2989'
    );

    -- Echo PR pair 15: #2986 creates the Change, #2985 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Maintenance'),
        'fix: inherit parent RouteNotFound handler for groups with middleware',
        '## Summary
- When `Group.Use()` registers catch-all RouteNotFound routes for middleware execution, inherit the most specific parent RouteNotFound handler from the router tree
- Fall back to the default `NotFoundHandler` only when no parent handler exists

Fixes #2485

## Problem
Groups created with middleware always registered the default JSON `NotFoundHandler` for 404 cases, shadowing a custom root `RouteNotFound` handler.

## Test plan
- [x] `go test ./...`
- [x] Added `TestGroupRouteNotFoundFallsBackToRootHandler` from the issue reproducer
- [x] Updated `TestGroup_RouteNotFoundWithMiddleware` expectations for inherited handlers'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        '## Summary

Fixes #2961

Rate limit middleware now sets standard rate limit response headers when using the built-in `RateLimiterMemoryStore`:

- `X-RateLimit-Limit` — configured burst
- `X-RateLimit-Remaining` — tokens remaining after the request
- `Retry-After` — seconds until the next token is available (on 429 only)

Implementation follows the maintainer suggestion from the issue: an optional unexported `rateLimiterStoreContext` interface with `AllowContext(c echo.Context, identifier string) (bool, error)`. When the configured store implements it, the middleware calls it instead of `Allow`. Custom stores can opt in without breaking the existing `RateLimiterStore` API.

`Retry-After` is computed via `rate.Limiter.ReserveN(...).Delay()` as suggested in the issue discussion.

## Test plan

- [x] `go test ./middleware -run TestRateLimiter`
- [x] `go test -race ./middleware -run TestRateLimiterMemoryStore`
- [x] New test `TestRateLimiterMemoryStore_AllowContext_SetsHeaders` verifies headers on allowed and denied requests',
        'https://github.com/labstack/echo/pull/2986'
    );

    -- Echo PR pair 16: #2984 creates the Change, #2983 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Router'),
        'fix(middleware): keep handler 404 responses with Static HTML5 mode',
        '## Summary

- Restrict Static middleware HTML5 SPA fallback to router-level 404 responses
- Skip index.html fallback when a matched handler returns its own `404` (e.g. API resource not found)
- Add regression test covering `/api/test/3` JSON 404 vs `/client-route` SPA index fallback

## Test plan

- [x] `go test ./middleware/...`
- [x] `TestStaticHTML5DoesNotOverrideHandler404`

Fixes #2775'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        '## Summary

- For simple (non-preflight) CORS requests, apply CORS response headers after the handler runs
- Skip setting `Access-Control-Allow-Origin` and related headers when an upstream handler (e.g. reverse proxy) already set them
- Add regression test covering chained CORS + reverse proxy setup from #2853

## Test plan

- [x] `go test ./middleware/...`
- [x] `TestCORSNoDuplicateHeadersFromUpstream` — proxy layer + upstream both use CORS middleware, response has single `Access-Control-Allow-Origin` and `Vary`

Fixes #2853',
        'https://github.com/labstack/echo/pull/2984'
    );

    -- Echo PR pair 17: #2982 creates the Change, #2979 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Middleware'),
        'fix(bind): restore []string values for map[string]interface{} duplicates',
        '## Summary
- Fix regression when binding multipart form data with duplicate field names to `map[string]interface{}`
- Store a single value as `string` and multiple values as `[]string`

## Problem
Since v4.13.0, binding duplicate multipart fields like two `ima_slice` values to `map[string]interface{}` only kept the first value. Applications expecting a slice silently broke.

This regressed after #2656, which intended to preserve pre-v4.12.0 single-string behavior but always bound `v[0]`.

## Fix
For `map[string]interface{}` / `map[string]any`:
- one value → `string`
- multiple values → `[]string`

## Test plan
- [x] Updated `TestDefaultBinder_bindDataToMap`
- [x] Added `TestBindMultipartFormToMapInterface`
- [x] `go test ./... -run ''TestDefaultBinder_bindDataToMap|TestBindMultipartFormToMapInterface''`

Fixes #2731'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Fixes typos in CSRFConfig comments so they reference the actual exported field names:

- TrustedOrigin -> TrustedOrigins
- AllowSecFetchSameSite -> AllowSecFetchSiteFunc
- CRSF -> CSRF

Also clarifies the trusted origin wording.

No behavior changes.',
        'https://github.com/labstack/echo/pull/2982'
    );

    -- Echo PR pair 18: #2977 creates the Change, #2973 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Binder'),
        'Fix proxy panic when balancer has no targets',
        'Fixes #2976

This fixes a panic when a proxy balancer returns no target.

Built-in balancers can return `nil` when the target list is empty, for example after `RemoveTarget` removes the last target. `ProxyWithConfig` now returns `502 Bad Gateway` through the configured error handler instead of passing the nil target to `proxyHTTP` or `proxyRaw`.

I also added a test for a custom balancer returning a target with a nil URL.

Tests pass with `go test ./... -count=1`.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Enables zero-copy (sendfile) serving. Disabled when ''After'' hooks are present to maintain backward compatibility.

fix #2725
-----

```
Environment
   - OS: darwin
   - Arch: amd64
   - CPU: Intel(R) Core(TM) i7-9750H CPU @ 2.60GHz
   - Go: goos: darwin, goarch: amd64

  Benchmark Results (100MB File)

   1 BenchmarkContext_File_RealServer/Zero-Copy-Optimized-12       32718676 ns/op    3204.82 MB/s
   2 BenchmarkContext_File_RealServer/User-Space-Standard-12       40801873 ns/op    2569.92 MB/s
```',
        'https://github.com/labstack/echo/pull/2977'
    );

    -- Echo PR pair 19: #2971 creates the Change, #2970 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Documentation'),
        'Update GitHub actions deps versions',
        'Closed Echo pull request #2971 did not include a body.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Modernizes the codebase using the Go 1.26 gofix tool to adopt newer idioms and library features while maintaining compatibility with the current toolchain.',
        'https://github.com/labstack/echo/pull/2971'
    );

    -- Echo PR pair 20: #2969 creates the Change, #2966 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Maintenance'),
        'refactor: replace Split in loops with more efficient SplitSeq',
        'strings.SplitSeq (introduced in Go 1.23)  returns a lazy sequence (strings.Seq), allowing gopher to iterate over tokens one by one without creating an intermediate slice.

It significantly reduces memory allocations and can improve performance for long strings.

More info: https://github.com/golang/go/issues/61901'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'In Go 1.21, the standard library includes built-in [max/min](https://pkg.go.dev/builtin@go1.21.0#max) function, which can greatly simplify the code.',
        'https://github.com/labstack/echo/pull/2969'
    );

    -- Echo PR pair 21: #2965 creates the Change, #2964 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Router'),
        'Changelog for v5.1.1',
        'Closed Echo pull request #2965 did not include a body.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Context.Json should not unwrap response and just wrap Response so other middlewares can use their own "wrapping" Responses and see the status code.

I found this during #2895 when to tried to create middleware that wraps existing response to own and status code setting did not work anymore with `Context.JSON` (always sends 200 to client).',
        'https://github.com/labstack/echo/pull/2965'
    );

    -- Echo PR pair 22: #2963 creates the Change, #2962 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Middleware'),
        'Changelog for v4.15.2',
        'changelog'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Context.Scheme should validate values taken from header

Backport PR #2953 (d1d8ad3f99dd9b80542cd0c357d56a8916c515df) to `v4`',
        'https://github.com/labstack/echo/pull/2963'
    );

    -- Echo PR pair 23: #2958 creates the Change, #2953 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Binder'),
        'chore: fix typos in httperror.go',
        'Closed Echo pull request #2958 did not include a body.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Relates to: https://github.com/labstack/echo/issues/2952',
        'https://github.com/labstack/echo/pull/2958'
    );

    -- Echo PR pair 24: #2951 creates the Change, #2946 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Documentation'),
        'docs: fix README command and slog reference',
        '**Repo:** labstack/echo (⭐ 30000)
**Type:** docs
**Files changed:** 1
**Lines:** +3/-3

## What
Updated `README.md` to fix three user-facing documentation issues: the installation snippet now uses a valid shell comment marker inside the `sh` code block, the `slog-echo` entry now links to the current standard-library `log/slog` package documentation instead of the retired `x/exp/slog` path, and the third-party middleware guidance sentence has a small grammar correction.

## Why
These changes remove small but real sources of friction in the first-touch documentation. The `//` line inside a shell block is misleading if copied verbatim, and the outdated `slog` link points readers at a pre-standard-library package path. The grammar fix keeps contributor-facing guidance polished without changing behavior.

## Testing
No code tests were run because this is a README-only change. Verification was done by inspecting the rendered diff and ensuring the updated commands and links are syntactically correct.

## Risk
Low / Documentation-only change with no runtime impact.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Adds a new middleware that mounts a JSON-RPC 2.0 endpoint at a configurable path and auto-exposes registered Echo routes as MCP tools, so AI clients (Claude Desktop, Cursor, etc.) can discover and call them.

Implements initialize, tools/list and tools/call. Tool input schemas are derived from RouteInfo.Parameters; tool calls substitute path parameters via RouteInfo.Reverse and dispatch the synthesized request through e.ServeHTTP, preserving the full middleware chain.

No core Echo files are modified and no new dependencies are introduced.

Fixes #2935',
        'https://github.com/labstack/echo/pull/2951'
    );

    -- Echo PR pair 25: #2945 creates the Change, #2944 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Maintenance'),
        'feat(middleware): add MCP (Model Context Protocol) middleware',
        'Adds a new middleware that mounts a JSON-RPC 2.0 endpoint at a configurable path and auto-exposes registered Echo routes as MCP tools, so AI clients (Claude Desktop, Cursor, etc.) can discover and call them.

Implements initialize, tools/list and tools/call. Tool input schemas are derived from RouteInfo.Parameters; tool calls substitute path parameters via RouteInfo.Reverse and dispatch the synthesized request through e.ServeHTTP, preserving the full middleware chain.

No core Echo files are modified and no new dependencies are introduced.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        '## What this does

This adds an **optional `AutoHead` flag** that automatically enables HEAD requests for any GET route. No need to define separate HEAD handlers anymore.

## Why this is useful

Right now, if you want proper HEAD support, you have to manually add a HEAD route for every GET route. That leads to:

* Repeated boilerplate
* Easy-to-miss HEAD routes (resulting in 405 errors)
* Inconsistent headers like missing `Content-Length`

According to HTTP spec (RFC 7231), HEAD should behave like GET without a body — so this just makes Echo handle that for you.

## How it works

When `AutoHead` is turned on:

* Every GET route automatically gets a HEAD equivalent
* The same handler runs, but the response body is suppressed
* Headers and status code are preserved
* `Content-Length` is still set correctly
* If you’ve defined a HEAD route manually, it won’t be overridden

## Implementation notes

* Added `AutoHead` to `Echo` and `Config`
* Wrapped handlers using a custom response writer
* Hooked into route registration (`add()`) to register HEAD routes

## Usage

```go
e := echo.New()
e.AutoHead = true

e.GET("/api/users", handler)
// HEAD /api/users now works automatically
```

## Performance

Very small overhead:

* ~532 ns per request
* 3 allocations

No impact if the feature is disabled (default).

## Testing

Covers:

* Enabled vs disabled behavior
* Explicit HEAD route priority
* Middleware compatibility

All tests pass.

---

## Compatibility

* No breaking changes
* Fully opt-in
* Existing code works as-is
#2895 @markbates @aldas',
        'https://github.com/labstack/echo/pull/2945'
    );

    -- Echo PR pair 26: #2941 creates the Change, #2937 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Router'),
        'fix(lint): resolve staticcheck issues and improve code quality',
        'This PR resolves several `staticcheck` lint issues, got the report from  `golangci-lint`.

### Changes

* Applied De Morgan’s law to simplify boolean expressions
* Replaced `fmt.Sprintf` with `fmt.Fprintf` to avoid unnecessary allocations
* Removed redundant embedded field access
* Improved overall code readability and idiomatic Go usage

### Impact

* No functional changes
* Minor performance improvements
* Cleaner and more maintainable code

### Notes

All changes were verified with:

```bash
golangci-lint run
```

there is no remaining lint issues.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'the behavior is opt-out.

I created one flag in both Echo and Group struct (since they are related to the register of new routes) that is private and a function to explicit cancel this behavior.

Why: Mentioned in the issue #2895 I searched and saw that the default behavior in many frameworks is to automatically register a head request with GET, so I agree with the author of the issue that it should be included to guarantee an expected behavior from the programmer.

I added tests and only modified the high level functions, if the author think it is good and relevant enough to be merged, it will be good.',
        'https://github.com/labstack/echo/pull/2941'
    );

    -- Echo PR pair 27: #2936 creates the Change, #2934 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Middleware'),
        'Make StartConfig listener creation context-aware',
        '## Summary
- Create listeners with net.ListenConfig so StartConfig respects the provided context during listener setup
- Keep the existing serving behavior by defaulting ListenerNetwork to `tcp` and wrapping TLS listeners with
`tls.NewListener`
- simplify the listener creation path by using the same flow for TLS and non-TLS listeners

## Benefit
`StartConfig.Start` and `StartConfig.StartTLS` already accept a context, but listener creation previously used `net.Listen`
and `tls.Listen`, which do not use that context. This change makes listener setup context-aware without changing how the
server behaves after the listener is created.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Closed Echo pull request #2934 did not include a body.',
        'https://github.com/labstack/echo/pull/2936'
    );

    -- Echo PR pair 28: #2933 creates the Change, #2932 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Binder'),
        'Remove legacy IP extraction logic from context.RealIP method',
        'This change does not break the API contract, but it does introduce breaking changes in logic/behavior.  But as promised - 31.03.2026 will be last day for potentially breaking stuff.  I want to get this security related thing done.

-------------

Remove legacy IP extraction logic from context.RealIP method and move it to LegacyIPExtractor IP extraction function.

`v4` behavior can be restored with:
```go
e := echo.New()
e.IPExtractor = echo.LegacyIPExtractor()
```

but you should instead with proper trust options
- https://pkg.go.dev/github.com/labstack/echo/v5#ExtractIPFromRealIPHeader
- https://pkg.go.dev/github.com/labstack/echo/v5#ExtractIPFromXFFHeader

For example:
 ```go
_, lbIPRange, _ := net.ParseCIDR("203.0.113.199/24")
e.IPExtractor = echo.ExtractIPFromXFFHeader(
	echo.TrustLinkLocal(false),
	echo.TrustIPRange(lbIPRange),
)
 ```

Read https://echo.labstack.com/docs/ip-address'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Relates to https://github.com/labstack/echox/issues/397 and https://github.com/labstack/echo/issues/2918

we did not set in `v4` and it causing problems for users',
        'https://github.com/labstack/echo/pull/2933'
    );

    -- Echo PR pair 29: #2931 creates the Change, #2930 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Documentation'),
        'Add NewDefaultFS function to help create filesystem that allows absolute paths',
        'Add `NewDefaultFS` function to help create filesystem that allows absolute paths with `Open` method.

fs.FS does not like paths like `/tmp/temp.file` even if fs is created like `os.DirFS("/"). You need to remove leading slash.

Also when file is absolute (`/tmp/temp.file`) and has same prefix as filesystem  `os.DirFS("/tmp/")` this would not work. Echo `defaultFs` was working similarly to `os.Open` and therefore allowed absolute paths.

This PR makes `echo.defaultFs` to accept absolute path filenames in Open when it matches the filesystem prefix.

Relates to https://github.com/labstack/echo/issues/2929'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Relates to: https://github.com/labstack/echo/issues/2924

NB: it does not fix couple of staticcheck problems that are being reported
```bash
[x@x echo]$ golangci-lint run
bind.go:158:6: QF1001: could apply De Morgan''s law (staticcheck)
                if !(isElemSliceOfStrings || isElemString || isElemInterface) {
                   ^
middleware/basic_auth.go:124:8: QF1001: could apply De Morgan''s law (staticcheck)
                                if !(len(auth) > l+1 && strings.EqualFold(auth[:l], basic)) {
                                   ^
middleware/static.go:253:8: QF1001: could apply De Morgan''s law (staticcheck)
                                if !(errors.As(err, &he) && config.HTML5 && he.StatusCode() == http.StatusNotFound) {
                                   ^
route.go:88:4: QF1012: Use fmt.Fprintf(...) instead of WriteString(fmt.Sprintf(...)) (staticcheck)
                        uri.WriteString(fmt.Sprintf("%v", pathValues[n]))
                        ^
router.go:998:30: QF1008: could remove embedded field "RouteInfo" from selector (staticcheck)
                rPath = matchedRouteMethod.RouteInfo.Path
```',
        'https://github.com/labstack/echo/pull/2931'
    );

    -- Echo PR pair 30: #2928 creates the Change, #2925 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Maintenance'),
        'Add doc comments to clarify usage of File related methods and leading slash handling',
        'Add doc comments to clarify usage of File related methods and leading slash handling

Relates to https://github.com/labstack/echo/issues/2922#issuecomment-4150975101

Example:
```go
package main

import (
	"embed"

	"github.com/labstack/echo/v5"
)

//go:embed dist/*
var efs embed.FS

func main() {
	e := echo.New()
	e.Filesystem = efs

	e.File("/test", "dist/private.txt") // <--- file path must not have a leading slash

	if err := e.Start(":8080"); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}
}

```

and
```go
package main

import (
	"os"

	"github.com/labstack/echo/v5"
)

func main() {
	e := echo.New()
	e.Filesystem = os.DirFS("/")

	e.File("/test.jpg", "opt/app/assets/test.jpg") // <--- file path must not have a leading slash

	if err := e.Start(":8080"); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}
}
```'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'The documentation for `NewRateLimiterMemoryStore` and
`NewRateLimiterMemoryStoreWithConfig` states that the default Burst
value is the "rounded down" value of the rate. This was accurate when
the documentation was added in #2366, where the code used `int(config.Rate)`.

However, #2852 changed the default burst calculation to use `math.Ceil`,
making it the rounded up value. The documentation was not updated to
reflect this change.',
        'https://github.com/labstack/echo/pull/2928'
    );

    -- Echo PR pair 31: #2921 creates the Change, #2920 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Router'),
        'fix: prefer exact path match over parameterized match for method checking',
        '## Summary
- Fix route matching to prefer exact path matches over parameterized matches when checking HTTP methods
- Previously, a POST to `/VerifiedCallerId/Verification/` would match `GET VerifiedCallerId/:phone_number` and return 405

## Root Cause
The router matches parameterized routes before checking for exact matches at the same path level. This causes incorrect 405 responses when an exact match exists for a different method.

## Testing
- Added test case for the reported scenario

Fixes #2547

Made with [Cursor](https://cursor.com)'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Add StartConfig.Listener so server with custom Listener is easier to create

relates to https://github.com/labstack/echo/issues/2918#issuecomment-4089341521
https://github.com/labstack/echo/issues/1942

Example:
```go
package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v5"
)

func main() {
	e := echo.New()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	lc := net.ListenConfig{
		KeepAlive: 15 * time.Second,
		//Control:   nil,
		//KeepAliveConfig: net.KeepAliveConfig{
		//	Enable:   false,
		//	Idle:     0,
		//	Interval: 0,
		//	Count:    0,
		//},
	}
	l, err := lc.Listen(ctx, "tcp", ":8080")
	if err != nil {
		e.Logger.Error("failed to create listener", "error", err)
		return
	}
	defer l.Close()

	sc := echo.StartConfig{
		Listener: l,
	}
	if err := sc.Start(ctx, e); err != nil {
		e.Logger.Error("failed to run server", "error", err)
	}
}

```',
        'https://github.com/labstack/echo/pull/2921'
    );

    -- Echo PR pair 32: #2919 creates the Change, #2917 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Middleware'),
        'Add https://github.com/labstack/echo-prometheus to the middleware list in README.md',
        'Add https://github.com/labstack/echo-prometheus to the middleware list in README.md'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        '## Summary

Fixes #2485

When a route exists in the router but the HTTP method is not allowed, the router previously always fell back to `methodNotAllowedHandler`. However, this bypassed any `RouteNotFound` handler registered at a parent or root group level.

## Root Cause

In `DefaultRouter.Route()`, when `currentNode.isHandler` is true (path matched but method not), the code immediately set `rInfo = methodNotAllowedRouteInfo` without checking if any parent node had a `notFoundHandler` registered.

## Fix

Traverse the parent node chain to look for a `notFoundHandler`. If found, use it; otherwise fall back to the existing `methodNotAllowedHandler` behavior.

## Test

Existing route tests pass. The fix ensures `RouteNotFound` handlers registered at a group level are properly invoked for sub-paths.

Signed-off-by: lyydsheep <2230561977@qq.com>',
        'https://github.com/labstack/echo/pull/2919'
    );

    -- Echo PR pair 33: #2916 creates the Change, #2910 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Binder'),
        'fix: correct spelling mistakes in comments and field name',
        'Fix multiple typos found across the codebase:

- `response.go`: rename unexported field `commited` to `committed` and fix related comments
- `server.go`: fix `addres` to `address` in comment
- `middleware/csrf.go`: fix `formated` to `formatted` in comment
- `middleware/request_logger.go`: fix `commited` to `committed` in comment
- `echotest/context.go`: fix `wil` to `will` in comments (2 occurrences)

All changes are either in comments or in an unexported (internal) struct field name. No behavioral changes.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Fix for issue: #2761

This PR addresses #2761 by introducing SkipMiddlewareOnNotFound. This allows developers to avoid executing heavy global middleware (Auth, DB logging) for requests that result in a 404 or 405, improving performance and reducing log noise

**Testing with below cmd saves 3 allocations.**

```bash
go test -bench=BenchmarkMiddleware404 -benchmem
```
Sample output:
```
BenchmarkMiddleware404/Normal_404-8             {"time":"2026-03-04T22:00:36.5728429Z","level":"ERROR","msg":"failed to shut down server within given timeout","error":"context deadline exceeded"}
 1416462               890.7 ns/op           894 B/op         10 allocs/op
BenchmarkMiddleware404/Optimized_404-8           1000000              1472 ns/op             515 B/op          7 allocs/op
PASS
ok      github.com/labstack/echo/v5     6.217s
```

10-7 = 3 allocations saved for faster runtime',
        'https://github.com/labstack/echo/pull/2916'
    );

    -- Echo PR pair 34: #2908 creates the Change, #2905 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Documentation'),
        'Add echo-opentelemetry to the README.md',
        'Add https://github.com/labstack/echo-opentelemetry to the middleware list in README.md'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'CSRF: support older token-based CSRF protection handler that want to render token into template

(cherry picked from commit 9183f1e80901fe3e55a61fce607e2c925e1e987b)
same thing in `v5` https://github.com/labstack/echo/pull/2894


relates to:
- https://github.com/labstack/echo/issues/2874
- https://github.com/labstack/echo/pull/2903',
        'https://github.com/labstack/echo/pull/2908'
    );

    -- Echo PR pair 35: #2903 creates the Change, #2902 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Maintenance'),
        'Fix CSRF middleware to set token in context when Sec-Fetch-Site validation passes',
        '## Summary

Fixes #2874 - CSRF middleware now correctly sets the token in context and cookie even when Sec-Fetch-Site validation passes, allowing handlers to render forms with CSRF tokens.

## Problem

In v4.15.0, when `checkSecFetchSiteRequest()` returns `(true, nil)` (e.g., for direct URL navigation with `Sec-Fetch-Site: none`), the middleware calls `return next(c)` without setting the CSRF token in context. This breaks handlers that need the token to render forms.

**Reproduction:**
```bash
curl -H "Sec-Fetch-Site: none" https://example.com/users/register
# Returns 500 "CSRF token not found" in v4.15.0
```

## Solution

Move token generation/retrieval before the Sec-Fetch-Site check and set the token in context and cookie even when the request is deemed "safe". This ensures handlers can always access the CSRF token for form rendering while still skipping token validation for safe requests.

## Changes

- Token generation/retrieval now happens before Sec-Fetch-Site validation
- When a request passes Sec-Fetch-Site validation, the middleware now:
  - Sets the token in context via `c.Set(config.ContextKey, token)`
  - Sets the CSRF cookie with proper expiration and security flags
  - Adds the `Vary: Cookie` header to prevent caching issues
- Token validation is still skipped for safe requests (no behavior change)

## Test Plan

- [x] All existing CSRF tests pass
- [x] Added `TestCSRF_SecFetchSite_SetsTokenInContext` with 4 test cases covering:
  - GET requests with `Sec-Fetch-Site: none`
  - GET requests with `Sec-Fetch-Site: same-origin`
  - POST requests with `Sec-Fetch-Site: none`
  - POST requests with `Sec-Fetch-Site: same-origin`
- [x] All tests verify that:
  - Token is set in context
  - CSRF cookie is set
  - Token in context matches cookie value

## Impact

This fix restores the ability to render server-side forms with CSRF tokens when users access pages via direct browser navigation (typing URL, bookmarks, external links). The `Sec-Fetch-Site: none` header is automatically sent by all modern browsers in these scenarios.

## Backward Compatibility

This change is backward compatible - it only adds functionality (setting token in context) that was missing, without changing any existing validation logic.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        '## Summary

Fixes issue #2853 - CORS middleware was duplicating headers when multiple Echo services with CORS middleware were chained (e.g., Service A proxies to Service B, both with `middleware.CORS` enabled).

## Changes

- Added check to detect existing `Access-Control-Allow-Origin` headers in responses (indicating the request was proxied from an upstream service that already applied CORS)
- When CORS headers are already present, the middleware now skips re-applying them to prevent duplication
- Updated `Vary` header handling to check if values already exist before adding them, preventing duplicate Vary entries
- Added comprehensive test cases for proxy chain scenarios (both regular and preflight requests)

## Test Plan

- [x] All existing CORS tests pass
- [x] Added `TestCORSProxyChain` to verify headers are not duplicated in proxy scenarios
- [x] Added `TestCORSProxyChainPreflight` to verify preflight requests in proxy chains
- [x] Verified that the fix prevents duplicate `Access-Control-Allow-Origin` and `Vary` headers

## Reproduces Issue

This fix addresses the exact scenario described in #2853 where multiple proxy layers each independently apply CORS headers, causing accumulation.

**Before**: Multiple CORS middlewares in a chain would each add headers, resulting in duplicates
**After**: Middleware detects existing CORS headers and skips processing, preventing duplication',
        'https://github.com/labstack/echo/pull/2903'
    );

    -- Echo PR pair 36: #2901 creates the Change, #2900 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Router'),
        'Add changelog for v5.0.4 release',
        'Closed Echo pull request #2901 did not include a body.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Add `ResolveResponseStatus` function to help middleware/handlers determine HTTP status code and echo.Response.

Loggers and tracing middlewares need to determine status code from either from error or `echo.Response`.  Also - response is needed often for knowing response size from `echo.Response.Size`.  so this function tries to shorten these 2 requirements.

Relates to https://github.com/labstack/echo-contrib/pull/141
and https://github.com/labstack/echo-contrib/blob/master/internal/helpers/statuscode.go


also https://github.com/labstack/echo/pull/2892',
        'https://github.com/labstack/echo/pull/2901'
    );

    -- Echo PR pair 37: #2899 creates the Change, #2898 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Middleware'),
        'add Go 1.26 to CI flow',
        'https://go.dev/doc/go1.26

+ update security.md'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'After `http.Server.Serve` returns we need to wait for graceful shutdown goroutine to finish because when application exits immediately there are no graceful shutdown.

Fixes: https://github.com/labstack/echo/issues/2897',
        'https://github.com/labstack/echo/pull/2899'
    );

    -- Echo PR pair 38: #2896 creates the Change, #2894 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Binder'),
        'Update location of oapi-codegen in README',
        'Closed Echo pull request #2896 did not include a body.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'I though I already merged this.  I think  https://github.com/labstack/echo/pull/2876 got closed when I purged all old branches at my fork. I should not have deleted that branch as it was not merged yet

-------------

In case CSRF flow is using `Sec-Fetch-Site` header but there is form still wanting to render CSRF token fields into form we  can help them by setting dummy value to context that atleast something can be rendered into form. As we already know that this browser is able to send `Sec-Fetch-Site` header, we do not need to generate token value or deal with cookies.',
        'https://github.com/labstack/echo/pull/2896'
    );

    -- Echo PR pair 39: #2893 creates the Change, #2892 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Documentation'),
        'Fix CSRF middleware not setting token when Sec-Fetch-Site passes',
        '## Summary

Fixes #2874

When `checkSecFetchSiteRequest()` returns `(true, nil)` — e.g. for direct URL navigation where `Sec-Fetch-Site: none`, or same-origin requests — the CSRF middleware immediately calls `return next(c)` **without**:

1. Generating or retrieving the CSRF token
2. Setting the CSRF cookie
3. Storing the token in context via `c.Set(config.ContextKey, token)`
4. Adding the `Vary: Cookie` response header

This breaks all server-rendered forms that use `c.Get("csrf")` to embed a CSRF token in HTML for subsequent POST requests. The bug is triggered by every modern browser during direct navigation (typing a URL, clicking a bookmark, or following an external link), since browsers automatically send `Sec-Fetch-Site: none` in these scenarios.

## Changes

- **`middleware/csrf.go`**: Extract token generation and cookie/context setting into a `setTokenInContext` helper closure, and call it in the `Sec-Fetch-Site` allow path before calling `next(c)`. The existing legacy token path remains unchanged.
- **`middleware/csrf_test.go`**:
  - Add `expectCookieContains` assertion to the existing `SecFetchSite=same-origin` test case
  - Add two new table-driven test cases verifying cookie is set for `Sec-Fetch-Site: none` and `same-origin` GET requests
  - Add a dedicated `TestCSRF_SecFetchSiteSetsTokenInContext` regression test that verifies the token is accessible in context, the cookie is set, the `Vary` header is correct, and existing cookie tokens are reused

## Test plan

- [x] All existing CSRF tests pass (`go test ./middleware/ -run TestCSRF -v`)
- [x] New regression tests pass for the exact scenario from the issue
- [x] Full test suite passes with race detector (`go test -race ./...`)
- [x] Verified the fix handles both new token generation and reuse of existing cookie tokens

🤖 Generated with [Claude Code](https://claude.com/claude-code)'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        '## Overview
Implemented the `Is` method on the `HTTPError` struct, enabling error checking using `errors.Is` (particularly for comparing with sentinel errors).

## Background
Starting with Go 1.13, `errors.Is` is recommended for error checking, but Echo''s `HTTPError` did not previously implement the `Is` method. Echo''s predefined errors (such as `echo.ErrNotFound`) are of the internal `httpError` type, while errors created with `NewHTTPError` are of type `*HTTPError`. Because these are distinct types in Go''s type system, `errors.Is(err, echo.ErrNotFound)` would return false if `err` was of type `*HTTPError`, making intended error handling difficult. This change resolves the issue by adding logic that considers errors with matching status codes to be the same error.

## Changes
- `httperror.go`: Added an `Is` method to the `HTTPError` struct. When comparing against `*HTTPError` or `*httpError`, it compares the status code.
- `httperror_test.go`: Added a test case for the `Is` method (`TestHTTPError_Is`).',
        'https://github.com/labstack/echo/pull/2893'
    );

    -- Echo PR pair 40: #2891 creates the Change, #2889 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Maintenance'),
        'Fix staticmw',
        'Fix directory traversal vulnerability under Windows in Static middleware when default Echo filesystem is used. Reported by @shblue21.

This applies to cases when:
- Windows is used as OS
- `middleware.StaticConfig.Filesystem` is `nil` (default)
- `echo.Filesystem` is has not been set explicitly (default)'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Closed Echo pull request #2889 did not include a body.',
        'https://github.com/labstack/echo/pull/2891'
    );

    -- Echo PR pair 41: #2888 creates the Change, #2887 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Router'),
        'Changelog for version 5.0.2',
        'Closed Echo pull request #2888 did not include a body.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'fix Static middleware listing all files from given filesystem root when browser=true

fixes: https://github.com/labstack/echo/issues/2886',
        'https://github.com/labstack/echo/pull/2888'
    );

    -- Echo PR pair 42: #2885 creates the Change, #2881 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Middleware'),
        'Fill c.Request().Pattern field with route path to help standard library based middlewares',
        'Fill c.Request().Pattern field with route path to help standard library based middlewares.  For example Otel standard library  Request/Response field extration uses `Request.Pattern` as route.

Relates to https://github.com/labstack/echo-contrib/pull/141'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'The repository lacked proper documentation for security vulnerability reporting. The existing SECURITY.md only stated "look for maintainers email(s) in commits and email them."

## Changes

- **Primary reporting method**: Added instructions for GitHub Private Vulnerability Reporting with direct link to Security tab
- **Fallback contact**: Listed all current maintainers with GitHub profile links for email-based reporting
- **Reporter guidance**: Added sections for what to include in reports and response time expectations (48h acknowledgment, 7d detailed response)
- **Security process**: Documented vulnerability handling workflow
- **Version support**: Fixed table formatting and clarified supported versions (`>= 4.15.x` vs previous ambiguous `> 4.15.x`)

The updated policy provides a professional disclosure pathway while GitHub Private Vulnerability Reporting awaits admin enablement in repository settings.

<!-- START COPILOT ORIGINAL PROMPT -->



<details>

<summary>Original prompt</summary>

>
> ----
>
> *This section details on the original issue you should resolve*
>
> <issue_title>Enable GitHub Private Vulnerability Reporting</issue_title>
> <issue_description>Could you please enable GitHub Private Vulnerability Reporting for this repository?
> https://docs.github.com/en/code-security/how-tos/report-and-fix-vulnerabilities/configure-vulnerability-reporting/configuring-private-vulnerability-reporting-for-a-repository
>
> This will allow users and contributors to open private issues or advisories regarding security fixes. These issues are private to maintainers and reporters until published. There is currently not a SECURITY.md or security contact present in the documentation.
>
> </issue_description>
>
> ## Comments on the Issue (you are @copilot in this section)
>
> <comments>
> <comment_new><author>@aldas</author><body>
> @vishr , I do not see these [options](https://docs.github.com/en/code-security/how-tos/report-and-fix-vulnerabilities/configure-vulnerability-reporting/configuring-private-vulnerability-reporting-for-a-repository) under settings, so probably only you can enable this.</body></comment_new>
> <comment_new><author>@aldas</author><body>
> @wodzen  I created the most basic security.md that can be. At the moment if you have something, look for my or @vishr email. For example in verified commits and email us.</body></comment_new>
> </comments>
>


</details>



<!-- START COPILOT CODING AGENT SUFFIX -->

- Fixes labstack/echo#2879

<!-- START COPILOT CODING AGENT TIPS -->
---

✨ Let Copilot coding agent [set things up for you](https://github.com/labstack/echo/issues/new?title=✨+Set+up+Copilot+instructions&body=Configure%20instructions%20for%20this%20repository%20as%20documented%20in%20%5BBest%20practices%20for%20Copilot%20coding%20agent%20in%20your%20repository%5D%28https://gh.io/copilot-coding-agent-tips%29%2E%0A%0A%3COnboard%20this%20repo%3E&assignees=copilot) — coding agent works faster and does higher quality work when set up for your repo.',
        'https://github.com/labstack/echo/pull/2885'
    );

    -- Echo PR pair 43: #2880 creates the Change, #2878 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Binder'),
        'Changelog for v5.0.1 release',
        'Closed Echo pull request #2880 did not include a body.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Hi maintainers,
Just a quick doc fix about the DenyHandler provided example.

Re-opened from #2864',
        'https://github.com/labstack/echo/pull/2880'
    );

    -- Echo PR pair 44: #2877 creates the Change, #2876 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Documentation'),
        'Context: json should not send status code before serialization is complete',
        'Context: json should not send status code before serialization is complete

Relates to https://github.com/labstack/echo/pull/2866#issuecomment-3778036694'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'CSRF: support older token-based CSRF protection handler that want torender token into template',
        'https://github.com/labstack/echo/pull/2877'
    );

    -- Echo PR pair 45: #2875 creates the Change, #2871 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Maintenance'),
        'fix: validate Connection header in IsWebSocket() [per RFC 6455]',
        '## Description
This PR fixes the `IsWebSocket()` function to properly validate WebSocket upgrade requests according to RFC 6455 specification.

## Problem
The current implementation only checks the `Upgrade` header but ignores the `Connection` header requirement specified in RFC 6455 Section 1.3. A valid WebSocket upgrade request must have both headers present with specific values.

## Solution
Updated `IsWebSocket()` to validate both required headers:
- `Upgrade: websocket` (case-insensitive)
- `Connection: upgrade` (case-insensitive, may contain other values)

## Changes
- Modified `IsWebSocket()` in `context.go` to check both headers
- Updated `TestContext_IsWebSocket()` in `context_test.go` with additional test cases
- Added test case for missing/invalid `Connection` header

## Testing
All existing tests pass, including new test cases that verify:
- Valid WebSocket requests with both headers present
- Case-insensitive header matching
- Invalid requests missing the `Connection` header
- Invalid requests with `Connection: close`

## RFC Reference
[RFC 6455 - The WebSocket Protocol](https://tools.ietf.org/html/rfc6455#section-1.3)'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Panic middleware: will now return a custom PanicStackError with stack trace when config.DisablePrintStack is set to false.

relates to https://github.com/labstack/echo/issues/2869#issuecomment-3771782789',
        'https://github.com/labstack/echo/pull/2875'
    );

    -- Echo PR pair 46: #2868 creates the Change, #2866 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Router'),
        'Merge V5 to master branch',
        'See https://github.com/labstack/echo/discussions/2861

If you are using Linux you can migrate easier parts like that:
```bash
find . -type f -name "*.go" -exec sed -i ''s/ echo.Context/ *echo.Context/g'' {} +
find . -type f -name "*.go" -exec sed -i ''s/echo\/v4/echo\/v5/g'' {} +
```
or in your favorite IDE

Replace all:
1. ` echo.Context` -> ` *echo.Context`
2. `echo/v4` -> `echo/v5`'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        '##  Bug Fix

### Problem
The `json()` method in `context.go` (line 504) was inconsistent with other response methods in how it sets the HTTP status code.

**Current behavior:**
```go
func (c *context) json(code int, i any, indent string) error {
    c.writeContentType(MIMEApplicationJSON)
    c.response.Status = code  // ❌ Directly setting Status field
    return c.echo.JSONSerializer.Serialize(c, i, indent)
}
```

This approach directly sets the `Status` field instead of properly calling `WriteHeader()`, which bypasses header commitment and prevents warnings about header modifications after the status is set.

### Solution
Updated the method to use `c.response.WriteHeader(code)` for consistency with other response methods:

```go
func (c *context) json(code int, i any, indent string) error {
    c.writeContentType(MIMEApplicationJSON)
    c.response.WriteHeader(code)  // ✅ Properly calls WriteHeader
    return c. echo.JSONSerializer.Serialize(c, i, indent)
}
```

All other similar response methods already use `WriteHeader()`:
- `jsonPBlob()` - line 489:  `c.response.WriteHeader(code)`
- `xml()` - line 543: `c.response.WriteHeader(code)`
- `Blob()` - line 578: `c.response.WriteHeader(code)`
- `JSONPBlob()` - line 530: `c.response.WriteHeader(code)`',
        'https://github.com/labstack/echo/pull/2868'
    );

    -- Echo PR pair 47: #2864 creates the Change, #2860 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Middleware'),
        'docs: add missing err parameter to DenyHandler example',
        'Hi maintainers,
Just a quick doc fix about the DenyHandler provided example.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        '# Changelog

## v4.15.0 - 2026-01-01


**Security**

NB: **If your application relies on cross-origin or same-site (same subdomain) requests do not blindly push this version to production**


The CSRF middleware now supports the [**Sec-Fetch-Site**](https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Sec-Fetch-Site) header as a modern, defense-in-depth approach to [CSRF
protection](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html#fetch-metadata-headers), implementing the OWASP-recommended Fetch Metadata API alongside the traditional token-based mechanism.

**How it works:**

Modern browsers automatically send the `Sec-Fetch-Site` header with all requests, indicating the relationship
between the request origin and the target. The middleware uses this to make security decisions:

- **`same-origin`** or **`none`**: Requests are allowed (exact origin match or direct user navigation)
- **`same-site`**: Falls back to token validation (e.g., subdomain to main domain)
- **`cross-site`**: Blocked by default with 403 error for unsafe methods (POST, PUT, DELETE, PATCH)

For browsers that don''t send this header (older browsers), the middleware seamlessly falls back to
traditional token-based CSRF protection.

**New Configuration Options:**
- `TrustedOrigins []string`: Allowlist specific origins for cross-site requests (useful for OAuth callbacks, webhooks)
- `AllowSecFetchSiteFunc func(echo.Context) (bool, error)`: Custom logic for same-site/cross-site request validation

**Example:**
  ```go
  e.Use(middleware.CSRFWithConfig(middleware.CSRFConfig{
      // Allow OAuth callbacks from trusted provider
      TrustedOrigins: []string{"https://oauth-provider.com"},

      // Custom validation for same-site requests
      AllowSecFetchSiteFunc: func(c echo.Context) (bool, error) {
          // Your custom authorization logic here
          return validateCustomAuth(c), nil
          // return true, err  // blocks request with error
          // return true, nil  // allows CSRF request through
          // return false, nil // falls back to legacy token logic
      },
  }))
  ```
PR: https://github.com/labstack/echo/pull/2858

**Type-Safe Generic Parameter Binding**

* Added generic functions for type-safe parameter extraction and context access by @aldas in https://github.com/labstack/echo/pull/2856

  Echo now provides generic functions for extracting path, query, and form parameters with automatic type conversion,
  eliminating manual string parsing and type assertions.

  **New Functions:**
  - Path parameters: `PathParam[T]`, `PathParamOr[T]`
  - Query parameters: `QueryParam[T]`, `QueryParamOr[T]`, `QueryParams[T]`, `QueryParamsOr[T]`
  - Form values: `FormParam[T]`, `FormParamOr[T]`, `FormParams[T]`, `FormParamsOr[T]`
  - Context store: `ContextGet[T]`, `ContextGetOr[T]`

  **Supported Types:**
  Primitives (`bool`, `string`, `int`/`uint` variants, `float32`/`float64`), `time.Duration`, `time.Time`
  (with custom layouts and Unix timestamp support), and custom types implementing `BindUnmarshaler`,
  `TextUnmarshaler`, or `JSONUnmarshaler`.

  **Example:**
  ```go
  // Before: Manual parsing
  idStr := c.Param("id")
  id, err := strconv.Atoi(idStr)

  // After: Type-safe with automatic parsing
  id, err := echo.PathParam[int](c, "id")

  // With default values
  page, err := echo.QueryParamOr[int](c, "page", 1)
  limit, err := echo.QueryParamOr[int](c, "limit", 20)

  // Type-safe context access (no more panics from type assertions)
  user, err := echo.ContextGet[*User](c, "user")
  ```

PR: https://github.com/labstack/echo/pull/2856



**DEPRECATION NOTICE** Timeout Middleware Deprecated - Use ContextTimeout Instead

The `middleware.Timeout` middleware has been **deprecated** due to fundamental architectural issues that cause
data races. Use `middleware.ContextTimeout` or `middleware.ContextTimeoutWithConfig` instead.

**Why is this being deprecated?**

The Timeout middleware manipulates response writers across goroutine boundaries, which causes data races that
cannot be reliably fixed without a complete architectural redesign. The middleware:

- Swaps the response writer using `http.TimeoutHandler`
- Must be the first middleware in the chain (fragile constraint)
- Can cause races with other middleware (Logger, metrics, custom middleware)
- Has been the source of multiple race condition fixes over the years

**What should you use instead?**

The `ContextTimeout` middleware (available since v4.12.0) provides timeout functionality using Go''s standard
context mechanism. It is:

- Race-free by design
- Can be placed anywhere in the middleware chain
- Simpler and more maintainable
- Compatible with all other middleware

**Migration Guide:**

```go
// Before (deprecated):
e.Use(middleware.Timeout())

// After (recommended):
e.Use(middleware.ContextTimeout(30 * time.Second))
```

**Important Behavioral Differences:**

1. **Handler cooperation required**: With ContextTimeout, your handlers must check `context.Done()` for cooperative
   cancellation. The old Timeout middleware would send a 503 response regardless of handler cooperation, but had
   data race issues.

2. **Error handling**: ContextTimeout returns errors through the standard error handling flow. Handlers that receive
   `context.DeadlineExceeded` should handle it appropriately:

```go
e.GET("/long-task", func(c echo.Context) error {
    ctx := c.Request().Context()

    // Example: database query with context
    result, err := db.QueryContext(ctx, "SELECT * FROM large_table")
    if err != nil {
        if errors.Is(err, context.DeadlineExceeded) {
            // Handle timeout
            return echo.NewHTTPError(http.StatusServiceUnavailable, "Request timeout")
        }
        return err
    }

    return c.JSON(http.StatusOK, result)
})
```

3. **Background tasks**: For long-running background tasks, use goroutines with context:

```go
e.GET("/async-task", func(c echo.Context) error {
    ctx := c.Request().Context()

    resultCh := make(chan Result, 1)
    errCh := make(chan error, 1)

    go func() {
        result, err := performLongTask(ctx)
        if err != nil {
            errCh <- err
            return
        }
        resultCh <- result
    }()

    select {
    case result := <-resultCh:
        return c.JSON(http.StatusOK, result)
    case err := <-errCh:
        return err
    case <-ctx.Done():
        return echo.NewHTTPError(http.StatusServiceUnavailable, "Request timeout")
    }
})
```

**Enhancements**

* Fixes by @aldas in https://github.com/labstack/echo/pull/2852
* Generic functions by @aldas in https://github.com/labstack/echo/pull/2856
* CRSF with Sec-Fetch-Site checks by @aldas in https://github.com/labstack/echo/pull/2858',
        'https://github.com/labstack/echo/pull/2864'
    );

    -- Echo PR pair 48: #2859 creates the Change, #2858 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Binder'),
        'fix: RouteNotFound handler does not falls back to root one',
        'Fixes #2485

## Changes
- Store RouteNotFound handler for "/*" path on Echo instance for fallback use
- Modify group middleware to check for root RouteNotFound handler before using default
- Add tests verifying root 404 handler fallback behavior for groups with middleware'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'From: https://github.com/labstack/echo/issues/2855

Note to self: [Hackernews](https://news.ycombinator.com/item?id=46351666) had this blog post posted.

* https://blog.miguelgrinberg.com/post/csrf-protection-without-tokens-or-hidden-form-fields
* https://words.filippo.io/csrf/
* https://github.com/rails/rails/pull/56350

see https://github.com/golang/go/blob/master/src/net/http/csrf.go  which was introduced in GO 1.25',
        'https://github.com/labstack/echo/pull/2859'
    );

    -- Echo PR pair 49: #2856 creates the Change, #2852 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Documentation'),
        'Generic functions',
        'Add generic functions to get typed PathParam/QueryParam/FormParam values. Also *Or variants for default values.
Note: those who have forgotten - structs can not have generic methods. only generic functions are allowed.
Types that are supported:
  - bool
  - float32
  - float64
  - int
  - int8
  - int16
  - int32
  - int64
  - uint
  - uint8/byte
  - uint16
  - uint32
  - uint64
  - string
  - echo.BindUnmarshaler interface
  - encoding.TextUnmarshaler interface
  - json.Unmarshaler interface
  - time.Duration
  - time.Time use echo.TimeOpts or echo.TimeLayout to set time parsing configuration

Scalar values:
* `id, err := echo.PathParam[int](c, "id")`
* `id, err := echo.QueryParam[int](c, "id")`
* `id, err := echo.FormParam[int](c, "id")`
* `id, err := echo.PathParamOr[int](c, "id", -1)`
* `id, err := echo.QueryParamOr[int](c, "id", -1)`
* `id, err := echo.FormParamOr[int](c, "id", -1)`

For getting slices:
* `ids, err := echo.QueryParams[int](c, "id")`   returns `[]int`
* `ids, err := echo.FormParams[int](c, "id")`

Generic parse functions:
* `id, err := echo.ParseValue[int]("123")`
* `id, err := echo.ParseValueOr[int]("123",-1)`
* `ids, err := echo.ParseValues[int]([]string{"123", "124"})`
* `ids, err := echo.ParseValuesOr[int]([]string{"123", "124"}, []int{666})`

Example
```go
	e.GET(`/user/:id`, func(c echo.Context) error {
		id, err := echo.PathParam[int](c, "id")
		if err == nil {
			return err
		}
		return c.String(http.StatusOK, fmt.Sprintf("Hello %v\n", id))
	})
```

-----------------------

For `Context.Get()` generic versions

* `v, err := ContextGetOr[int64](c, "key", 999)`
* `v, err := ContextGet[int64](c, "key")`

Example with JWT middleware:
```go
	e.Use(echojwt.JWT([]byte("secret")))

	e.GET(`/user/:id`, func(c echo.Context) error {
		token, err := echo.ContextGet[*jwt.Token](c, "user")
		if err == nil {
			return echo.ErrUnauthorized.WithInternal(err)
		}
		return c.String(http.StatusOK, fmt.Sprintf("Hello %v\n", token.Claims))
	})
```'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        '* Mark timeout middleware as deprecated
* fixes/improvements for things that Claude and Codex analysis found',
        'https://github.com/labstack/echo/pull/2856'
    );

    -- Echo PR pair 50: #2851 creates the Change, #2850 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Maintenance'),
        'Changelog for 4.14.0',
        'changelog and version number bump'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Closed Echo pull request #2850 did not include a body.',
        'https://github.com/labstack/echo/pull/2851'
    );

    -- Echo PR pair 51: #2849 creates the Change, #2843 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Router'),
        'Logger middleware json string escaping and deprecation',
        '**Security**

* Logger middleware: escape string values when logger format looks like JSON. See  #2846



**Enhancements**

* Add `middleware.RequestLogger` function to replace `middleware.Logger`. `middleware.RequestLogger` uses default slog logger.
  Default slog logger output can be configured to JSON format like that:
  ```go
  slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
  e.Use(middleware.RequestLogger())
  ```
* Deprecate `middleware.Logger` function and point users to `middleware.RequestLogger` and `middleware.RequestLoggerWithConfig`'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'dependabot complains that [golang.org/x/crypto](http://golang.org/x/crypto) need upgrading

Altough we do not used SSH package we still are "marked" as affected:
* https://github.com/advisories/GHSA-f6x5-jh6r-wrfv
* https://github.com/advisories/GHSA-j5w8-q4qc-rx2x',
        'https://github.com/labstack/echo/pull/2849'
    );

    -- Echo PR pair 52: #2842 creates the Change, #2838 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Middleware'),
        'Add Echo MCP tool',
        'Hi!

I added a reference to the echo-mcp tool.

I created this lib to automatically convert any Echo API to an MCP Tool that can be called by any agent.

The setup is really simple:

```go
    e := echo.New()

    // Existing API routes
    e.GET("/ping", func(c echo.Context) error {
        return c.JSON(http.StatusOK, map[string]string{"message": "pong"})
    })


    // Add MCP support
    mcp := server.New(e)
    mcp.Mount("/mcp")

    e.Start(":8080")
```'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        '(#2837)
Ensure proxy connection is closed in proxy middleware `proxyRaw` function',
        'https://github.com/labstack/echo/pull/2842'
    );

    -- Echo PR pair 53: #2835 creates the Change, #2833 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Binder'),
        'Replace custom private IP range check with built-in net.IP.IsPrivate',
        'Replace `isPrivateIPRange` with built-in method `net.IP.IsPrivate`'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Correct the fixture path used in `group_test.go`.

### Change

- Remove the redundant slash in the file path used in `group_test.go`',
        'https://github.com/labstack/echo/pull/2835'
    );

    -- Echo PR pair 54: #2832 creates the Change, #2829 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Documentation'),
        'Fix duplicate plus operator in router backtracking logic',
        '# Summary

This PR fixes a typo where searchIndex += +len(search) was incorrectly written with
duplicate plus operators. The correct expression should be searchIndex += len(search).

# Description

## Problem

In router.go at line 695, there is a syntax error with duplicate plus operators:

searchIndex += +len(search)

While Go interprets this as searchIndex = searchIndex + (+len(search)) (treating the
second + as a unary plus operator), it''s clearly a typo and not the intended expression.

## Solution

Changed the expression to the correct form:

searchIndex += len(search)

## Impact

• Functionality: The behavior remains the same since Go''s unary plus operator doesn''t
change the value, but the code is now clearer and matches the intended logic.
• Readability: Improves code clarity by removing the confusing duplicate operator.

## Testing

• Existing tests should continue to pass as the functional behavior is unchanged
• The fix is in the router''s backtracking logic for "any" node matching

## Location

• File: router.go
• Line: 695
• Function: Router.Find()'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        '## Summary

Complete redesign of the README with a modern, professional, and visually appealing layout that positions Echo as the premium choice for Go web development.

## ✨ Key Improvements

### **Visual & Design**
- 🎨 Modern layout with emojis, icons, and professional styling
- 🏢 Centered hero section with logo and clear value proposition
- 📊 Feature grid layout for better organization
- 🏗️ ASCII architecture diagram
- 📈 Performance benchmarks table
- 🔥 GitHub badges and project statistics

### **Content Enhancement**
- 🎯 **"Why Echo?"** section highlighting key differentiators
- 🚀 **Quick Start** with practical, copy-paste examples
- 🌟 **Feature showcase** organized in logical categories
- 📦 **Ecosystem** section with official and community packages
- 🎓 **Learning Resources** for developers at all levels
- 🏢 **Trust signals** showing companies using Echo

### **Developer Experience**
- ⚡ **60-second quick start** with complete working example
- 🛠️ **Enhanced contribution guidelines** with clear steps
- 🎯 **Roadmap** showing future features and vision
- 🤝 **Community** focus with discussion links
- 🆚 **Framework comparison** table

### **Professional Positioning**
- 📊 **Performance benchmarks** demonstrating superiority
- 🔒 **Production-ready** messaging with security focus
- 🌍 **Enterprise adoption** highlighting major users
- 📈 **Project metrics** showing healthy ecosystem

## 🎯 Impact

This README transforms Echo''s first impression from a simple framework description to a **premium, enterprise-ready solution** that developers and organizations can trust for critical applications.

### Before vs After:
- ❌ **Before**: Basic feature list, minimal examples
- ✅ **After**: Compelling value proposition, comprehensive showcase, professional presentation

## 🧪 Testing

- [x] All markdown renders correctly
- [x] All links are valid and functional
- [x] Images and badges display properly
- [x] No breaking changes to existing functionality
- [x] Maintains all original information while enhancing presentation

This positions Echo competitively against other frameworks and provides a compelling case for adoption.

🤖 Generated with [Claude Code](https://claude.ai/code)',
        'https://github.com/labstack/echo/pull/2832'
    );

    -- Echo PR pair 55: #2828 creates the Change, #2827 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Maintenance'),
        'Fix typo in SetParamValues comment',
        '## Summary

Fixes a simple typo in the SetParamValues method documentation.

**Change:**
- Line 362: `brake` → `break` in Router#Find code comment

**Benefits:**
- Correct English spelling
- Better code documentation clarity

## Test plan

- [x] No functional changes
- [x] Tests pass

🤖 Generated with [Claude Code](https://claude.ai/code)'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        '## Summary

Fixes a simple typo in the ContextTimeout middleware documentation.

**Change:**
- Line 19: `aries` → `arises` in ErrorHandler comment

**Benefits:**
- Correct English grammar
- Better code documentation

## Test plan

- [x] No functional changes
- [x] Tests pass

🤖 Generated with [Claude Code](https://claude.ai/code)',
        'https://github.com/labstack/echo/pull/2828'
    );

    -- Echo PR pair 56: #2826 creates the Change, #2825 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Router'),
        'Improve secure middleware readability and add deprecation notice',
        '## Summary

Improves code readability and maintainability of the secure middleware with better user guidance.

**Changes:**
1. **Refactor HSTS header construction** - Replace nested `fmt.Sprintf` with slice building and `strings.Join` for clearer logic
2. **Add X-XSS-Protection deprecation notice** - Document that CSP is recommended over the deprecated header
3. **Clean up imports** - Remove unused `fmt` import

**Benefits:**
- Cleaner, more maintainable HSTS directive building
- Better user guidance about modern security practices
- Improved code readability

## Test plan

- [x] All existing tests pass
- [x] Linting passes
- [x] No behavioral changes to security headers

Fixes #2799

🤖 Generated with [Claude Code](https://claude.ai/code)'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        '## Summary

Modernizes the BasicAuth middleware with improved code readability and RFC compliance.

**Changes:**
1. **Use `strings.Cut` for credential parsing** - Replaces manual for loop with Go 1.18+ `strings.Cut`
2. **Fix RFC 7617 compliance** - Always quote realm parameter as required by RFC

**Benefits:**
- Cleaner, more readable code using modern Go idioms
- Proper RFC 7617 compliance for WWW-Authenticate header
- Reduced code complexity (fewer lines, simpler logic)

## Test plan

- [x] All existing tests pass
- [x] Linting passes
- [x] No behavioral changes to authentication logic

Fixes #2794

🤖 Generated with [Claude Code](https://claude.ai/code)',
        'https://github.com/labstack/echo/pull/2826'
    );

    -- Echo PR pair 57: #2824 creates the Change, #2823 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Middleware'),
        'Modernize remaining interface{} to any in context.go',
        '## Summary

Completes the modernization of `context.go` by replacing all remaining `interface{}` types with `any`.

**Changes:**
- Updated Context interface method signatures (23 occurrences)
- Updated implementation method signatures
- Methods affected: Get, Set, Bind, Validate, Render, JSON*, XML*, JSONP

**Benefits:**
- Improved code readability
- Follows Go 1.18+ best practices
- Consistent with modern Go codebases

## Test plan

- [x] All existing tests pass
- [x] Linting passes
- [x] No behavioral changes

🤖 Generated with [Claude Code](https://claude.ai/code)'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        '## Summary

Modernizes a for loop in `context.go` to use Go 1.22''s new range over int syntax for cleaner iteration.

**Changes:**
- Replace `for i := 0; i < limit; i++` with `for i := range limit` in `SetParamValues` method

**Benefits:**
- Cleaner, more idiomatic Go 1.22+ code
- Slight performance improvement
- Reduced cognitive load

## Test plan

- [x] All existing tests pass
- [x] Linting passes
- [x] No behavioral changes

🤖 Generated with [Claude Code](https://claude.ai/code)',
        'https://github.com/labstack/echo/pull/2824'
    );

    -- Echo PR pair 58: #2822 creates the Change, #2821 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Binder'),
        'Modernize context.go by replacing interface{} with any',
        '## Summary

Modernizes the Echo Context interface by replacing all instances of `interface{}` with the more readable `any` type alias introduced in Go 1.18.

## Changes

**23 Modernizations across Context interface methods:**
- `Get(key string) interface{}` → `Get(key string) any`
- `Set(key string, val interface{})` → `Set(key string, val any)`
- `Bind(i interface{})` → `Bind(i any)`
- `Validate(i interface{})` → `Validate(i any)`
- `Render(code int, name string, data interface{})` → `Render(code int, name string, data any)`
- `JSON(code int, i interface{})` → `JSON(code int, i any)`
- `JSONP(code int, callback string, i interface{})` → `JSONP(code int, callback string, i any)`
- `XML(code int, i interface{})` → `XML(code int, i any)`
- `Blob(code int, contentType string, b []byte)` → (internal any usage)
- `Stream(code int, contentType string, r io.Reader)` → (internal any usage)
- Plus all other Context interface methods with `interface{}` parameters

## Benefits

### 🚀 **Modernization**
- Aligns with Go 1.18+ best practices and conventions
- Makes the API more approachable for developers familiar with modern Go
- Improves code readability and reduces cognitive load

### 📖 **Developer Experience**
- `any` is more intuitive and self-documenting than `interface{}`
- Easier to read in IDE tooltips and documentation
- Follows patterns used by modern Go libraries

### 🔒 **Safety & Compatibility**
- **Zero breaking changes** - `any` is just an alias for `interface{}`
- **100% backward compatible** - all existing code continues to work
- **Identical runtime behavior** - no performance or functional differences

## Testing

- ✅ **Compilation**: Code builds successfully
- ✅ **Tests**: Context tests pass without issues
- ✅ **Compatibility**: No API changes, only type alias substitution
- ✅ **Linting**: Addresses modernization suggestions from static analysis

## Type of Change

- 🎨 **Code modernization** - Updates to current Go standards
- 📚 **API clarity** - Improves readability without functional changes
- 🔧 **Developer experience** - Makes interfaces more approachable

## Impact

This change modernizes Echo''s public API to follow current Go conventions while maintaining perfect backward compatibility. Developers using Echo will benefit from:

- More readable method signatures
- Consistency with modern Go codebases
- Better IDE experience with cleaner type information

---

*This is a pure modernization change with zero risk - `any` and `interface{}` are functionally identical.*'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        '## Summary

Addresses issue #2382 by correcting the misleading comment on `Context.Bind` that did not accurately describe the actual binding behavior.

## Problem

The comment on `Context.Bind` in `context.go` was incomplete and confusing:

**Previous comment:**
```go
// Bind binds path params, query params and the request body into provided type `i`. The default binder
// binds body based on Content-Type header.
```

**Issues with the old comment:**
1. ❌ Didn''t explain the binding **order** (path → query → body)
2. ❌ Didn''t mention that later steps can **override** earlier values
3. ❌ Didn''t specify that query params are only bound for **GET/DELETE/HEAD**
4. ❌ Didn''t reference **single-source binding methods** for more control

## Solution

**New accurate comment:**
```go
// Bind binds data from multiple sources to the provided type `i` in this order:
// 1) path parameters, 2) query parameters (for GET/DELETE/HEAD only), 3) request body.
// Each step can override values from the previous step. For single source binding use
// BindPathParams, BindQueryParams, BindHeaders, or BindBody directly.
// The default binder handles body based on Content-Type header.
```

**Improvements:**
- ✅ **Clear binding order**: Explicitly states the 1-2-3 sequence
- ✅ **Override behavior**: Warns that later steps can override earlier values
- ✅ **HTTP method specifics**: Notes query param binding only for GET/DELETE/HEAD
- ✅ **Alternative methods**: References single-source binding methods
- ✅ **Content-Type info**: Preserves useful body binding information

## Impact

This fix prevents confusion for developers who might expect different behavior from `Context.Bind()` based on the previous misleading documentation. Now the comment accurately reflects the actual implementation in `bind.go`.

## Type of Change

- 📚 **Documentation fix** - No code logic changes
- 🔧 **Comment improvement** - Better developer experience
- 🎯 **Issue resolution** - Directly addresses reported confusion

## Testing

- ✅ Code compiles without issues
- ✅ No functional changes - documentation only
- ✅ Comment aligns with actual `DefaultBinder.Bind` implementation

Fixes #2382

---

*This is a simple documentation improvement that enhances clarity for Echo developers using the binding functionality.*',
        'https://github.com/labstack/echo/pull/2822'
    );

    -- Echo PR pair 59: #2819 creates the Change, #2818 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Documentation'),
        'Document ContextTimeout middleware with comprehensive examples',
        '## Summary

Addresses issue #2745 by providing comprehensive documentation for the ContextTimeout middleware, which was completely undocumented despite being the recommended approach for handling request timeouts in Echo.

## Problem Solved

Users were confused because:
- ContextTimeout middleware exists but has **zero documentation**
- Website only documents the **deprecated Timeout middleware** (with warnings against it)
- No examples showing how to properly implement handlers that work with ContextTimeout
- No explanation of why ContextTimeout is preferred over Timeout

## Changes

### 📚 **Comprehensive Configuration Documentation**

**Overview & Key Differences:**
- Clear explanation of ContextTimeout vs deprecated Timeout middleware
- Safety benefits: no response writer interference, no data races
- Cooperative cancellation model explanation

**Configuration Examples (3 scenarios):**
- Basic usage: `middleware.ContextTimeout(30 * time.Second)`
- Custom error handling with JSON responses
- Route-specific timeout skipping for health checks

### 🛠️ **Practical Handler Examples**

**3 Detailed Real-World Scenarios:**
1. **Database Operations**: Context-aware SQL queries with proper error handling
2. **Long-Running Processing**: Goroutine-based operations with select statements
3. **HTTP Proxy/Client**: Outbound requests with context propagation

### 📖 **Best Practices & Patterns**

**Common Integration Patterns:**
- Database: `db.QueryContext(ctx, query, args...)`
- HTTP Client: `http.NewRequestWithContext(ctx, method, url, body)`
- Redis: `redisClient.Get(ctx, key)`
- CPU-intensive loops with `ctx.Done()` checking

**Practical Guidelines:**
- Recommended timeout values for different use cases
- How to handle context cancellation gracefully
- When and where to place the middleware

### 🔧 **Enhanced Field Documentation**

- **Skipper**: Examples for excluding health check endpoints
- **ErrorHandler**: Custom timeout response patterns with JSON
- **Timeout**: Recommended durations for APIs, uploads, background tasks

### 🚀 **Function Documentation**

- **ContextTimeout()**: Basic usage with handler requirements
- **ContextTimeoutWithConfig()**: Advanced configuration examples
- **ToMiddleware()**: Validation and error handling scenarios

## Impact

This documentation addresses the exact concerns raised in issue #2745:

1. ✅ **"Cannot find any mention of ContextTimeout middleware"** → Now has 280+ lines of comprehensive docs
2. ✅ **"Documentation only lists the recommended-against Timeout middleware"** → Clear explanation of why ContextTimeout is preferred
3. ✅ **"Usage example showing what the handler should look like"** → 3 detailed handler examples + common patterns

## Before/After

**Before:** Zero documentation, users confused about which timeout middleware to use
**After:** Enterprise-grade documentation with practical examples and best practices

## Testing

- ✅ All existing ContextTimeout tests pass
- ✅ Code compiles without issues
- ✅ Documentation follows Go conventions
- ✅ Examples are syntactically correct and functional

Fixes #2745

---

This PR complements PR #2818 (Logger middleware documentation) as part of ongoing efforts to improve Echo''s middleware documentation quality.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        '## Summary

Addresses issue #2665 by providing comprehensive documentation for the Logger middleware that was previously lacking detailed explanations and examples.

## Changes

**📚 Configuration Examples (8 different scenarios):**
- Basic usage with default settings
- Custom simple format
- JSON format with custom fields
- Custom time formatting
- Logging headers, query params, form data, and cookies
- File output configuration
- Custom tag functions with user logic
- Conditional logging with Skipper
- External logging service integration

**🏷️ Complete Tag Reference organized by category:**
- **Time Tags**: 7 different timestamp formats
- **Request Information**: 10 request-related tags
- **Response Information**: 6 response-related tags
- **Dynamic Tags**: 4 parameterized tag types with examples

**📖 Enhanced Field Documentation:**
- Clear purpose explanation for each LoggerConfig field
- Usage examples and best practices
- Default values and behavior
- Proper Go reference time format examples

**🔧 Troubleshooting Section:**
- 4 common issues with solutions
- Performance optimization tips
- Best practices for high-traffic applications

**🚀 Function Documentation:**
- Detailed explanation of default Logger() behavior with example JSON output
- Comprehensive LoggerWithConfig() documentation with usage examples

## Impact

This enhancement transforms the Logger middleware from having minimal documentation to having enterprise-grade documentation that:

- **Helps new users** quickly understand and configure the middleware
- **Provides advanced patterns** for experienced developers
- **Reduces support burden** by answering common questions upfront
- **Improves developer experience** with clear examples and troubleshooting

## Testing

- ✅ All existing tests pass
- ✅ Code compiles without issues
- ✅ Documentation follows Go documentation conventions
- ✅ Examples are syntactically correct and functional

Fixes #2665

## Before/After

**Before:** Basic tag list with minimal explanations
**After:** 200+ lines of comprehensive documentation with 8+ complete configuration examples

The issue specifically requested "detailed explanations of configuration options and comprehensive examples for various use cases" - this PR delivers exactly that.',
        'https://github.com/labstack/echo/pull/2819'
    );

    -- Echo PR pair 60: #2815 creates the Change, #2812 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Maintenance'),
        'router: return error when registering nil handler',
        'Currently, `router.insert` only logs a warning when a nil handler is registered,
and continues processing. This can lead to runtime panics due to unnoticed misconfiguration.

This change makes router registration fail fast by returning an error
when a nil handler is provided, so that misconfigurations can be detected earlier.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Closed Echo pull request #2812 did not include a body.',
        'https://github.com/labstack/echo/pull/2815'
    );

    -- Echo PR pair 61: #2810 creates the Change, #2807 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Router'),
        'Use Go 1.25 in CI',
        'https://tip.golang.org/doc/go1.25'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'update `golang.org/x/` libs to current versions',
        'https://github.com/labstack/echo/pull/2810'
    );

    -- Echo PR pair 62: #2798 creates the Change, #2797 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Middleware'),
        'add optional IP anonymization',
        'This PR provides an option to anonymize IP:
* in context via `Context.AnonymizedIP()`
* in (old) logger middleware via `remote_ip_anon` token
* in (new) request logger middleware via `AnonymizeRemoteIP` option'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        '## Proposal
### 1. Error handling when reading request body:
Currently, errors in io.ReadAll(c.Request().Body) are ignored. If the read fails, subsequent handlers may receive an incomplete body, causing unexpected behavior. I believe that the robustness of the middleware can be improved by interrupting processing when an error occurs and returning an error.

```diff
import (
	"bufio"
	"bytes"
+++	"fmt"
	"errors"
	"io"
	"net"
// omission of sentence
			// Request
			reqBody := []byte{}
			if c.Request().Body != nil { // Read
---				reqBody, _ = io.ReadAll(c.Request().Body)
+++				var errRead error
+++				reqBody, errRead = io.ReadAll(c.Request().Body)
+++				if errRead != nil {
+++					return errRead
+++				}
			}
			c.Request().Body = io.NopCloser(bytes.NewBuffer(reqBody)) // Reset
```

### 2. Improvement of Flush panic messages:
Improve the messages when a panic occurs in the Flush method to be more detailed and consistent with the rest of the Echo framework (e.g., response.go). We believe this will make it clearer which ResponseWriters do not support the Flusher interface and make debugging easier.

```diff
import (
	"bufio"
	"bytes"
+++	"fmt"
	"errors"
	"io"
	"net"
// omission of sentence
func (w *bodyDumpResponseWriter) Flush() {
	err := http.NewResponseController(w.ResponseWriter).Flush()
	if err != nil && errors.Is(err, http.ErrNotSupported) {
		panic(errors.New("response writer flushing is not supported"))
		panic(fmt.Errorf("echo: response writer %T does not support flushing (http.Flusher interface)", w.ResponseWriter))
	}
}
```

### 3. Modification of tests to accommodate the above changes',
        'https://github.com/labstack/echo/pull/2798'
    );

    -- Echo PR pair 63: #2795 creates the Change, #2793 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Binder'),
        'Improved code readability and RFC compliance',
        '## Proposal
### 1. Improved parsing of authentication information:
I currently use a for loop to split username and password, but we believe that using strings.Cut, available in Go 1.18 or later, would make the code simpler and more readable.

```diff
// basic_auth.go


				cred := string(b)
---				for i := 0; i < len(cred); i++ {
---					if cred[i] == '':'' {
---						// Verify credentials
---						valid, err := config.Validator(cred[:i], cred[i+1:], c)
---						if err != nil {
---							return err
---						} else if valid {
---							return next(c)
---						}
---						break
---					}
+++				user, pass, ok := strings.Cut(cred, ":")
+++				if ok {
+++					// Verify credentials
+++					valid, err := config.Validator(user, pass, c)
+++					if err != nil {
+++						return err
+++					} else if valid {
+++						return next(c)
+++					}
				}
```

### 2. Added Realm quoting in WWW-Authenticate header:
RFC 7617 requires that the value of the realm parameter be a quoted-string. In the current implementation, the default realm is unquoted. You can comply with this specification by always using strconv.Quote

```diff
// basic_auth.go
---	realm := defaultRealm
---			if config.Realm != defaultRealm {
---				realm = strconv.Quote(config.Realm)
---			}

			// Need to return `401` for browsers to pop-up login box.
---			c.Response().Header().Set(echo.HeaderWWWAuthenticate, basic+" realm="+realm)
+++			// Realm is case-insensitive, so we can use "basic" directly. See RFC 7617.
+++			c.Response().Header().Set(echo.HeaderWWWAuthenticate, basic+" realm="+strconv.Quote(config.Realm))
			return echo.ErrUnauthorized

```

These changes will further improve the robustness and maintainability of the middleware.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Sorry, I accidentally deleted the local directory I forked from, so I created a new Pull Request!

## Changes
Changed the PANIC message to be more specific.
`Tests related to it''s changes`

## Improved debugging efficiency.
When Flush is not supported, it is clear which ResponseWriter is the culprit, speeding problem identification and resolution.

## Improved Developer Experience:
More specific error information makes it easier for developers to understand the cause of problems.
More specific error information helps developers understand the cause of problems.',
        'https://github.com/labstack/echo/pull/2795'
    );

    -- Echo PR pair 64: #2792 creates the Change, #2790 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Documentation'),
        'fixed issue #2791 Improved robustness of Response process and added debugging information',
        '## Changes
Improvements have been made to the WriteHeader and Flush methods of response.go.

## 1. **Check logger existence with ``WriteHeader``**:
`r.echo` and `r.echo.Logger.Warn` are not `nil` before calling `r.echo.Logger.Warn`. This reduces the risk of panic due to unexpected nil pointer references

## 2. **Additional error logging with `Flush`**:.
 If `http.ResponseController.Flush()` returns an error other than `http.ErrNotSupported`, it will now log the error if in debug mode (`r.echo.Debug == true`). This additional logging is useful for debugging during development, since the `Flush` method of the `http.Flusher` interface is by convention not to return errors, but the `ResponseController` may.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        '## Changes
Changed the PANIC message to be more specific.

## Improved debugging efficiency.
When Flush is not supported, it is clear which ResponseWriter is the culprit, speeding problem identification and resolution.

## Improved Developer Experience:
More specific error information makes it easier for developers to understand the cause of problems.
More specific error information helps developers understand the cause of problems.',
        'https://github.com/labstack/echo/pull/2792'
    );

    -- Echo PR pair 65: #2787 creates the Change, #2783 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Maintenance'),
        'Add authorization header handling in proxy middleware',
        '## Problem

When a proxy target URL contains User credentials, proxy middleware does not send request with the authorization header. This can lead to UnAuthorized Error (401) when connecting to upstream server.

## Solution

- Implemented logic to pass the Authorization header to the target if the proxy URL includes user credentials.
- Added unit tests to verify behavior for both scenarios: with and without user credentials in the proxy URL.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Closed Echo pull request #2783 did not include a body.',
        'https://github.com/labstack/echo/pull/2787'
    );

    -- Echo PR pair 66: #2782 creates the Change, #2781 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Router'),
        'Revert "CORS: reject requests with 401 for non-preflight request with not matching origin header"',
        'Reverts labstack/echo#2732

See: https://github.com/labstack/echo/issues/2767'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Closed Echo pull request #2781 did not include a body.',
        'https://github.com/labstack/echo/pull/2782'
    );

    -- Echo PR pair 67: #2780 creates the Change, #2764 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Middleware'),
        'Upgrade dependencies',
        'related to  https://github.com/labstack/echo/discussions/2779

* https://pkg.go.dev/vuln/GO-2025-3487  (affects: `golang.org/x/crypto/ssh`)
* https://pkg.go.dev/vuln/GO-2025-3503 (affects: `golang.org/x/net/http/httpproxy` and `golang.org/x/net/proxy` )
* https://pkg.go.dev/vuln/GO-2025-3595 (affects: `golang.org/x/net/html` )'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Added test explicit verification for [reuse CSRF token logic](https://github.com/labstack/echo/blob/3598f295f95f316bbeb252b7b332fe34e120815c/middleware/csrf.go#L136):
- Strictly validates token is reused when cookie exists
- Confirms new token is generated when no cookie provided

This reinforces detailed test for CSRF token behavior.',
        'https://github.com/labstack/echo/pull/2780'
    );

    -- Echo PR pair 68: #2762 creates the Change, #2760 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Binder'),
        'Add support for TLS WebSocket proxy',
        'This PR fixes the issue below.
https://github.com/labstack/echo/issues/2200


## Detail

- middleware/proxy.go
  - I change proxyRaw method to be able to proxy tls connection
- middleware/proxy_test.go
  - I added two tests. In this test, I use [golang.org/x/net/websocket](https://pkg.go.dev/golang.org/x/net/websocket)

## About websocket package in testing
I used an external package to easily implement web socket testing.

We considered the following packages, but we thought it would be better not to add too many third-party package dependencies for echo, so we adopted the official golang.org/x/net/websocket.

https://pkg.go.dev/github.com/gorilla/websocket
https://pkg.go.dev/github.com/coder/websocket'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Closed Echo pull request #2760 did not include a body.',
        'https://github.com/labstack/echo/pull/2762'
    );

    -- Echo PR pair 69: #2755 creates the Change, #2753 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Documentation'),
        'Add Transfer-Encoding as header',
        'Closed Echo pull request #2755 did not include a body.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Closed Echo pull request #2753 did not include a body.',
        'https://github.com/labstack/echo/pull/2755'
    );

    -- Echo PR pair 70: #2752 creates the Change, #2750 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Maintenance'),
        'feat: Add Mountpoint interface and related tests for App routing flexibility',
        '# Add Mountpoint Interface for Flexible Routing

## Description
This PR introduces a new `Mountpoint` interface that abstracts common routing methods shared between `Echo` and `Group` types. This enhancement improves code flexibility by allowing applications to register routes against either an `Echo` instance or a `Group` without needing to know which type they''re working with.

## Changes
- Added new `mountpoint.go` file defining the `Mountpoint` interface with all common routing methods
- Added comprehensive tests in `mountpoint_test.go` demonstrating the interface usage with both `Echo` instances and `Group` instances
- Verified that both `Echo` and `Group` types properly implement the interface

## Benefits
- Enables more modular application design by allowing route registration against any mountpoint
- Simplifies code that needs to work with both `Echo` instances and route groups
- Facilitates better code organization in larger applications with complex routing needs
- Maintains backward compatibility with existing code

## Testing
The PR includes extensive tests that verify:
- Basic functionality with both `Echo` and `Group` instances
- Proper handling of route prefixes with groups
- Nested group functionality
- Multiple mountpoint registration
- Middleware application

All tests pass successfully, confirming the interface works as expected.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'relates to #23

- Replaces large switch statement to a map to lower CCN from 15 to 5 (67% reduction).',
        'https://github.com/labstack/echo/pull/2752'
    );

    -- Echo PR pair 71: #2749 creates the Change, #2748 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Router'),
        'test(insertNode): add the second unit test for insertNode function',
        'Related to #5'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Go 1.24 was released https://tip.golang.org/doc/go1.24',
        'https://github.com/labstack/echo/pull/2749'
    );

    -- Echo PR pair 72: #2744 creates the Change, #2735 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Middleware'),
        'feat: add Forwarded header parsing and real IP extraction with tests',
        'For issue #2694

- Added parser for the "Forwarded" header to extract the "for" field.
- Implemented real IP extraction from the "Forwarded" headers.
- Added unit tests to validate header parsing and real IP extraction functionality.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Closed Echo pull request #2735 did not include a body.',
        'https://github.com/labstack/echo/pull/2744'
    );

    -- Echo PR pair 73: #2733 creates the Change, #2732 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Binder'),
        'Fix/only set request id if not exists',
        'A small optimization of not trying to reset request id incase one already exists'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'reject requests with 401 for non-preflight request with not matching origin header

fixes #2730',
        'https://github.com/labstack/echo/pull/2733'
    );

    -- Echo PR pair 74: #2727 creates the Change, #2722 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Documentation'),
        'Fix error in README example code',
        'the example code is missing an `errors` package import'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Update golang.org/x/net dependency [GO-2024-3333](https://pkg.go.dev/vuln/GO-2024-3333)',
        'https://github.com/labstack/echo/pull/2727'
    );

    -- Echo PR pair 75: #2721 creates the Change, #2719 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Maintenance'),
        'Update dependencies (dependabot reports https://pkg.go.dev/vuln/GO-2024-3321',
        'Update dependencies. dependabot reports https://pkg.go.dev/vuln/GO-2024-3321 / https://github.com/advisories/GHSA-v778-237x-gjrc

I do not see us directly affected us but dependabot reports are going to make people making issues/ticket in this repo. so it is better to prevent that.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'https://github.com/labstack/echo/pull/2717 which fixes https://github.com/labstack/echo/pull/2710',
        'https://github.com/labstack/echo/pull/2721'
    );

    -- Echo PR pair 76: #2717 creates the Change, #2715 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Router'),
        'Add Conditions to Ensure Bind Succeeds with `Transfer-Encoding: chunked`',
        'In Go, when the length of the Body is unknown, `ContentLength` is set to -1. As demonstrated in #2716, this applies to cases where `Transfer-Encoding: chunked` is used.

```
	// ContentLength records the length of the associated content.
	// The value -1 indicates that the length is unknown.
	// Values >= 0 indicate that the given number of bytes may
	// be read from Body.
	//
	// For client requests, a value of 0 with a non-nil Body is
	// also treated as unknown.
	ContentLength [int64](https://pkg.go.dev/builtin#int64)
```

https://pkg.go.dev/net/http#Request

Previously, only `0` was excluded during Bind, but starting from #2710, `-1` was also excluded, making it impossible to Bind requests with `Transfer-Encoding: chunked`. I have fixed this issue.


Fixes #2716.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'echo.Static serves files but unsorted. To make them sorted, I''ve followed the same procedure as was done for [net/http](https://github.com/golang/go/issues/11879) (the fix they did is shown [here](https://github.com/golang/go/commit/25b00177af9f62f683ec68f1d697c2607d087ea6#diff-0661442fffb473f85dc4d4472172edbfb4b9b1837db3ab1a73e838bed3e6ab70R597)).',
        'https://github.com/labstack/echo/pull/2717'
    );

    -- Echo PR pair 77: #2713 creates the Change, #2712 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Middleware'),
        'Update README.md',
        'Update to install and with version.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Next release `v4.13.0` will be Wednesday 2024.12.04.  This will upset probably quite a few people as we have breaking change. At least it is not on Friday :)

I will wait till December, after the Black Friday is over.',
        'https://github.com/labstack/echo/pull/2713'
    );

    -- Echo PR pair 78: #2711 creates the Change, #2710 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Binder'),
        'Shorten Github issue template and add test example',
        'Issue template is such a hassle to fill. hardly anyone does that correctly. it is better to concentrate people focus to provide test to demonstrate the problem than filling paragraphs that they will not fill, most of the time.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'I am writing a unit test and using httptest.NewRequestWithContext to create an http request, it will return a new http request with content-length = -1 with body as http.NoBody

```go
httpReq := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
httpReq.Header.Set(echo.HeaderContentType, "application/json")
```

https://github.com/golang/go/blob/master/src/net/http/httptest/httptest.go#L46C1-L77C3
```go
func NewRequestWithContext(ctx context.Context, method, target string, body io.Reader) *http.Request {
	if method == "" {
		method = "GET"
	}
	req, err := http.ReadRequest(bufio.NewReader(strings.NewReader(method + " " + target + " HTTP/1.0\r\n\r\n")))
	if err != nil {
		panic("invalid NewRequest arguments; " + err.Error())
	}
	req = req.WithContext(ctx)

	// HTTP/1.0 was used above to avoid needing a Host field. Change it to 1.1 here.
	req.Proto = "HTTP/1.1"
	req.ProtoMinor = 1
	req.Close = false

	if body != nil {
		switch v := body.(type) {
		case *bytes.Buffer:
			req.ContentLength = int64(v.Len())
		case *bytes.Reader:
			req.ContentLength = int64(v.Len())
		case *strings.Reader:
			req.ContentLength = int64(v.Len())
		default:
			req.ContentLength = -1
		}
		if rc, ok := body.(io.ReadCloser); ok {
			req.Body = rc
		} else {
			req.Body = io.NopCloser(body)
		}
	}
```
this is inconsistent in go-source-code as http.NewRequestWithContext with body as http.NoBody will have content-length as 0. i don''t know if this should be fixed in go-source-code. https://github.com/golang/go/issues/18117
https://github.com/golang/go/blob/master/src/net/http/request.go#L946-L951
```go
func NewRequestWithContext(ctx context.Context, method, url string, body io.Reader) (*Request, error) {
...
      if body != nil {
      ...
		case *bytes.Reader:
			req.ContentLength = int64(v.Len())
			snapshot := *v
			req.GetBody = func() (io.ReadCloser, error) {
				r := snapshot
				return io.NopCloser(&r), nil
			}
		case *strings.Reader:
			req.ContentLength = int64(v.Len())
			snapshot := *v
			req.GetBody = func() (io.ReadCloser, error) {
				r := snapshot
				return io.NopCloser(&r), nil
			}
		default:
			// This is where we''d set it to -1 (at least
			// if body != NoBody) to mean unknown, but
			// that broke people during the Go 1.8 testing
			// period. People depend on it being 0 I
			// guess. Maybe retry later. See Issue 18117.
```',
        'https://github.com/labstack/echo/pull/2711'
    );

    -- Echo PR pair 79: #2709 creates the Change, #2705 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Documentation'),
        'CORS middleware should compile allowOrigin regexp at creation',
        'This change preserves previous behavior - invalid patterns are just ignored.

we can not add panics as this would cause runtime unrecovered panics (i.e. some people very angry as their applications crash at prod)

Reported by https://github.com/labstack/echo/issues/2708'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Reported by https://github.com/labstack/echo/issues/2703',
        'https://github.com/labstack/echo/pull/2709'
    );

    -- Echo PR pair 80: #2702 creates the Change, #2701 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Maintenance'),
        'Fix issue #2694',
        'Fix Fix issue #2694 and add "Forwarded"'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'For #2699

[Seems consensus is to remove this middleware](https://github.com/labstack/echo/issues/2699#issuecomment-2464675851) and rely on the middleware for  https://github.com/labstack/echo-jwt .

This PR removes the dependencies and the middleware.',
        'https://github.com/labstack/echo/pull/2702'
    );

    -- Echo PR pair 81: #2700 creates the Change, #2698 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Router'),
        'dep: update golang-jwt to v4.5.1',
        'Fixes #2699

We want to avoid a known vulnerability in golang-jwt library is flagged as a security concern when using echo as a framework in our applications.

Tests are passing locally with the new version.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        '## CHANGE
use method, `echo.AcqurireContext` which defined.

```diff
-func (e *Echo) AcquireContext() Context {
-	return e.pool.Get().(Context)
+func (e *Echo) AcquireContext() *context {
+     return e.pool.Get().(*context)
}

func (e *Echo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Acquire context
-       c := e.pool.Get().(*context)
+	c := e.AcquireContext()
        ....
        // Release context
-	e.pool.Put(c)
+	e.ReleaseContext(c)
}
```',
        'https://github.com/labstack/echo/pull/2700'
    );

    -- Echo PR pair 82: #2695 creates the Change, #2692 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Middleware'),
        'Refactor basic auth middleware to support multiple auth headers',
        'This is taken from `v5` branch + improved tests

reasoning for it:
multiple auth headers is something that can happen in environments like corporate test environments that are secured by application proxy servers where front facing proxy is configured to require own basic auth value + checks it and your application also requires basic auth headers from clients.  As Go standard library stores headers in map and keys are retrieved in random order the middleware may need to check multiple headers to match correct one.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        '## Summary
This PR formats `interface{}` -> `any`.
`any` is an alias for `interface{}`
```go
var any = interface{}
```
https://github.com/golang/go/blob/67f131485541f362c8e932cd254982a8ad2cfc09/src/builtin/builtin.go#L97

## What was changed
- `interface{}` -> `any`
  - it required `go >= 1.18`
- It has nothing to do with performance.

## Why the change was made
- IMO: `any` is easier than an empty interface literal.

## How it was tested
```sh
go test ./... -cover
```',
        'https://github.com/labstack/echo/pull/2695'
    );

    -- Echo PR pair 83: #2691 creates the Change, #2690 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Binder'),
        'set cookie to request',
        'When setting a cookie to response also set it the request so that it can be retrieved in the same request.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Add TemplateRenderer struct to ease creating renderers for `html/template` and `text/template` packages.

Different take on #2673 ideas',
        'https://github.com/labstack/echo/pull/2691'
    );

    -- Echo PR pair 84: #2688 creates the Change, #2684 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Documentation'),
        'Refactor TestBasicAuth to utilize table-driven test format',
        '### Summary
This PR refactors the `basic_auth_test.go` file to use a table-driven test approach. The new structure improves readability, simplifies the addition of new test cases, and makes it easier to maintain the tests as the codebase evolves.

### What was changed
- Refactored individual test cases within TestBasicAuth into a table-driven test.
- Moved repeated logic into the test table to reduce redundancy.
- No changes were made to production code.

### Why the change was made
Table-driven tests provide a more scalable way to manage and add test cases. This refactor ensures that future test cases can be added with minimal repetition.

### How it was tested
- From the ''middleware'' directory, ran `go test -v -run TestBasicAuth`. All tests passed successfully.
- Manually checked each test within TestBasicAuth to ensure that if test conditions were changed the test failed.
- No functional changes were made to the codebase, so there should be no impact on production.

### Follow on
Am willing to refactor more tests if the table-driven format is desirable'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        '**Summary**
This PR introduces support for handling multipart requests that contain multiple files in the `bind` function. It extends the current functionality to allow seamless parsing and binding of multiple files uploaded through multipart form data.

**Changes**
- Modified the `bind` logic to handle multiple file uploads in a single request.
- Added support for accessing multiple files via `FormFile`.
- Updated the relevant documentation and comments for clarity.
- Introduced test cases to validate multi-file uploads.

**Testing**
- Added unit tests to ensure multi-file binding is working as expected.

**Additional Information**
This update improves the flexibility of the `bind` function when dealing with file uploads, making it easier to handle bulk file operations in a single request.',
        'https://github.com/labstack/echo/pull/2688'
    );

    -- Echo PR pair 85: #2683 creates the Change, #2682 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Maintenance'),
        'Improve error control & define limits',
        'In this PR:
- Improved error control
- Define a startup limit for arrays'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'improve `MultipartForm` to make it easier to know how to use it',
        'https://github.com/labstack/echo/pull/2683'
    );

    -- Echo PR pair 86: #2675 creates the Change, #2673 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Router'),
        'Add Go 1.23 to CI',
        'https://tip.golang.org/doc/go1.23

So we support Go 1.20, 1.21, 1.22, 1.23 from now.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'I have added a pre-built templates function to render HTML easily.',
        'https://github.com/labstack/echo/pull/2675'
    );

    -- Echo PR pair 87: #2671 creates the Change, #2664 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Middleware'),
        'Improve Logger Middleware Documentation',
        'This pull request enhances the documentation for Echo''s Logger middleware. The improvements aim to provide clearer, more comprehensive information for users implementing logging in their Echo applications.

Key changes include:
- Expanded overview of the Logger middleware and its features
- More detailed configuration options with examples
- Advanced usage scenarios, including custom output and route-specific logging
- Best practices for using the Logger middleware effectively
- Performance considerations for high-traffic scenarios

These updates should help both new and experienced Echo users better understand and utilize the Logger middleware, leading to improved debugging and monitoring capabilities in their applications.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'github.com/golang-jwt/jwt v3 version is not getting updates anymore, updated to v5
all tests pass',
        'https://github.com/labstack/echo/pull/2671'
    );

    -- Echo PR pair 88: #2660 creates the Change, #2659 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Binder'),
        'Implement Context interceptor for ServeHTTP',
        'I need to adjust the Context when using ServeHTTP, when not feasible with normal middleware.

Consider the following scenario.

When using WebSockets, I want to convert a received message to a http.Request.
Together with a custom ResponseWriter the request is handled by echo''s ServeHTTP.
But I want to adjust the Context and add my Socket reference to it.

There is no way to access the Context, except via middleware, but when handling middleware I have no actual access the the websocket anymore.

The easiest way to do this without interfering with other functionality or adding new functions, is checking if ResponseWriter conforms to a specific interface (ServeHTTPContextInterceptor) , if so, execute the interceptor and continue handling with the adjusted context.

Now a custom ResponseWriter can implement this interface and return a custom Context before `handle` and cleanup aftwards.

```
func (responseWriter *responseWriter) InterceptContext(ctx Echo.Context, handle func(ctx Echo.Context)) {

	handle(ctx)
}
```

Implemented and running in production at scale.

Another implemented usecase:

I have an Amazon AWS API Gateway routing all traffic to one Lambda Function, written in golang.

On receiving the call, I create a http.Request and ResponseWriter and perform ServeHTTP on echo, to handle the call.

Now I can create a Custom Context that hold various extra details about the request from API gateway.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'I need to adjust the context when using ServeHTTP, this is not feasible with normal middleware.

Consider the following scenario.

When using WebSockets, I want to convert a message to  http.Request.
Together with a custom ResponseWriter the request is handled by echo''s ServeHTTP.
But I want to adjust the Context and add my Socket reference to it.

There is no way to access the Context, except via middleware, but when handling middleware I have no actual access the the websocket anymore.

The easiest way to do this without interfering with other functionality or adding new functions, is checking if ResponseWriter conforms to a specific interface (ServeHTTPContextInterceptor) , if so, execute the interceptor and continue handling with the adjusted context.

Now a custom ResponseWriter can implement this interface and return a custom Context before `handle` and cleanup aftwards.

```
func (responseWriter *responseWriter) InterceptContext(ctx Echo.Context, handle func(ctx Echo.Context)) {

	handle(ctx)
}
```

Implemented and running in production at scale.

Another implemented usecase:

I have an Amazon AWS API Gateway routing all traffic to one Lambada Function, written in golang.

On receiving the call, I create a http.Request and ResponseWriter and perform ServeHTTP on echo, to handle the call.

Now I can create a Custom Context that hold various extra details about the request from API gateway.',
        'https://github.com/labstack/echo/pull/2660'
    );

    -- Echo PR pair 89: #2657 creates the Change, #2656 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Documentation'),
        'fix: Unclear behaviour of * in routes',
        'This should hopefully bring more clarity into this issue https://github.com/labstack/echo/issues/2619

Examples would explain this better so here''s similar to one from discussion

```
func main() {
	handler := func(c echo.Context) error {
		fmt.Printf("%s\n", c.Path())
		return nil
	}

	e := echo.New()
	v2 := e.Group("/v2")

	v2.DELETE("/*/blobs/:digest", handler)
	v2.GET("/*/blobs/:digest", handler)
	v2.HEAD("/*/blobs/:digest", handler)

	v2.DELETE("/*/manifests/:ref", handler)
	v2.GET("/*/manifests/:ref", handler)
	v2.HEAD("/*/manifests/:ref", handler)
	v2.PUT("/*/manifests/:ref", handler)

	v2.GET("/*/tags/list", handler)     // one wildcard
	v2.GET("/*/*/tags/list", handler)   // two wildcards
	v2.GET("/*/*/tags/list2*", handler) // two wildcards and trailing one
	v2.GET("/*/*/tags/list2", handler)  // two wildcards with fixed ending that may conflict with previous route

	v2.GET("/*/blobs/uploads/:ref", handler)
	v2.PATCH("/*/blobs/uploads/:ref", handler)
	v2.POST("/*/blobs/uploads", handler)
	v2.PUT("/*/blobs/uploads/:ref", handler)

	v2.GET("", handler)
	err := e.Start(":8080")
	if err != nil {
		panic(err)
	}
}
```

Example curl calls that should match those routes:

```
curl -ik  http://localhost:8080/v2/wildcard1/tags/list
# Matches /*/tags/list

curl -ik  http://localhost:8080/v2/wildcard1/wildcard2/tags/list
# Matches /*/*/tags/list

curl -ik  http://localhost:8080/v2/wildcard1/wildcard2/tags/list2wildcard3
# Matches /*/*/tags/list2*

curl -ik  http://localhost:8080/v2/wildcard1/wildcard2/tags/list2
# Matches /*/*/tags/list2
```


This one doesn''t match since you would need to register it as `/*/*/*/tags/list` and this probably better implemented as separate `**` wildcard
```
curl -ik  http://localhost:8080/v2/wildcard1/wildcard2/nowildcard/tags/list
{"message":"Method Not Allowed"}
```'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'This PR modifies the bindData function to preserve the pre v4.12.0 behavior for **map[string]interface{}** while supporting the new functionality:

- Single values are stored as strings
- Multiple values are stored as []string

This approach maintains compatibility with existing code that expects single string values, while allowing new code to take advantage of multiple values when present.

The change addresses the issue reported in https://github.com/labstack/echo/issues/2652, where the binding behavior for map[string]interface{} changed in v4.12.0, potentially breaking existing implementations.

Testing:
- Updated existing tests to reflect the new behavior

Please review and let me know if any further changes are needed.',
        'https://github.com/labstack/echo/pull/2657'
    );

    -- Echo PR pair 90: #2655 creates the Change, #2654 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Maintenance'),
        'Allow content type middleware',
        'Based on #2551

Adds a new middleware `AllowContentType` which restricts routes to only accepts requests with certain `Content-Type` header values.

It''s similar to the middleware of the same name available with Chi

The middleware also sets the `Accept` header field to the specified content types for all requests'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'just added http:// or https:// in front of the address when printed into the console to make it clickable.
probably better if i print it myself but this way its the last thing that''s printed in the console and it''s just easier',
        'https://github.com/labstack/echo/pull/2655'
    );

    -- Echo PR pair 91: #2653 creates the Change, #2636 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Router'),
        'chore: fix typo',
        'Closed Echo pull request #2653 did not include a body.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        '## Pull requset to resolve #2632

Modified files with saved memory size:

| File name   | Saved size(bytes) |
| ----------- | ----------------- |
| binder.go   | 40                |
| echo.go     | 8                 |
| echo_fs.go  | 8                 |
| group.go    | 16                |
| ip.go       | 8                 |
| response.go | 16                |
| router.go   | 32                |
| context.go  | 16                |
| echo.go     | 8                 |


**Total saved size:** 144

There was potential to further optimize the `Echo` struct in the `echo.go` file, but I chose to respect the readability of the codebase.

@aldas a quick review from you would be awesome whenever you’re free. Thanks!',
        'https://github.com/labstack/echo/pull/2653'
    );

    -- Echo PR pair 92: #2633 creates the Change, #2631 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Middleware'),
        'Update README.md',
        'A new tool that uses Echo as part of its core.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        '## What

replaced `interface{}` to  `any`.

## Why

golang 1.18 support any as type alias for interafce{}.
If echo follow golang support policy, the smarter using any than using interface{}.

## Concerns

Impact for an application run the environment that golang version  is < golang 1.18.
The application run at environment is  under golang 1.18  will be failed to run, since type alias any is supported after golang 1.18.
If echo support the application run at environment is under golang 1.18, this PR should be contained to v5.',
        'https://github.com/labstack/echo/pull/2633'
    );

    -- Echo PR pair 93: #2627 creates the Change, #2626 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Binder'),
        'When struct tag is not set, use actual field name for binding',
        'recently been using echo and even porting apps to echo from net/http and others.

i have added some tiny improvements in struct binding:
- if struct tag is missing then try to use field name as lookup key in data map (similar behavior as bson)
  - this feature is controlled by flag, `(*Echo).Binder` continues to work as it is
  - user can opt in to use this feature by setting `(*Echo).Binder = echo.BinderWithFallback()`)
- if the struct tag value is specified as a dash `-` it is skipped (similar behavior as encoding/json)
  - this feature is right there without a flag but it should not be a problem as reading certain value from `-` field is extremely unlikely and us gophers already treat it as a skipper (from well known behavior of json and others)'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Changelog for v4.12.0

I''ll tag it as minor version as we have quite a lot of different things here this time',
        'https://github.com/labstack/echo/pull/2627'
    );

    -- Echo PR pair 94: #2625 creates the Change, #2624 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Documentation'),
        'Update golang.org/x/* deps',
        '`golang.org/x/net` needs to be updated

> Vulnerability #1: GO-2024-2687
    HTTP/2 CONTINUATION flood in net/http
  More info: https://pkg.go.dev/vuln/GO-2024-2687
  Module: golang.org/x/net
    Found in: golang.org/x/net@v0.[22](https://github.com/labstack/echo/actions/runs/8693604426/job/23840712643?pr=2624#step:6:23).0
    Fixed in: golang.org/x/net@v0.[23](https://github.com/labstack/echo/actions/runs/8693604426/job/23840712643?pr=2624#step:6:24).0
    Example traces found:'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Relates to #1172

Use `httputil.ReverseProxy` to proxy SSE requests as it has support for streaming responses. See:
https://github.com/golang/go/blob/b107d95b9a66bfe7150fd4f2915e9bb876a6999a/src/net/http/httputil/reverseproxy.go#L601

------------

can be tested with

1. create separate package and execute this code to start serving proxy application at port 8080 that proxies requests to localhost:8081

```go
package main

import (
	"errors"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"log"
	"net/http"
	"net/url"
)

func main() {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	tmpURL, err := url.Parse("http://localhost:8081")
	if err != nil {
		log.Fatal(err)
	}
	e.Use(middleware.Proxy(middleware.NewRoundRobinBalancer([]*middleware.ProxyTarget{{URL: tmpURL}})))

	if err := e.Start(":8080"); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

```

2. Create application for serving SSE

Go file for application
```go
package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"log"
	"net/http"
	"time"
)

func main() {
	e := echo.New()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.File("/", "./index.html")

	e.GET("/sse", func(c echo.Context) error {
		log.Printf("SSE client connected, ip: %v", c.RealIP())

		w := c.Response()
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-c.Request().Context().Done():
				log.Printf("SSE client disconnected, ip: %v", c.RealIP())
				return nil
			case <-ticker.C:
				event := Event{
					Data: []byte("ping: " + time.Now().Format(time.RFC3339Nano)),
				}
				if err := event.WriteTo(w); err != nil {
					return err
				}
				w.Flush()
			}
		}
	})

	if err := e.Start(":8081"); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

// Event structure is defined here: https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events#event_stream_format
type Event struct {
	ID      []byte
	Data    []byte
	Event   []byte
	Retry   []byte
	Comment []byte
}

// WriteTo writes Event to given ResponseWriter
func (ev *Event) WriteTo(w http.ResponseWriter) error {
	// Marshalling part is taken from: https://github.com/r3labs/sse/blob/c6d5381ee3ca63828b321c16baa008fd6c0b4564/http.go#L16
	if len(ev.Data) == 0 && len(ev.Comment) == 0 {
		return nil
	}

	if len(ev.Data) > 0 {
		if _, err := fmt.Fprintf(w, "id: %s\n", ev.ID); err != nil {
			return err
		}

		sd := bytes.Split(ev.Data, []byte("\n"))
		for i := range sd {
			if _, err := fmt.Fprintf(w, "data: %s\n", sd[i]); err != nil {
				return err
			}
		}

		if len(ev.Event) > 0 {
			if _, err := fmt.Fprintf(w, "event: %s\n", ev.Event); err != nil {
				return err
			}
		}

		if len(ev.Retry) > 0 {
			if _, err := fmt.Fprintf(w, "retry: %s\n", ev.Retry); err != nil {
				return err
			}
		}
	}

	if len(ev.Comment) > 0 {
		if _, err := fmt.Fprintf(w, ": %s\n", ev.Comment); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprint(w, "\n"); err != nil {
		return err
	}

	return nil
}
```

in the same folder as app create index.html

```html
<!DOCTYPE html>
<html>
<body>

<h1>Getting server updates</h1>
<div id="result"></div>

<script>
  // Example taken from: https://www.w3schools.com/html/html5_serversentevents.asp
  if (typeof (EventSource) !== "undefined") {
    const source = new EventSource("/sse");
    source.onmessage = function (event) {
      document.getElementById("result").innerHTML += event.data + "<br>";
    };
  } else {
    document.getElementById("result").innerHTML = "Sorry, your browser does not support server-sent events...";
  }
</script>

</body>
</html>
```

3. Open http://localhost:8080 in your browser. You should see Ping messages streamed, assuming proxy middleware handles SSE requests as raw proxy',
        'https://github.com/labstack/echo/pull/2625'
    );

    -- Echo PR pair 95: #2618 creates the Change, #2616 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Maintenance'),
        'fix(middleware): Fix Allow method of RateLimiterMemoryStore',
        '## What
Modify `(*RateLimiterMemoryStore).Allow` method in rate limiter middleware.

## Why
Currently, `Allow` method acts unexpected behavior that it denies the request nevertheless subtract of `lastSeen`  and `now` exceeds `expiresIn`.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'When route is registered with empty path it is normalized to `/`. Make sure that returned echo.Route structs reflect that behavior.  Internally router has changed `` (empty path) to `/` for a long time but Route that is returned did not reflect that. Is is problematic with `Reverse` function that uses empty string as "not found"

Related to #2615

Previous behavior can be seen with this test:
```go
func TestTest(t *testing.T) {
	e := echo.New()

	handler := func(c echo.Context) error {
		return c.String(http.StatusNotImplemented, "Nope")
	}
	r := e.GET("", handler) // path is registered as "" previously. After change `/` is registered
	r.Name = "test"

	existingEmpty := e.Reverse("test")
	assert.Equal(t, "", existingEmpty)

	notExistingEmpty := e.Reverse("not-existing")
	assert.Equal(t, "", notExistingEmpty)
}
```

whis this change `assert.Equal(t, "", existingEmpty)` shoulb be change to  `assert.Equal(t, "/", existingEmpty)`to pass the test',
        'https://github.com/labstack/echo/pull/2618'
    );

    -- Echo PR pair 96: #2611 creates the Change, #2609 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Router'),
        'Remove maxparam dependence from Context',
        'This addresses race when Context is changing echo.Echo.maxParam value. This change does not address issues with Router not being safe when routes are added when server has already started and is serving incoming requests. This case is still unsafe (multiple goroutines writing+reading route tree)

Relates to https://github.com/labstack/echo/issues/1705'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'I have fixed the TargetHeader option of the RequestIDConfig, which was disabled.',
        'https://github.com/labstack/echo/pull/2611'
    );

    -- Echo PR pair 97: #2608 creates the Change, #2607 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Middleware'),
        'Default binder can bind pointer to slice as struct field. For example  `*[]string`',
        'Default binder can bind pointer to slice as struct field. For example `*[]string`

Related issue https://github.com/labstack/echo/issues/2381'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Default binder can use `UnmarshalParams(params []string) error` interface to bind multiple input values at one go.

Relates to https://github.com/labstack/echo/pull/2602

This allows developers to build fancy unmarsallers like related PR had. that turns `/?a=1,2,3&a=4,5,6` into `IntArrayB([]int{1, 2, 3, 4, 5, 6})`

```go
type IntArrayB []int

func (i *IntArrayB) UnmarshalParams(params []string) error {
	var numbers = make([]int, 0, len(params))

	for _, param := range params {
		var values = strings.Split(param, ",")
		for _, v := range values {
			n, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return fmt.Errorf("''%s'' is not an integer", v)
			}
			numbers = append(numbers, int(n))
		}
	}

	*i = append(*i, numbers...)
	return nil
}

func TestBindUnmarshalParams(t *testing.T) {
	t.Run("ok, target is an alias to slice and is nil, append multiple inputs", func(t *testing.T) {
		e := New()
		req := httptest.NewRequest(http.MethodGet, "/?a=1,2,3&a=4,5,6", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		result := struct {
			V IntArrayB `query:"a"`
		}{}
		err := c.Bind(&result)

		assert.NoError(t, err)
		assert.Equal(t, IntArrayB([]int{1, 2, 3, 4, 5, 6}), result.V)
	})
}
```',
        'https://github.com/labstack/echo/pull/2608'
    );

    -- Echo PR pair 98: #2606 creates the Change, #2605 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Binder'),
        'Change type definition blocks to single declarations. This helps copy…',
        'Change type definition blocks to single declarations. This helps copy/pasting Echo code in examples (for issues and discussions and for Echo website).

I admit this is yak shaving but answering issues with good examples is probably biggest time sink when it comes to maintaining Echo. And I feel that is important to answer with examples etc.   Also - website middleware part has block with middleware configurations. It is dedious to copy/paste middleware conf type there and have markdown intentation broken (blocks like that https://github.com/labstack/echox/blob/master/website/docs/middleware/csrf.md#configuration).'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Upgrade deps',
        'https://github.com/labstack/echo/pull/2606'
    );

    -- Echo PR pair 99: #2604 creates the Change, #2603 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Documentation'),
        'Add SPDX licence comments to files.',
        'Add SPDX licence comments to files. See https://spdx.dev/learn/handling-license-info/  There have been cases when Echo code has been copied but copyright reference has been hard to find in these repos. With these comments it should easier to understand what parts derive from Echo. Assuming these are not removed - but in that case, this is out of our hands.

NB: year is 2015 as this is @vishr first commit year in this repo. copypright number does not need to be updated every year.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'Closed Echo pull request #2603 did not include a body.',
        'https://github.com/labstack/echo/pull/2604'
    );

    -- Echo PR pair 100: #2602 creates the Change, #2596 supplies pr.
    select pg_temp.fn_demo_change_insert(
        _project_id,
        array['feature']::text[],
        (select id from public.epic where project_id = _project_id and name = 'Echo Maintenance'),
        'Aliased slices should be consistent with builtin slices',
        'In working with an API at work we had the use case on query params to accept the following syntaxes:

- `?foo=1&foo=2&foo=3`
- `?foo=1,2,3`

Initially we were using only the former syntax, but then a new library was developed that sent the latter syntax.  When we switched from defining our field as `[]int` to a custom type `Ints` that implemented `UnmarshalParam` we found that when there were multiple values for the same key, the `UnmarshalParam` would only receive the first value and others would be lost.

This change brings
consistency between the builtin slices and the aliased slices for query parameters.'
    ) into _change_id;
    call pg_temp.sp_demo_change_pr_update(
        _change_id,
        'I just fixed some typos.',
        'https://github.com/labstack/echo/pull/2602'
    );

end;
$$;

do
$$
declare
    _project_id bigint;
    _change_id bigint;
    _pr record;
    _epic_names text[] := array[
        'Echo Router',
        'Echo Middleware',
        'Echo Binder',
        'Echo Documentation',
        'Echo Maintenance'
    ];
begin
    select id into _project_id from public.project where name = 'demo1';

    for _pr in
        select
            pr.ordinal,
            pr.create_number,
            pr.create_updated_at,
            pr.create_title,
            pr.create_spec,
            pr.pr_number,
            pr.pr
        from jsonb_to_recordset($echo_pr_pairs$[{"ordinal":1,"create_number":2595,"create_updated_at":"2024-03-09T08:50:52Z","create_title":"Allow ResponseWriters to unwrap writers when flushing/hijacking","create_spec":"Relates to https://github.com/labstack/echo/issues/2592\r\n\r\nnote: these tests are quite stupid","pr_number":2590,"pr":"update dependencies which  and outdated github.com/golang-jwt/jwt"},{"ordinal":2,"create_number":2588,"create_updated_at":"2024-02-07T05:54:25Z","create_title":"CI: Use Go 1.22","create_spec":"Go 1.22 is released. See https://go.dev/doc/go1.22\r\n\r\n\r\nnote to self:  seems that in Go 1.22 reflection package gives different names for package level function variables. Previously these resulted as `github.com/labstack/echo/v4.glob..func1` but in 1.22  you get `github.com/labstack/echo/v4.init.func1`. I have changed problematic test to pass in newer/older versions.","pr_number":2584,"pr":"CI flow has notices\r\n\r\n> Node.js 16 actions are deprecated. Please update the following actions to use Node.js 20: actions/checkout@v3, actions/setup-go@v4. For more information see: https://github.blog/changelog/2023-09-22-github-actions-transitioning-from-node-16-to-node-20/.\r\n\r\n\r\nWe are using:\r\n* https://github.com/actions/checkout\r\n* https://github.com/actions/setup-go\r\n"},{"ordinal":3,"create_number":2581,"create_updated_at":"2026-01-28T19:08:32Z","create_title":"Reorder paragraphs in README.md","create_spec":"Reorder things in README.md\r\n\r\nI feel that Official website needs stand out more as it sometimes feels that people do not know that it exists.\r\nAlso I moved information about supported version down to example of adding library as depency is. That paragraph is not very relevant today. `v4` is quite old.","pr_number":2579,"pr":"There wasn't a sponsors section so I had to design one, hope you think it makes sense."},{"ordinal":4,"create_number":2576,"create_updated_at":"2024-02-05T23:38:04Z","create_title":"Allow content type middleware","create_spec":"See #2551\r\nI used the example code posted by @pzolo85 and made some modifications, mainly the Accept header feature.\r\n\r\nThis middleware will check if the Content-Type of an incoming request matches a list of allowed values.\r\nIn addition, it will modify the Accept field of the response's header with the allowed content types.\r\n\r\nCould be useful in telling API client's how the request should be sent and preventing invalid data formats from being sent to the server.\r\n\r\n","pr_number":2574,"pr":"Given the following code:\r\n\r\n```\r\nvar params map[string]interface{} // <- this is nil\r\nc.Bind(&params)\r\n```\r\n\r\nCurrently, if the context does not define any path or query params, this might work when parsing a JSON body.\r\nIf there are path parameters involved, this will panic with:\r\n```\r\necho: http: panic serving 127.0.0.1:41198: assignment to entry in nil map\r\ngoroutine 6 [running]:\r\nnet/http.(*conn).serve.func1()\r\n\t/usr/lib/golang/src/net/http/server.go:1868 +0x13d\r\npanic({0x845340?, 0x8deaf0?})\r\n\t/usr/lib/golang/src/runtime/panic.go:920 +0x290\r\nreflect.mapassign_faststr0(0x842580, 0x100c0000f8b18?, {0x8dc8c0?, 0x0?}, 0x842580?)\r\n\t/usr/lib/golang/src/runtime/map.go:1376 +0x25\r\nreflect.mapassign_faststr(0x842580, 0x0, {0x8dc8c0, 0x1}, 0xc000017520)\r\n\t/usr/lib/golang/src/reflect/value.go:3837 +0x65\r\nreflect.Value.SetMapIndex({0x842580, 0xc00006a088, 0x195}, {0x8327e0, 0xc000017510, 0x98}, {0x83d3a0, 0xc000017520, 0x94})\r\n\t/usr/lib/golang/src/reflect/value.go:2402 +0x2e5\r\ngithub.com/labstack/echo/v4.(*DefaultBinder).bindData(0xad1780, {0x82ed80, 0xc00006a088}, 0xc0000f91b8, {0x89016d, 0x5})\r\n\t/home/georg/go/pkg/mod/github.com/labstack/echo/v4@v4.11.5-0.20231220133251-60fc2fb1b76f/bind.go:152 +0x150d\r\ngithub.com/labstack/echo/v4.(*DefaultBinder).BindPathParams(0xad1780, {0x8e42b8, 0xc0000aaa00}, {0x82ed80, 0xc00006a088})\r\n\t/home/georg/go/pkg/mod/github.com/labstack/echo/v4@v4.11.5-0.20231220133251-60fc2fb1b76f/bind.go:40 +0x33d\r\ngithub.com/labstack/echo/v4.(*DefaultBinder).Bind(0xad1780, {0x82ed80, 0xc00006a088}, {0x8e42b8, 0xc0000aaa00})\r\n\t/home/georg/go/pkg/mod/github.com/labstack/echo/v4@v4.11.5-0.20231220133251-60fc2fb1b76f/bind.go:111 +0x6d\r\ngithub.com/labstack/echo/v4.(*context).Bind(0xc0000aaa00, {0x82ed80, 0xc00006a088})\r\n\t/home/georg/go/pkg/mod/github.com/labstack/echo/v4@v4.11.5-0.20231220133251-60fc2fb1b76f/context.go:439 +0x5d\r\n```\r\n\r\nWith this patch applied, there are no panics anymore.\r\n"},{"ordinal":5,"create_number":2568,"create_updated_at":"2024-05-07T18:55:36Z","create_title":"Remove default charset from 'application/json' Content-Type header","create_spec":"Using application/json; charset=UTF-8 in response header is a common misuse. I think it is better to remove `; charset=UTF-8` from default json response Content-Type header to prevent the misconception.\r\n\r\nSee: https://github.com/labstack/echo/issues/2567","pr_number":2564,"pr":"Changelog for v4.11.4\r\n\r\n\r\n-------------\r\n\r\n## v4.11.4 - 2023-12-20\r\n\r\n**Security**\r\n\r\n* Upgrade golang.org/x/crypto to v0.17.0 to fix vulnerability [issue](https://pkg.go.dev/vuln/GO-2023-2402) [#2562](https://github.com/labstack/echo/pull/2562)\r\n\r\n**Enhancements**\r\n\r\n* Update deps and mark Go version to 1.18 as this is what golang.org/x/* use [#2563](https://github.com/labstack/echo/pull/2563)\r\n* Request logger: add example for Slog https://pkg.go.dev/log/slog [#2543](https://github.com/labstack/echo/pull/2543)"},{"ordinal":6,"create_number":2563,"create_updated_at":"2023-12-20T13:17:24Z","create_title":"Update deps and mark Go version to 1.18 as this is what golang.org/x/* use","create_spec":"Update deps and mark Go version to 1.18 as this is what golang.org/x/* use.","pr_number":2562,"pr":"Vulnerability found on 12/18/2023 regarding to `golang.org/x/crypto` for versions `v0.16.0`and below. \r\n\r\nhttps://nvd.nist.gov/vuln/detail/CVE-2023-48795 | https://pkg.go.dev/vuln/GO-2023-2402\r\n\r\nThis MR upgrades dependency to `v0.17.0` to avoid vulnerability issue."},{"ordinal":7,"create_number":2560,"create_updated_at":"2026-06-15T19:12:51Z","create_title":"[issue-2557] Add :from-:to range route formats","create_spec":"Implements this feature (https://github.com/labstack/echo/issues/2557)\r\n\r\nAdds :from-:to route.\r\n\r\nexample /flights/:from-:to\r\n\r\n![image](https://github.com/labstack/echo/assets/42649107/99a3325a-acd1-4092-b5e0-863f85f69124)\r\n![image](https://github.com/labstack/echo/assets/42649107/39aa6917-1884-4ae2-9131-62ac23682a5c)\r\n","pr_number":2554,"pr":"Relates to #2552 and #988\r\n\r\nDifference from previous implementations is that in case we are binding to unsupported Map we ended in with panic. Now we skip binding (params/query/header) and try other sources (ala body)\r\n```go\r\npackage main\r\n\r\nimport (\r\n\t\"github.com/labstack/echo/v4\"\r\n\t\"github.com/labstack/echo/v4/middleware\"\r\n\t\"net/http\"\r\n)\r\n\r\nfunc main() {\r\n\te := echo.New()\r\n\te.Use(middleware.Logger())\r\n\te.Use(middleware.Recover())\r\n\r\n\t// test: `curl -XPOST --header \"Content-Type: application/json\" -d '{\"module1\": \"2\", \"module2\": \"3\"}' http://127.0.0.1:8080/test/string/1`\r\n\t// output old: {\"id\":\"1\",\"module1\":\"2\",\"module2\":\"3\"}\r\n\t// output new: {\"id\":\"1\",\"module1\":\"2\",\"module2\":\"3\"}\r\n\te.POST(\"/test/string/:id\", func(c echo.Context) error {\r\n\t\tp := map[string]string{}\r\n\t\tif err := c.Bind(&p); err != nil {\r\n\t\t\treturn err\r\n\t\t}\r\n\t\treturn c.JSON(http.StatusOK, p)\r\n\t})\r\n\r\n\t// test: `curl -XPOST --header \"Content-Type: application/json\" -d '{\"module1\": 2, \"module2\": 3}' http://127.0.0.1:8080/test/int/1`\r\n\t// output old: {\"message\":\"Internal Server Error\"}\r\n\t// output new: {\"module1\":\"2\",\"module2\":\"3\"}\r\n\te.POST(\"/test/int/:id\", func(c echo.Context) error {\r\n\t\tp := map[string]int{}\r\n\t\tif err := c.Bind(&p); err != nil {\r\n\t\t\treturn err\r\n\t\t}\r\n\t\treturn c.JSON(http.StatusOK, p)\r\n\t})\r\n\r\n\te.Start(\"127.0.0.1:8080\")\r\n}\r\n\r\n```"},{"ordinal":8,"create_number":2550,"create_updated_at":"2024-03-10T17:09:45Z","create_title":"Fix Real IP logic","create_spec":"Hello.\r\nThis fix for realIP logic.\r\nWe should check for trusting not real IP, but RemoteIP, who sends the request.\r\nFor example, we have a client - 1.1.1.1 and LB - 8.8.8.8.\r\nLB are trusting, all requests sended by it have X-Real-Ip header with client IP and we should extract it from headers.\r\nWe should not extract RealIP from requests sended from another hosts (not our LB).\r\nCurrent implementation checking client IP for trusting, but it's incorrect.","pr_number":2548,"pr":"Closed Echo pull request #2548 did not include a body."},{"ordinal":9,"create_number":2543,"create_updated_at":"2023-11-07T13:09:48Z","create_title":"request logger: add example for Slog https://pkg.go.dev/log/slog","create_spec":"request logger middleware: add example for Slog https://pkg.go.dev/log/slog","pr_number":2542,"pr":"**Security**\r\n\r\n* 'c.Attachment' and 'c.Inline' should escape filename in 'Content-Disposition' header to avoid 'Reflect File Download' vulnerability. [#2541](https://github.com/labstack/echo/pull/2541)\r\n\r\n**Enhancements**\r\n\r\n* Tests: refactor context tests to be separate functions [#2540](https://github.com/labstack/echo/pull/2540)\r\n* Proxy middleware: reuse echo request context [#2537](https://github.com/labstack/echo/pull/2537)\r\n* Mark unmarshallable yaml struct tags as ignored [#2536](https://github.com/labstack/echo/pull/2536)"},{"ordinal":10,"create_number":2541,"create_updated_at":"2023-11-07T12:10:09Z","create_title":"Security: c.Attachment and c.Inline should escape filename in `Content-Disposition` header","create_spec":"This fixes #2531\r\n\r\nc.Attachment and c.Inline should escape filename in `Content-Disposition` header to avoid 'Reflect File Download' vulnerability.\r\n\r\nThis is same as Go std does escaping https://github.com/golang/go/blob/9d836d41d0d9df3acabf7f9607d3b09188a9bfc6/src/mime/multipart/writer.go#L132\r\n\r\n","pr_number":2540,"pr":"refactor context tests to be separate functions. "},{"ordinal":11,"create_number":2539,"create_updated_at":"2026-01-28T19:08:45Z","create_title":"middleware: basic auth middleware can extract and check multiple auth…","create_spec":"basic auth middleware can extract and check multiple auth headers. \r\n\r\nThis is taken from `v5`.  Because of #2461 to get better tests in `v4` for that middleware.  Multiple auth headers is rare case. You probably can see this in test environments where application uses JWT token (which is also auth header) but for TEST env  Nginx/Apache2 etc is configured to have basic auth for all requests.  In that case your web-browser actually will send 2 auth header and it is 50/50 change that your request will pass as when previous version gets the header from Headers map you have 50/50 chance not to get JWT header.","pr_number":2537,"pr":"I have used the proxy middleware in one of my projects and need the context values in modifyResponse, which I had set before in my custom balancer.\r\nUnfortunately, I had to realise that the context does not seem to be taken over and there is no option to get it.\r\nWith this change, the context from `http.Request` is reused.\r\n\r\nIf there is an alternative way to get the previous request context, please tell me.\r\n"},{"ordinal":12,"create_number":2536,"create_updated_at":"2023-10-24T18:12:18Z","create_title":"Mark unmarshallable yaml struct tags as ignored","create_spec":"Relates to #2535 ","pr_number":2530,"pr":"Closed Echo pull request #2530 did not include a body."},{"ordinal":13,"create_number":2529,"create_updated_at":"2026-01-28T19:08:37Z","create_title":"Changelog for v4.11.2","create_spec":"## v4.11.2 - 2023-10-11\r\n\r\n**Security**\r\n\r\n* Bump golang.org/x/net to prevent CVE-2023-39325 / CVE-2023-44487 HTTP/2 Rapid Reset Attack [#2527](https://github.com/labstack/echo/pull/2527)\r\n* fix(sec): randomString bias introduced by #2490 [#2492](https://github.com/labstack/echo/pull/2492)\r\n* CSRF/RequestID mw: switch math/random usage to crypto/random [#2490](https://github.com/labstack/echo/pull/2490)\r\n\r\n**Enhancements**\r\n\r\n* Delete unused context in body_limit.go [#2483](https://github.com/labstack/echo/pull/2483)\r\n* Use Go 1.21 in CI [#2505](https://github.com/labstack/echo/pull/2505)\r\n* Fix some typos [#2511](https://github.com/labstack/echo/pull/2511)\r\n* Allow CORS middleware to send Access-Control-Max-Age: 0 [#2518](https://github.com/labstack/echo/pull/2518)\r\n* Bump dependancies [#2522](https://github.com/labstack/echo/pull/2522)","pr_number":2527,"pr":"Bump golang.org/x/net from v0.12.0 to v0.17.0\r\n\r\nRelated:\r\n* https://github.com/golang/go/issues/63417\r\n* https://github.com/golang/net/releases/tag/v0.17.0\r\n* https://www.cve.org/CVERecord?id=CVE-2023-44487"},{"ordinal":14,"create_number":2522,"create_updated_at":"2023-10-11T05:06:14Z","create_title":"Bump dependancies","create_spec":"Bump:\r\n* ~~golang.org/x/net v0.12.0 -> v0.15.0 (diff https://github.com/golang/net/compare/v0.12.0...v0.15.0)~~ (obsolete since #2527)\r\n* ~~golang.org/x/crypto v0.11.0 -> v0.13.0 (diff https://github.com/golang/net/compare/v0.11.0...v0.13.0)~~ (obsolete since #2527)\r\n* github.com/stretchr/testify v1.8.1 -> v1.8.4 (diff https://github.com/stretchr/testify/compare/v1.8.1...v1.8.4)\r\n\r\nThe main motivation is `golang.org/x/net` which got cleaned up recently in [v0.15.0](https://github.com/golang/net/releases/tag/v0.15.0) via https://github.com/golang/net/commit/4a2d37ed365334ff00b166660d7c497fcfeaef1b removing the reference to ancient [`ubuntu:trusty`, which has currently 559 known vulnerabilities, 4 of which are critical](https://hub.docker.com/layers/library/ubuntu/trusty/images/sha256-881afbae521c910f764f7187dbfbca3cc10c26f8bafa458c76dda009a901c29d?context=explore), triggering various false positive alerts...","pr_number":2518,"pr":"Allow CORS middleware to send `Access-Control-Max-Age: 0` value (when config.MaxAge is negative number) to instruct browsers not to cache that response.   This is backwards compatible change.\r\n\r\nimplements #2471"},{"ordinal":15,"create_number":2515,"create_updated_at":"2023-09-05T04:07:21Z","create_title":"Append a funtion to remove elements registerd in echo.context","create_spec":"- append function to remove elements registered in echo.context.\r\n- append test to test appended function.","pr_number":2511,"pr":"@aldas  Typo correction of variable names and comments"},{"ordinal":16,"create_number":2508,"create_updated_at":"2024-12-04T20:45:10Z","create_title":"Remove unneeded/duplicate jwt middleware in favor of https://github.com/labstack/echo-jwt","create_spec":"I was looking at one of my projects and realized there was an old JWT library reference in there. Then I realized that echo still uses some old version of the jwt middleware baked into the echo repo, while simultaneously noting that https://github.com/labstack/echo-jwt is the officially supported jwt middleware library.\r\n\r\nProblem: I think this is a confusing strategy - why not just recommend the use of https://github.com/labstack/echo-jwt instead of having 2 separate jwt middlewares, one baked into the echo lib itself with an old version of the jwt lib?\r\n\r\nThis PR removes the old jwt middleware with an old version of golang-jwt and updates dependencies. \r\n\r\n**I would recommend a minor or major version bump and just ask people to use the echo-jwt library as it uses the latest JWT version and is otherwise exactly the same code.**","pr_number":2506,"pr":"Closed Echo pull request #2506 did not include a body."},{"ordinal":17,"create_number":2505,"create_updated_at":"2023-08-12T06:01:36Z","create_title":"Use Go 1.21 in CI","create_spec":"Go 1.21 was released couple of days ago\r\n\r\n* https://go.dev/blog/go1.21\r\n* https://go.dev/doc/go1.21","pr_number":2500,"pr":"Closed Echo pull request #2500 did not include a body."},{"ordinal":18,"create_number":2496,"create_updated_at":"2024-03-10T19:00:27Z","create_title":"add context.pvalues len check while reset","create_spec":"see context.go:644","pr_number":2494,"pr":"https://github.com/labstack/echo/pull/2492#issuecomment-1646452964\r\n\r\nI'm not very good at English, hope this is enough\r\n\r\n"},{"ordinal":19,"create_number":2492,"create_updated_at":"2023-07-22T09:04:56Z","create_title":"fix(sec): `randomString` bias","create_spec":"security issue added by #2490\r\n\r\n`len(\"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz\")==52`, and `256 = 52 * 4 + 48`, so the possibility of each characters generated by `randomString` is not equal.\r\n\r\nA-Za-v: 5/256\r\nwxyz: 4/256\r\n\r\nalso perfermance improve, in newer (>=1.19) go version, `rand.Reader` is not buffered any more, so it's suggested to wrap `rand.Reader` with bufio if the data read from reader is small.\r\n\r\nhttps://tip.golang.org/doc/go1.19#:~:text=Read%20no%20longer%20buffers%20random%20data%20obtained%20from%20the%20operating%20system%20between%20calls","pr_number":2490,"pr":"switch math/random usage to crypto/random\r\n\r\nrelated to #2489"},{"ordinal":20,"create_number":2483,"create_updated_at":"2023-07-21T08:07:26Z","create_title":"delete unused context in body_limit.go","create_spec":"according to this [issue](https://github.com/labstack/echo/issues/2469), I remove unused context in limitedReader struct","pr_number":2482,"pr":"patch release  v4.11.1 to fix  https://github.com/labstack/echo/issues/2480 (PR https://github.com/labstack/echo/pull/2481)"},{"ordinal":21,"create_number":2481,"create_updated_at":"2023-07-22T15:45:21Z","create_title":"fix gzip not sending response code for no content responses (404, 301/302 redirects etc)","create_spec":"This is fix for #2480\r\n\r\nResponse code is not written from buffered writer to the actual writer when  handler sent no-content response - ala 404 or redirect\r\n\r\n```go\r\nfunc main() {\r\n\te := echo.New()\r\n\r\n\te.Use(middleware.Gzip())\r\n\r\n\te.GET(\"/404\", func(ctx echo.Context) error {\r\n\t\treturn ctx.NoContent(http.StatusNotFound)\r\n\t})\r\n\r\n\te.GET(\"/redirect\", func(ctx echo.Context) error {\r\n\t\treturn ctx.Redirect(http.StatusTemporaryRedirect, \"/login\")\r\n\t})\r\n\r\n\tif err := e.Start(\":8080\"); err != nil && !errors.Is(err, http.ErrServerClosed) {\r\n\t\tlog.Fatal(err)\r\n\t}\r\n}\r\n```\r\n\r\nExample output after fix:\r\n```bash\r\nx@x:~/$ curl -v --compressed \"http://localhost:8080/404\"\r\n*   Trying 127.0.0.1:8080...\r\n* Connected to localhost (127.0.0.1) port 8080 (#0)\r\n> GET /404 HTTP/1.1\r\n> Host: localhost:8080\r\n> User-Agent: curl/7.88.1\r\n> Accept: */*\r\n> Accept-Encoding: deflate, gzip, br, zstd\r\n> \r\n< HTTP/1.1 404 Not Found\r\n< Vary: Accept-Encoding\r\n< Date: Sun, 16 Jul 2023 17:20:11 GMT\r\n< Content-Length: 0\r\n< \r\n* Connection #0 to host localhost left intact\r\n\r\nx@x:~/$ curl -v --compressed \"http://localhost:8080/redirect\"\r\n*   Trying 127.0.0.1:8080...\r\n* Connected to localhost (127.0.0.1) port 8080 (#0)\r\n> GET /redirect HTTP/1.1\r\n> Host: localhost:8080\r\n> User-Agent: curl/7.88.1\r\n> Accept: */*\r\n> Accept-Encoding: deflate, gzip, br, zstd\r\n> \r\n< HTTP/1.1 307 Temporary Redirect\r\n< Location: /login\r\n< Vary: Accept-Encoding\r\n< Date: Sun, 16 Jul 2023 17:20:16 GMT\r\n< Content-Length: 0\r\n< \r\n* Connection #0 to host localhost left intact\r\n```","pr_number":2479,"pr":"Changes between last release and today https://github.com/labstack/echo/compare/v4.10.2...master\r\n\r\nWe have not done release for some time and there are plenty of fixes/enchantments in master waiting to be released. There are couple of fields added to middleware conf structs so it is not a patch release but minor version bump instead.\r\n\r\n@lammel or @vishr just to sync this with you."},{"ordinal":22,"create_number":2478,"create_updated_at":"2026-06-15T19:02:04Z","create_title":"fix(Context.Bind): should unescape special char in path","create_spec":"Other famous frameworks, such as expressjs or rails, will unescape the path params. We should follow the industry convention too.\r\n\r\nFor example:\r\n\r\n```go\r\npackage main\r\n​\r\nimport (\r\n\t\"fmt\"\r\n​\r\n\t\"github.com/labstack/echo/v4\"\r\n)\r\n​\r\ntype Req struct {\r\n\tA string `param:\"a\"`\r\n\tB string `query:\"b\"`\r\n}\r\n​\r\nfunc main() {\r\n\te := echo.New()\r\n\te.GET(\"/:a\", func(c echo.Context) error {\r\n\t\treq := Req{}\r\n\t\tc.Bind(&req)\r\n\t\tfmt.Println(req.A, req.B)\r\n\t\treturn nil\r\n\t})\r\n\te.Logger.Fatal(e.Start(\":3000\"))\r\n}\r\n```\r\n\r\nIf we send request `curl http://localhost:3000/%26\\?b\\=%26`, it will print `%26 &` not `& &`.\r\n\r\nIf we try expressjs, it won't have the problem:\r\n\r\n```js\r\nconst express = require(\"express\");\r\nconst app = express();\r\n\r\napp.get(\"/:name\", (req, res) => {\r\n  res.send(req.params);\r\n});\r\n\r\napp.listen(\"3000\");\r\n```","pr_number":2477,"pr":"fixes #2476 . This is problematic in tests as this is only place where that global `now` variable could be mutated"},{"ordinal":23,"create_number":2475,"create_updated_at":"2023-07-10T09:24:53Z","create_title":"Upgrade packages","create_spec":"Closed Echo pull request #2475 did not include a body.","pr_number":2468,"pr":"using a http.NoBody yields a content length of -1\r\n\r\nas per the documentation of request.ContentLength\r\n```go\r\n\t// ContentLength records the length of the associated content.\r\n\t// The value -1 indicates that the length is unknown.\r\n\t// Values >= 0 indicate that the given number of bytes may\r\n\t// be read from Body.\r\n\t//\r\n\t// For client requests, a value of 0 with a non-nil Body is\r\n\t// also treated as unknown.\r\n```"},{"ordinal":24,"create_number":2465,"create_updated_at":"2024-03-24T08:53:10Z","create_title":"Fix `echo.bindData`  not binding multiple values when map is used as destination","create_spec":"When calling `echo.Bind()`, `echo.BindBody()`, `echo.BindHeaders()`, etc. with `*map[string]any` as destination, currently multiple values of a single entry are being swallowed and only the first value is binded.\r\n\r\nMultiple values could be a result from repeatable/array `multipart/form-data` fields or header with multiple values.\r\n\r\nThis issue seems to be available in both v4 and v5_alpha.\r\n\r\nI've submitted the PR against v5_alpha since it could be a breaking change for v4 in case someone is relying on the current single value bind behavior.","pr_number":2461,"pr":"I Think Better To Add Skipper Sample Instead Of Nil, And The Explanation of BasicAuthValidator i think can me make more clear"},{"ordinal":25,"create_number":2456,"create_updated_at":"2023-05-29T20:27:11Z","create_title":"fix(DefaultHTTPErrorHandler): return error message when message is an error","create_spec":"The current behavior when returning a go `error` from a echo handler like this:\r\n\r\n```go\r\nreturn echo.NewHTTPError(status.BadRequest, errors.New(\"error in httperror\"))\r\n```\r\n\r\nIs that echo returns an empty JSON object `{}`.\r\n\r\nSee the failing test result, without the fix:\r\n\r\n```\r\n    echo_test.go:1356:\r\n        \tError Trace:\t/~/echo/echo_test.go:1356\r\n        \tError:      \tNot equal:\r\n        \t            \texpected: \"{\\\"message\\\":\\\"error in httperror\\\"}\\n\"\r\n        \t            \tactual  : \"{}\\n\"\r\n\r\n        \t            \tDiff:\r\n        \t            \t--- Expected\r\n        \t            \t+++ Actual\r\n        \t            \t@@ -1,2 +1,2 @@\r\n        \t            \t-{\"message\":\"error in httperror\"}\r\n        \t            \t+{}\r\n\r\n        \tTest:       \tTestDefaultHTTPErrorHandler\r\n```","pr_number":2455,"pr":"This PR should fix https://github.com/labstack/echo/issues/2447.\r\n\r\nIt removes the static node check optimization as it is preventing `UnescapePathParamValues` option to apply."},{"ordinal":26,"create_number":2453,"create_updated_at":"2026-06-15T19:00:55Z","create_title":"Add middleware with support proxy headers","create_spec":"Add middleware for support proxy headers (based on code gorilla mux handler https://github.com/gorilla/handlers/blob/master/proxy_headers.go) ","pr_number":2452,"pr":"gofmt fixes to comments"},{"ordinal":27,"create_number":2448,"create_updated_at":"2023-07-11T21:40:43Z","create_title":"RESTful Web Services with openapi3","create_spec":"Closed Echo pull request #2448 did not include a body.","pr_number":2444,"pr":"Closed Echo pull request #2444 did not include a body."},{"ordinal":28,"create_number":2442,"create_updated_at":"2026-01-20T12:15:31Z","create_title":"Add a stacktrace-inducing template token","create_spec":"Having `err` objects respond to `%+v` is quite widespread within the golang ecosystem. Add a logger template unit supporting this behavior.","pr_number":2440,"pr":"## we can bind vars with default value\r\nsometimes we want to bind default values, this pr can help you.\r\ncurrently only supports `struct` to set the default value.\r\n\r\n## examples\r\n```go\r\ntype person struct {\r\n    Name string `query:\"name\" default:\"bob\"`\r\n    Age  int    `query:\"age\" default:\"18\"`\t\r\n}\r\n```\r\nwhen `name` and `age` are not passed, they will be given default values.\r\n"},{"ordinal":29,"create_number":2439,"create_updated_at":"2023-04-28T04:41:58Z","create_title":"Custom Banner Support","create_spec":"Currently there are two options regarding the banner:\r\n\r\n- Default Banner\r\n- HideBanner\r\n\r\nI have created a `echo.CustomBanner` that can be used to supply a new banner. All the `echo` variables such as `Listen.Addr()` and other `echo.*` variables are injected into this function via an instance of `echo` as a parameter to it.\r\n\r\nFurthermore, I have included **two** tests that assert:\r\n\r\n- CustomBanner works.\r\n- Default Banner works when no custom banner is provided.\r\n\r\nAdditionally, I have also included a modification in the `Makefile` as an optional `test-verbose` command that would allow future developers to be able to generate output from test packages. ","pr_number":2436,"pr":"Closed Echo pull request #2436 did not include a body."},{"ordinal":30,"create_number":2433,"create_updated_at":"2025-08-08T21:12:34Z","create_title":"Add handling for empty content type in request header","create_spec":"Updated the code to handle the scenario when the content type in the request header is an empty string. Added a custom error to handle this scenario. No changes were made to other parts of the code.\r\n","pr_number":2429,"pr":"Fix `recover` and `request_logger` middlewares omit checking `err` before calling centralized errors."},{"ordinal":31,"create_number":2428,"create_updated_at":"2023-04-03T08:09:38Z","create_title":"Ohos 1490 bugfixes","create_spec":"Closed Echo pull request #2428 did not include a body.","pr_number":2426,"pr":"- Deprecated ErrStatusRequestEntityTooLarge and Add ErrStatusRequestEntityTooLarge"},{"ordinal":32,"create_number":2425,"create_updated_at":"2023-04-06T10:37:42Z","create_title":"add supprt for go1.20 http.rwUnwrapper","create_spec":"Support ResponseController for Go 1.20","pr_number":2424,"pr":"Closed Echo pull request #2424 did not include a body."},{"ordinal":33,"create_number":2418,"create_updated_at":"2023-03-16T05:46:03Z","create_title":"documentation: changed description for `Bind()` method","create_spec":"changed description for the `Bind()` method of `Context interface`. Because `Bind()` binds not only the request body but also the path and query params","pr_number":2416,"pr":"A small follow-up for https://github.com/labstack/echo/pull/1988 - add support of `\\\\:` into Reverse method of the router"},{"ordinal":34,"create_number":2414,"create_updated_at":"2023-05-15T10:27:13Z","create_title":"Support retries of failed proxy requests","create_spec":"Implements #2372 \r\n\r\nSupport for retrying proxy requests that fail due to an unavailable backend instance. ","pr_number":2411,"pr":"Fix group.RouteNotFound not working when group has attached middlewares.\r\n\r\nProblems is/was that `g.Use` registers special catch all routes with `g.Any` and those routes have priority over route registered by `g.NotFoundHandler`. \r\nSolution is to register these special routes also with `NotFoundHandler` so if you register custom one - it will override special catch all.\r\n\r\nFixes #2401\r\nFor history sake: somewhat relates to #1981 , #2256 , #1728"},{"ordinal":35,"create_number":2410,"create_updated_at":"2026-04-04T09:55:02Z","create_title":"Added a optional config variable to disable centralized error handler in recovery middleware","create_spec":"Recovery Middleware calls the centralized Error Handler. This is a change to have a optional config variable to disable centralized error handler in recovery. If the centrailzed Error Handler is disabled, panic error caught, will be returned to upstream middleware.\r\n","pr_number":2409,"pr":"Round Robin Balancer (RRB) `Next()` implementation did not properly use synchronization mechanisms to ensure right values are visible between concurrently executed code entering the same critical section.\r\n\r\nPrevious use of an atomic add to update the index value without also using an atomic load to read it is incorrect use of atomic synchronization (stale values are read in go-routines).\r\n\r\nThe index value, obviously, depends on the size of the `targets` slice. \r\nIf between index calculation and getting a value by index from the slice a target was removed and the index pointed to the last element then a panic due to out of bounds will be the result. \r\n\r\nHence, the logic must be guarded with the semaphore."},{"ordinal":36,"create_number":2407,"create_updated_at":"2023-07-22T15:45:38Z","create_title":"Changelog for v4.10.2","create_spec":"Closed Echo pull request #2407 did not include a body.","pr_number":2406,"pr":"Go 1.20 \"silently\" changed how `filepath.Clean` works on Windows. This is not backported to 1.19.6 etc.  Now we need todo some extra hoops to get static mw things work as they used to and securely\r\n\r\n```go\r\n// As of 1.20 on Windows filepath.Clean has different behaviour on OS related filesystems so we need to use path.Clean\r\n// which is more suitable for path coming from web but this has some caveats on Windows. When we eventually end up in\r\n// os related filesystem Open methods we are getting different errors as earlier versions. As of 1.20 path checks are\r\n// more strict on path you provide and consider path with [UNC](https://en.wikipedia.org/wiki/Path_(computing)#UNC)\r\n// but missing host etc parts as invalid. Previously it would result you `fs.ErrNotExist`.\r\n//\r\n// So for 1.20@Windows we need to consider it as same not exist so we can continue next middleware/handler and not error\r\n// which would result status 500 instead of potential route hit or 404.\r\n```"},{"ordinal":37,"create_number":2405,"create_updated_at":"2023-07-22T15:45:45Z","create_title":"CORS wildcard origin and allow credentials","create_spec":"Add middleware.CORSConfig.UnsafeWildcardOriginWithAllowCredentials to make UNSAFE usages of wildcard origin + allow cretentials less likely.\r\n\r\nSee: #2400","pr_number":2404,"pr":"Changelog for v4.10.1"},{"ordinal":38,"create_number":2402,"create_updated_at":"2023-07-22T15:45:46Z","create_title":"Upgrade deps","create_spec":"Upgrade deps due to the latest golang.org/x/net vulnerability","pr_number":2394,"pr":"Added https:// prefix, without it github markdown rendering does strange things:\r\n<img width=\"470\" alt=\"image\" src=\"https://user-images.githubusercontent.com/6695292/217364428-e4daa080-69ad-4be3-9e4e-dcf1a2a9bf13.png\">\r\n"},{"ordinal":39,"create_number":2388,"create_updated_at":"2023-01-30T10:58:00Z","create_title":"Replaced multiple mutex unlocks in function by using defer","create_spec":"Closed Echo pull request #2388 did not include a body.","pr_number":2386,"pr":"Closed Echo pull request #2386 did not include a body."},{"ordinal":40,"create_number":2385,"create_updated_at":"2023-01-28T16:16:04Z","create_title":"Return an empty string for ctx.path if there is no registered path","create_spec":"Proposed fix for https://github.com/labstack/echo/issues/2384","pr_number":2380,"pr":"We need to introduce a new middleware (`middleware.ContextTimeout()`) that creates context with timeout and injects `ContextWithTimeout` to `c.Request().Context()`. If the handler returns an error that wraps `context.DeadlineExceeded`, it returns [Service Unavailable (503)](https://www.rfc-editor.org/rfc/rfc9110.html#name-503-service-unavailable)\r\n\r\nThis fixes  #2379, #2306.\r\n\r\nCo-authored-by: @erhanakp"},{"ordinal":41,"create_number":2377,"create_updated_at":"2023-07-22T15:45:48Z","create_title":"Add new JWT repository to the README","create_spec":"Add new JWT repository to the README","pr_number":2374,"pr":"Warn users with debug enabled that middleware errors are not handled\r\n\r\nUsers may not expect errors thrown in the middleware on the response path flow to be ignored if the response has been committed by a handler"},{"ordinal":42,"create_number":2373,"create_updated_at":"2023-01-09T22:52:16Z","create_title":"fix: setCookie avoid security breach","create_spec":"Threat SetCookie method to always use security cookies.\r\n\r\nSimilar to https://security.snyk.io/vuln/SNYK-GOLANG-GITHUBCOMOPENSHIFTORIGINPKGCMDSERVERORIGINAUTHGO-2944969\r\n\r\n[A cookie with the Secure attribute is only sent to the server with an encrypted request over the HTTPS protocol](https://developer.mozilla.org/en-US/docs/Web/HTTP/Cookies#restrict_access_to_cookies).\r\n[A cookie with the HttpOnly attribute is inaccessible to the JavaScript Document.cookie API](https://developer.mozilla.org/en-US/docs/Web/HTTP/Cookies#restrict_access_to_cookies).","pr_number":2371,"pr":"This version will deprecate JWT middleware and introduces new [repository](https://github.com/labstack/echo-jwt) for it. This addresses many-many tickets like that #2323 and https://github.com/labstack/echo/pull/2122#issuecomment-1065904491\r\n\r\nAdd JWT middleware dependency with go modules\r\n  ```bash\r\n  go get github.com/labstack/echo-jwt/v4\r\n  ```\r\nDocs and example can be found here https://github.com/labstack/echo-jwt\r\n\r\n-------------------------------------\r\n\r\n\r\n**Security**\r\n\r\n* We are deprecating JWT middleware in this repository. Please use https://github.com/labstack/echo-jwt instead. \r\n\r\n  JWT middleware is moved to separate repository to allow us to bump/upgrade version of JWT implementation (`github.com/golang-jwt/jwt`) we are using which we can not do in Echo core because this would break backwards compatibility guarantees we try to maintain.\r\n\r\n* This minor version bumps minimum Go version to 1.17 (from 1.16) due `golang.org/x/` packages we depend on. There are several vulnerabilities fixed in these libraries.\r\n\r\n  Echo still tries to support last 4 Go versions but there are occasions we can not guarantee this promise.\r\n\r\n\r\n**Enhancements**\r\n\r\n* Bump x/text to 0.3.8 [#2305](https://github.com/labstack/echo/pull/2305)\r\n* Bump dependencies and add notes about Go releases we support [#2336](https://github.com/labstack/echo/pull/2336)\r\n* Add helper interface for ProxyBalancer interface [#2316](https://github.com/labstack/echo/pull/2316)\r\n* Expose `middleware.CreateExtractors` function so we can use it from echo-contrib repository [#2338](https://github.com/labstack/echo/pull/2338)\r\n* Refactor func(Context) error to HandlerFunc [#2315](https://github.com/labstack/echo/pull/2315)\r\n* Improve function comments [#2329](https://github.com/labstack/echo/pull/2329)\r\n* Add new method HTTPError.WithInternal [#2340](https://github.com/labstack/echo/pull/2340)\r\n* Replace io/ioutil package usages [#2342](https://github.com/labstack/echo/pull/2342)\r\n* Add staticcheck to CI flow [#2343](https://github.com/labstack/echo/pull/2343)\r\n* Replace relative path determination from proprietary to std [#2345](https://github.com/labstack/echo/pull/2345)\r\n* Remove square brackets from ipv6 addresses in XFF (X-Forwarded-For header) [#2182](https://github.com/labstack/echo/pull/2182)\r\n* Add testcases for some BodyLimit middleware configuration options [#2350](https://github.com/labstack/echo/pull/2350)\r\n* Additional configuration options for RequestLogger and Logger middleware [#2341](https://github.com/labstack/echo/pull/2341)\r\n* Add route to request log [#2162](https://github.com/labstack/echo/pull/2162)\r\n* GitHub Workflows security hardening [#2358](https://github.com/labstack/echo/pull/2358)\r\n* Add govulncheck to CI and bump dependencies [#2362](https://github.com/labstack/echo/pull/2362)\r\n* Fix rate limiter docs [#2366](https://github.com/labstack/echo/pull/2366)\r\n* Refactor how `e.Routes()` work and introduce `e.OnAddRouteHandler` callback [#2337](https://github.com/labstack/echo/pull/2337)\r\n"},{"ordinal":43,"create_number":2369,"create_updated_at":"2023-01-02T10:22:23Z","create_title":"context.Render doesn't return an error","create_spec":"context.Render doesn't return an error if there's an issue calling the c.echo.RendererRender; added the error to the `return` statement.","pr_number":2366,"pr":"Closes #1853 \r\n\r\n## What I did\r\n* fix docs and comment in `rate_limiter.go`\r\n## Why\r\n* It is difficult to understand the behavior of `middleware.NewRateLimiterMemoryStore` when the argument is  a float number.\r\n\r\n## Need to do\r\n* also Improve [this doc](https://github.com/labstack/echox/blob/master/website/content/middleware/rate-limiter.md)\r\n"},{"ordinal":44,"create_number":2362,"create_updated_at":"2022-12-29T14:29:20Z","create_title":"Add govulncheck to CI and bump dependencies","create_spec":"Add https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck to CI flow and bump dependencies \r\n\r\nI had to refactor CI flow to 2 parts as `govulncheck` reports problem for older Go versions thus causing workflow to fail. Therefore run  static analysis only with latest Go version.  There is no point to run golint and staticcheck with older versions as they are almost always teemed to fail (because of vulns that have patched in later versions). \r\n\r\nI added `env.LATEST_GO_VERSION` so it is easier in future to change Go version for step where are using it\"single\" version of GO.","pr_number":2358,"pr":"This PR adds explicit [permissions section](https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions#permissions) to workflows. This is a security best practice because by default workflows run with [extended set of permissions](https://docs.github.com/en/actions/security-guides/automatic-token-authentication#permissions-for-the-github_token) (except from `on: pull_request` [from external forks](https://securitylab.github.com/research/github-actions-preventing-pwn-requests/)). By specifying any permission explicitly all others are set to none. By using the principle of least privilege the damage a compromised workflow can do (because of an [injection](https://securitylab.github.com/research/github-actions-untrusted-input/) or compromised third party tool or action) is restricted.\r\nIt is recommended to have [most strict permissions on the top level](https://github.com/ossf/scorecard/blob/main/docs/checks.md#token-permissions) and grant write permissions on [job level](https://docs.github.com/en/actions/using-jobs/assigning-permissions-to-jobs) case by case."},{"ordinal":45,"create_number":2355,"create_updated_at":"2022-12-07T11:11:40Z","create_title":"Compress with empty content","create_spec":"This is a follow-up for PR #2044.\r\n\r\nIn [RFC9110](https://www.rfc-editor.org/rfc/rfc9110#name-content-type) is stated:\r\n> A sender that generates a message containing content SHOULD generate a Content-Type header field in that message unless the intended media type of the enclosed representation is unknown to the sender. If a Content-Type header field is not present, the recipient MAY either assume a media type of \"application/octet-stream\" ([[RFC2046](https://www.rfc-editor.org/rfc/rfc9110#RFC2046)], [Section 4.5.1](https://www.rfc-editor.org/rfc/rfc2046#section-4.5.1)) or examine the data to determine its type.\r\n\r\nSo this PR adds a check for the response size to ensure no `Content-Type` header is set if no payload is set.","pr_number":2350,"pr":"Add testcases for some BodyLimit middleware configuration options. Relates to #2344"},{"ordinal":46,"create_number":2346,"create_updated_at":"2022-11-25T09:15:34Z","create_title":"compress & decompress middleware support brotli","create_spec":"Closed Echo pull request #2346 did not include a body.","pr_number":2345,"pr":"I don't have a windows development environment, so I can't confirm that this will work as before, but I don't think replacing it with std will cause any problems."},{"ordinal":47,"create_number":2344,"create_updated_at":"2022-11-25T10:24:54Z","create_title":"`defer reader.Close()` in body_limit_test","create_spec":"`limitedReader.Close` function will be tested by this change.\r\nAs a result, the code coverage of `body_limit` increases.","pr_number":2343,"pr":"Add staticcheck to CI flow"},{"ordinal":48,"create_number":2342,"create_updated_at":"2022-11-22T00:40:58Z","create_title":" Replace io/ioutil package","create_spec":"`\"io/ioutil\"` package has been deprecated since Go 1.16.\r\n\r\nThis does not indicate that the package will be broken in the future, but analysis tools such as staticcheck will recognize it as a target for modification, thus burying the analysis to be fixed.\r\n\r\n```console\r\n$ git --no-pager grep ioutil\r\n$\r\n```\r\n\r\n<details>\r\n\r\n<summary>staticcheck-before</summary>\r\n\r\n```console\r\necho.go:46:2: \"io/ioutil\" has been deprecated since Go 1.16: As of Go 1.16, the same functionality is now provided by package io or package os, and those implementations should be preferred in new code. See the specific function documentation for details.  (SA1019)\r\necho_test.go:10:2: \"io/ioutil\" has been deprecated since Go 1.16: As of Go 1.16, the same functionality is now provided by package io or package os, and those implementations should be preferred in new code. See the specific function documentation for details.  (SA1019)\r\ngroup_test.go:4:2: \"io/ioutil\" has been deprecated since Go 1.16: As of Go 1.16, the same functionality is now provided by package io or package os, and those implementations should be preferred in new code. See the specific function documentation for details.  (SA1019)\r\nmiddleware/body_dump.go:7:2: \"io/ioutil\" has been deprecated since Go 1.16: As of Go 1.16, the same functionality is now provided by package io or package os, and those implementations should be preferred in new code. See the specific function documentation for details.  (SA1019)\r\nmiddleware/body_dump_test.go:5:2: \"io/ioutil\" has been deprecated since Go 1.16: As of Go 1.16, the same functionality is now provided by package io or package os, and those implementations should be preferred in new code. See the specific function documentation for details.  (SA1019)\r\nmiddleware/body_limit_test.go:5:2: \"io/ioutil\" has been deprecated since Go 1.16: As of Go 1.16, the same functionality is now provided by package io or package os, and those implementations should be preferred in new code. See the specific function documentation for details.  (SA1019)\r\nmiddleware/compress.go:7:2: \"io/ioutil\" has been deprecated since Go 1.16: As of Go 1.16, the same functionality is now provided by package io or package os, and those implementations should be preferred in new code. See the specific function documentation for details.  (SA1019)\r\nmiddleware/compress_test.go:7:2: \"io/ioutil\" has been deprecated since Go 1.16: As of Go 1.16, the same functionality is now provided by package io or package os, and those implementations should be preferred in new code. See the specific function documentation for details.  (SA1019)\r\nmiddleware/decompress_test.go:7:2: \"io/ioutil\" has been deprecated since Go 1.16: As of Go 1.16, the same functionality is now provided by package io or package os, and those implementations should be preferred in new code. See the specific function documentation for details.  (SA1019)\r\nmiddleware/jwt.go:265:2: this value of token is never used (SA4006)\r\nmiddleware/proxy_test.go:7:2: \"io/ioutil\" has been deprecated since Go 1.16: As of Go 1.16, the same functionality is now provided by package io or package os, and those implementations should be preferred in new code. See the specific function documentation for details.  (SA1019)\r\nmiddleware/proxy_test.go:387:2: this value of rec is never used (SA4006)\r\nmiddleware/rewrite_test.go:4:2: \"io/ioutil\" has been deprecated since Go 1.16: As of Go 1.16, the same functionality is now provided by package io or package os, and those implementations should be preferred in new code. See the specific function documentation for details.  (SA1019)\r\nmiddleware/timeout_test.go:8:2: \"io/ioutil\" has been deprecated since Go 1.16: As of Go 1.16, the same functionality is now provided by package io or package os, and those implementations should be preferred in new code. See the specific function documentation for details.  (SA1019)\r\nmiddleware/timeout_test.go:132:34: should use make(chan struct{}) instead (S1019)\r\nmiddleware/timeout_test.go:248:34: should use make(chan struct{}) instead (S1019)\r\nmiddleware/timeout_test.go:278:34: should use make(chan struct{}) instead (S1019)\r\n```\r\n\r\n</details>\r\n\r\n\r\n<details>\r\n\r\n<summary>staticcheck-after</summary>\r\n\r\n```console\r\nmiddleware/jwt.go:265:2: this value of token is never used (SA4006)\r\nmiddleware/proxy_test.go:387:2: this value of rec is never used (SA4006)\r\nmiddleware/timeout_test.go:132:34: should use make(chan struct{}) instead (S1019)\r\nmiddleware/timeout_test.go:248:34: should use make(chan struct{}) instead (S1019)\r\nmiddleware/timeout_test.go:278:34: should use make(chan struct{}) instead (S1019)\r\n```\r\n\r\n</details>\r\n","pr_number":2341,"pr":"* Add `middleware.RequestLoggerConfig.HandleError` configuration option to handle error within middleware with global error handler thus setting response status code decided by error handler and not derived from error itself.\r\n* Add `middleware.LoggerConfig.CustomTagFunc` so Logger middleware can add custom text/fields etc to logged (JSON or whatever format) row.\r\n\r\n```go\r\n\te.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{\r\n\t\tFormat: `{\"method\":\"${method}\",${custom}}` + \"\\n\",\r\n\t\tCustomTagFunc: func(c echo.Context, buf *bytes.Buffer) (int, error) {\r\n\t\t\treturn buf.WriteString(`\"tag\":\"my-value\"`)\r\n\t\t},\r\n\t}))\r\n```"},{"ordinal":49,"create_number":2340,"create_updated_at":"2022-12-29T14:29:41Z","create_title":"Add new method HTTPError.WithInternal","create_spec":"Add new method HTTPError.WithInternal to be able to create clone of HTTPError with given error set as internal\r\n\r\nCurrently HTTPError has method `SetInternal` but this mutates that same error which is problematic when we use those public errors we have ala `echo.ErrUnauthorized` etc. For these cases creating clone of that error instance is better. ","pr_number":2338,"pr":"Expose middleware.CreateExtractors function so we can use it from echo-contrib repository. JWT middleware will move there."},{"ordinal":50,"create_number":2337,"create_updated_at":"2022-12-29T14:29:22Z","create_title":"Routes and onhandlers","create_spec":"* Fix situation when Echo instance is used to serve multiple hosts. In this case all registered routes are seen from `e.Routes()` map but problem arises when multiple hosts have routes with same method+path - in this case latest added will only be in `e.Routes()` output. \r\n\r\n  * `e.Routes()` will only report routes added to default router (hosts = \"\")\r\n  * Routes for specific hosts are accessed by `e.Routers()[\"domain2.router.com\"].Routes()`\r\n  * Router has now new method `Reverse(name string, params ...interface{}) string`. Echos own `Reverse()` will call default router `Reverse` now.\r\n\r\n* Added handler to echo instance to help keeping track what routes are registered in a centralized way. There is new  handler field:\r\n```go\r\ne := echo.New()\r\ne.OnAddRouteHandler = func(host string, route Route, handler HandlerFunc, middleware []MiddlewareFunc) {\r\n  // for example: add this route info to your own registry \r\n}\r\n```\r\n\r\n","pr_number":2336,"pr":"Bump dependencies and add notes about Go releases we support.  `golang.org/x/` libraries do not work with Go 1.16 so we are bumping oldest version to Go 1.17. List of `golang.org/x` vuln is here https://pkg.go.dev/search?q=golang.org%2Fx&m=vuln \r\n\r\nThis is mostly done because we used version of testify that uses yaml library that is vulnerable. There are companies which security policies that disallow using libraries (even test) that have vulnerable dependencies. This case Echo is not using any of yaml stuff, even in tests but we still depend on those libraries.\r\n\r\nrelated to  #2326\r\n"},{"ordinal":51,"create_number":2329,"create_updated_at":"2022-11-12T21:56:47Z","create_title":"Modify comment syntax error","create_spec":"Modify comment syntax error","pr_number":2328,"pr":"added header constants;\r\n\"Accept-Language\"\r\n\"User-Agent\"\r\n\"Sec-CH-UA-Platform\"\r\n\"Sec-CH-UA-Platform-Version\"\r\n\"Date\""},{"ordinal":52,"create_number":2326,"create_updated_at":"2022-11-12T21:35:12Z","create_title":"fix(sec): upgrade gopkg.in/yaml.v3 to 3.0.0","create_spec":"### What happened？\nThere are 1 security vulnerabilities found in gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b\n- [CVE-2022-28948](https://www.oscs1024.com/hd/CVE-2022-28948)\n\n\n### What did I do？\nUpgrade gopkg.in/yaml.v3 from v3.0.0-20210107192922-496545a6307b to 3.0.0 for vulnerability fix\n\n### What did you expect to happen？\nIdeally, no insecure libs should be used.\n\n### The specification of the pull request\n[PR Specification](https://www.oscs1024.com/docs/pr-specification/) from OSCS","pr_number":2325,"pr":"Does not have any backward incompatibilities.\r\nCloses #2323"},{"ordinal":53,"create_number":2324,"create_updated_at":"2026-01-20T12:14:14Z","create_title":"Add zap4echo to README","create_spec":"I wrote a [middleware package](https://github.com/tomruk/zap4echo) to log requests and to recover from panics. It uses zap, my favorite logging package.","pr_number":2316,"pr":"Idea is discussed in this [issue](https://github.com/labstack/echo/issues/2313)"},{"ordinal":54,"create_number":2315,"create_updated_at":"2022-11-12T21:54:52Z","create_title":"fix func(Context) error to HandlerFunc","create_spec":"HandlerFunc has already been defined as func(Context) error in echo.go.\r\nI think using HandlerFunc is more smarter than func(Context) error!","pr_number":2311,"pr":"There are (rare) cases when you would want to bind value from different sources. So this PR allows binders to be chained together to create multi-source binder. \r\n\r\nThis has few use-cases because we have `ValueBinder`s only for Path/Query/Form and most of the time if multi-source binding is needed we talk about  body (ala JSON) + some other source (ala query).\r\n\r\nExample:\r\n```go\r\n// bound query params should have priority over path params\r\nb := QueryParamsBinder(c).UseBefore(PathParamsBinder(c))\r\n```\r\nor \r\n\r\n```go\r\n// bound params priority:\r\n// 1. Path params\r\n// 2. Query params\r\n// 3. Form fields\r\nb := PathParamsBinder(c).UseBefore(QueryParamsBinder(c)).UseBefore(FormFieldBinder(c))\r\n```\r\n\r\n-----------\r\n\r\nI am not sure if this name `UseBefore` is best. I considered `Combine`, `CombineWith`, `CombineBefore`, `Use`, `Chain`\r\n\r\n`*Before` is added as suffix to make it clear there is order of binding. `*Before` indicates that left side binder value is used before right side binder value. But I am at the moment no even sure that people would read/understand it like I do.\r\n\r\nI am open to suggestions for naming it.\r\n\r\np.s. this is low priority, low value change"},{"ordinal":55,"create_number":2309,"create_updated_at":"2022-10-25T06:15:11Z","create_title":"handle some errors in tests","create_spec":"Closed Echo pull request #2309 did not include a body.","pr_number":2305,"pr":"see https://go.dev/issue/56152, https://ossindex.sonatype.org/vulnerability/CVE-2022-32149?component-type=golang&component-name=golang.org%2Fx%2Ftext&utm_source=nancy-client&utm_medium=integration&utm_content=1.0.41"},{"ordinal":56,"create_number":2302,"create_updated_at":"2022-12-29T14:30:48Z","create_title":"bump gommon version and do release for v4.9.1","create_spec":"bump gommon version (to fix https://github.com/labstack/echo/issues/2295) and do release for v4.9.1","pr_number":2301,"pr":"remove all `xxx := assert.New()` usages"},{"ordinal":57,"create_number":2299,"create_updated_at":"2022-11-21T17:24:25Z","create_title":"add SetQueryParams and fix panic Context.Reset()","create_spec":"Closed Echo pull request #2299 did not include a body.","pr_number":2297,"pr":"replace all `assert.New` with `assert.Equal`"},{"ordinal":58,"create_number":2292,"create_updated_at":"2022-11-20T19:33:05Z","create_title":"v5: update doc on jwt","create_spec":"For echo `v5_alpha`, there's no more default implementation for `ParseTokenFunc`.","pr_number":2291,"pr":"for #2290 - Update readme about supported Go versions. \r\n\r\nCurrently we support\r\nhttps://github.com/labstack/echo/blob/666938e523c62170646fc2320cc7d97bcacdfd6f/.github/workflows/echo.yml#L29-L31"},{"ordinal":59,"create_number":2282,"create_updated_at":"2022-09-24T04:51:48Z","create_title":"Add protobuf binding","create_spec":"Added protobuf binding, because I need it.","pr_number":2281,"pr":"I believe it will be very useful to make commonBalancer(https://github.com/labstack/echo/blob/v4.9.0/middleware/proxy.go#L75) available for extending to create one's own loadbalancing strategies.\r\n\r\nThis PR solves for it. "},{"ordinal":60,"create_number":2277,"create_updated_at":"2023-02-23T06:39:48Z","create_title":"Add more http error values","create_spec":"- Add more http error values.\r\n","pr_number":2275,"pr":"Closed Echo pull request #2275 did not include a body."},{"ordinal":61,"create_number":2272,"create_updated_at":"2022-09-12T19:46:29Z","create_title":"Improve CORS documentation","create_spec":"* Provide links to further reading\r\n* Provide security warnings\r\n* Document undocumented wildcard feature\r\n* Update to go-1.19 style links","pr_number":2268,"pr":"I added ``level`` field to logging template and allow customize with ``LevelSetter`` function for different use cases. For example, you can set  ``level`` to ``info`` by just  checking ``err == nil`` and/or you can set ``error`` when request responded with ``500``\r\n and/or ``error`` is not nil.\r\n\r\nAdding ``level`` field to logs allows better filtering and observability in your log stack."},{"ordinal":62,"create_number":2267,"create_updated_at":"2023-06-01T02:12:15Z","create_title":"gzip response only if it exceeds a minimal length","create_spec":"If the response is too short, e.g. a few bytes, compressing the response makes it even larger. The new parameter MinLength to the GzipConfig struct allows to set a threshold (in bytes) as of which response size the compression should be applied. If the response is shorter, no compression will be applied.","pr_number":2261,"pr":"Changelog for 4.9.0\r\n\r\ngoing to 4.9.0 because we had csrf middleware errorhandler #2257 already commited to main. otherwise would have used 4.8.1"},{"ordinal":63,"create_number":2260,"create_updated_at":"2022-12-29T14:29:57Z","create_title":"Fix #2259 open redirect vulnerability in echo.StaticDirectoryHandler (used by e.Static, e.StaticFs etc)","create_spec":"Fix #2259 open redirect vulnerability in echo.StaticDirectoryHandler (used by e.Static, e.StaticFs etc)\r\n\r\nSimilar fix as #1771 had. `path.Clean()` could be alternative but potentially causes too much changes to path.\r\n\r\nremove pre Go1.16 and after differences not to duplicate stuff anymore","pr_number":2258,"pr":"Headers are supported in Bind() and we can find the following in the\r\ndocumentation[1]:\r\n\r\n\tEcho provides the following methods to bind data from different\r\n\tsources to Go Structs using the Context#Bind(i interface{}) method:\r\n\t- URL Path parameter\r\n\t- URL Query parameter\r\n\t- Request body\r\n\t- Header\r\n\r\n1. https://echo.labstack.com/guide/binding/#bind-using-struct-tags\r\n"},{"ordinal":64,"create_number":2257,"create_updated_at":"2022-09-01T07:51:55Z","create_title":"Added ErrorHandler and ErrorHandlerWithContext in CSRF middleware","create_spec":"Fixes #2183","pr_number":2254,"pr":"Closed Echo pull request #2254 did not include a body."},{"ordinal":65,"create_number":2247,"create_updated_at":"2022-08-20T19:25:45Z","create_title":"Replace http methods constancies with stdlib constancies","create_spec":"Closed Echo pull request #2247 did not include a body.","pr_number":2242,"pr":"Changelog for 4.8.0 and bump version string"},{"ordinal":66,"create_number":2240,"create_updated_at":"2022-08-09T11:47:10Z","create_title":"add:README.md-Third-party middlewares-github.com/go-woo/protoc-gen-echo","create_spec":"github.com/go-woo/protoc-gen-echo is a protoc plug-in. can generate echo server side code from .proto file.\r\n","pr_number":2239,"pr":"github.com/go-woo/protoc-gen-echo is a protoc plug-in. can generate echo server side code from .proto file.\r\n[README.md](https://github.com/labstack/echo/files/9278660/README.md)\r\n "},{"ordinal":67,"create_number":2238,"create_updated_at":"2022-12-29T14:30:04Z","create_title":"Update CI-flow (Go 1.19 +deps)","create_spec":"Update CI-flow (Go 1.19 +deps)","pr_number":2237,"pr":"Allow arbitrary HTTP method types to be added as routes. \r\n\r\nThis does not affect routing \"usual\" use-case performance as these arbitrary HTTP method types and underlying map is accessed only when request is of that type. This code is taken from `v5`.\r\n\r\nRelates to;\r\n* https://github.com/labstack/echo/issues/1952\r\n* https://github.com/labstack/echo/pull/2173\r\n* https://github.com/labstack/echo/issues/1610\r\n* https://github.com/labstack/echo/issues/1459\r\n\r\n\r\nExample:\r\n```go\r\nimport (\r\n\t\"fmt\"\r\n\t\"github.com/labstack/echo/v4\"\r\n\t\"log\"\r\n\t\"net/http\"\r\n)\r\n\r\nfunc main() {\r\n\te := echo.New()\r\n\r\n\te.Add(\"COPY\", \"/*\", func(c echo.Context) error {\r\n\t\treturn c.String(http.StatusOK, \"OK COPY\")\r\n\t})\r\n\r\n\tif err := e.Start(\":8080\"); err != http.ErrServerClosed {\r\n\t\tlog.Print(fmt.Errorf(\"error when starting HTTP server: %w\", err))\r\n\t}\r\n}\r\n```\r\n\r\nOutput:\r\n```bash\r\nx@x:~/code/$ curl -v -X COPY \"http://localhost:8080/something\"\r\n*   Trying 127.0.0.1:8080...\r\n* Connected to localhost (127.0.0.1) port 8080 (#0)\r\n> COPY /something HTTP/1.1\r\n> Host: localhost:8080\r\n> User-Agent: curl/7.81.0\r\n> Accept: */*\r\n> \r\n* Mark bundle as not supporting multiuse\r\n< HTTP/1.1 200 OK\r\n< Content-Type: text/plain; charset=UTF-8\r\n< Date: Sat, 06 Aug 2022 20:34:24 GMT\r\n< Content-Length: 7\r\n< \r\n* Connection #0 to host localhost left intact\r\nOK COPY\r\n```"},{"ordinal":68,"create_number":2229,"create_updated_at":"2023-02-21T21:21:36Z","create_title":"remove unused param","create_spec":"remove unused param","pr_number":2228,"pr":"- Helps consumers who want to wrap HTTPError, and other use cases\r\n- Added testing for HEAD requests which produce errors"},{"ordinal":69,"create_number":2227,"create_updated_at":"2022-07-21T18:24:19Z","create_title":"Middlewares should use errors.As() instead of type assertion on HTTPError","create_spec":"- Helps consumers who want to wrap HTTPError, and other use cases","pr_number":2219,"pr":"Fix case when routeNotFound handler is lost when new route is added to the router.\r\n\r\nThis happens when new handler we are registering has shorter and partially matching path as router node with routeNotFound handler. In that case when we split existing node into 2 parts we need to assign routeNotFound handler also to split node."},{"ordinal":70,"create_number":2218,"create_updated_at":"2022-07-11T19:03:09Z","create_title":"Make cleaner Bind func in bind.go","create_spec":"Trying to clean up bind.go file","pr_number":2217,"pr":"This PR adds support for registering handlers for 404 routes.  Echo instance and group has now method \r\n`RouteNotFound(path string, h HandlerFunc, m ...MiddlewareFunc) *Route` \r\nwhich registers handler for given path. Path supports any/path parameters and can be static (which is little bit silly but we still support it)\r\n\r\nCustom 404 handler has priority in router over global 404 handler and 405 handling logic.\r\n\r\nExample:\r\n```go\r\ne := echo.New()\r\n\r\ne.RouteNotFound(\"/*\", func(c echo.Context) error { return c.NoContent(http.StatusNotFound) })\r\n\r\ng := e.Group(\"/images\")\r\ng.RouteNotFound(\"/*\", func(c echo.Context) error { return c.NoContent(http.StatusNotFound) })\r\n// or\r\ng.Add(echo.RouteNotFound, \"/*\", func(c echo.Context) error { return c.NoContent(http.StatusNotFound) })\r\n```\r\nso you would have 2 404handlers in this example:\r\n* `/*`\r\n* `/images/*`\r\n\r\n----\r\nRouting performance is not significantly affected by this feature. \r\n\r\nBenchstat between current master branch and this branch. \r\n```\r\nx@x:~/code/echo$ benchstat benchmark_master2.txt benchmark_new2.txt \r\nname                                    old time/op    new time/op    delta\r\npkg:github.com/labstack/echo/v4 goos:linux goarch:amd64\r\nBindbindDataWithTags-6                    11.1µs ± 1%    11.1µs ± 0%    ~     (p=0.130 n=8+8)\r\nDefaultBinder_BindInt64_single-6           244ns ± 1%     243ns ± 1%    ~     (p=0.664 n=8+8)\r\nValueBinder_BindInt64_single-6            24.0ns ± 1%    23.8ns ± 1%    ~     (p=0.067 n=7+8)\r\nRawFunc_Int64_single-6                    12.9ns ± 1%    13.1ns ± 1%  +1.29%  (p=0.001 n=7+7)\r\nDefaultBinder_BindInt64_10_fields-6       2.30µs ± 1%    2.29µs ± 1%    ~     (p=0.168 n=8+8)\r\nValueBinder_BindInt64_10_fields-6          272ns ± 2%     275ns ± 1%  +1.32%  (p=0.021 n=8+8)\r\nAllocJSONP-6                               417ns ± 2%     397ns ± 1%  -4.80%  (p=0.000 n=8+8)\r\nAllocJSON-6                                280ns ± 1%     266ns ± 1%  -5.07%  (p=0.000 n=8+8)\r\nAllocXML-6                                1.69µs ± 2%    1.72µs ± 2%  +1.75%  (p=0.007 n=8+7)\r\nRealIPForHeaderXForwardFor-6              32.9ns ± 1%    32.6ns ± 1%  -0.93%  (p=0.043 n=8+7)\r\nContext_Store-6                           54.8ns ± 1%    55.0ns ± 2%    ~     (p=0.895 n=8+8)\r\nEchoStaticRoutes-6                        18.0µs ± 2%    17.1µs ± 2%  -4.70%  (p=0.000 n=8+8)\r\nEchoStaticRoutesMisses-6                  17.9µs ± 2%    17.3µs ± 2%  -2.91%  (p=0.000 n=8+8)\r\nEchoGitHubAPI-6                           31.7µs ± 1%    31.8µs ± 1%    ~     (p=0.442 n=8+8)\r\nEchoGitHubAPIMisses-6                     31.5µs ± 1%    31.8µs ± 2%  +0.98%  (p=0.014 n=7+8)\r\nEchoParseAPI-6                            2.04µs ± 1%    2.07µs ± 2%  +1.23%  (p=0.019 n=8+8)\r\nRouterStaticRoutes-6                      14.2µs ± 1%    13.7µs ± 1%  -3.20%  (p=0.000 n=8+8)\r\nRouterStaticRoutesMisses-6                 476ns ± 1%     479ns ± 1%    ~     (p=0.100 n=7+8)\r\nRouterGitHubAPI-6                         24.6µs ± 1%    24.3µs ± 1%  -1.13%  (p=0.000 n=8+7)\r\nRouterGitHubAPIMisses-6                    587ns ± 1%     584ns ± 2%    ~     (p=0.105 n=8+8)\r\nRouterParseAPI-6                          1.29µs ± 1%    1.25µs ± 1%  -3.17%  (p=0.000 n=8+8)\r\nRouterParseAPIMisses-6                     313ns ± 1%     315ns ± 1%    ~     (p=0.099 n=8+8)\r\nRouterGooglePlusAPI-6                      823ns ± 1%     817ns ± 0%  -0.70%  (p=0.005 n=8+6)\r\nRouterGooglePlusAPIMisses-6                470ns ± 1%     470ns ± 1%    ~     (p=0.702 n=8+8)\r\nRouterParamsAndAnyAPI-6                   2.02µs ± 1%    2.01µs ± 1%  -0.85%  (p=0.007 n=8+8)\r\npkg:github.com/labstack/echo/v4/middleware goos:linux goarch:amd64\r\nGzip-6                                    22.9µs ± 1%    22.9µs ± 0%    ~     (p=0.613 n=8+7)\r\nDecompress-6                              1.08µs ± 2%    1.14µs ± 6%  +5.18%  (p=0.014 n=8+8)\r\nLoggerWithConfig_withoutMapFields-6       3.02µs ±15%    3.09µs ± 6%    ~     (p=0.755 n=8+6)\r\nLoggerWithConfig_withMapFields-6          3.92µs ±26%    3.72µs ± 7%    ~     (p=0.505 n=8+8)\r\nRateLimiterMemoryStore_1000-6             2.37µs ± 3%    2.36µs ± 0%    ~     (p=0.143 n=8+7)\r\nRateLimiterMemoryStore_10000-6            2.45µs ± 1%    2.49µs ± 1%  +1.41%  (p=0.001 n=8+8)\r\nRateLimiterMemoryStore_100000-6           4.23µs ± 4%    4.36µs ± 1%  +3.04%  (p=0.000 n=8+8)\r\nRateLimiterMemoryStore_conc100_10000-6    29.2µs ± 3%    29.4µs ± 2%    ~     (p=0.645 n=8+8)\r\nRequestLogger_withoutMapFields-6          1.43µs ±16%    1.55µs ±15%    ~     (p=0.130 n=8+8)\r\nRequestLogger_withMapFields-6             2.81µs ±24%    2.97µs ±13%    ~     (p=0.161 n=8+8)\r\n\r\nname                                    old alloc/op   new alloc/op   delta\r\npkg:github.com/labstack/echo/v4 goos:linux goarch:amd64\r\nBindbindDataWithTags-6                    1.05kB ± 0%    1.05kB ± 0%    ~     (all equal)\r\nDefaultBinder_BindInt64_single-6           16.0B ± 0%     16.0B ± 0%    ~     (all equal)\r\nValueBinder_BindInt64_single-6             0.00B          0.00B         ~     (all equal)\r\nRawFunc_Int64_single-6                     0.00B          0.00B         ~     (all equal)\r\nDefaultBinder_BindInt64_10_fields-6         216B ± 0%      216B ± 0%    ~     (all equal)\r\nValueBinder_BindInt64_10_fields-6          0.00B          0.00B         ~     (all equal)\r\nAllocJSONP-6                                182B ± 2%      177B ± 2%  -2.75%  (p=0.002 n=8+8)\r\nAllocJSON-6                                 114B ± 1%      111B ± 1%  -2.82%  (p=0.001 n=6+8)\r\nAllocXML-6                                4.75kB ± 2%    4.82kB ± 2%  +1.49%  (p=0.013 n=8+8)\r\nEchoStaticRoutes-6                         0.00B          0.00B         ~     (all equal)\r\nEchoStaticRoutesMisses-6                   0.00B          0.00B         ~     (all equal)\r\nEchoGitHubAPI-6                            0.00B          0.00B         ~     (all equal)\r\nEchoGitHubAPIMisses-6                      0.00B          0.00B         ~     (all equal)\r\nEchoParseAPI-6                             0.00B          0.00B         ~     (all equal)\r\nRouterStaticRoutes-6                       0.00B          0.00B         ~     (all equal)\r\nRouterStaticRoutesMisses-6                 0.00B          0.00B         ~     (all equal)\r\nRouterGitHubAPI-6                          0.00B          0.00B         ~     (all equal)\r\nRouterGitHubAPIMisses-6                    0.00B          0.00B         ~     (all equal)\r\nRouterParseAPI-6                           0.00B          0.00B         ~     (all equal)\r\nRouterParseAPIMisses-6                     0.00B          0.00B         ~     (all equal)\r\nRouterGooglePlusAPI-6                      0.00B          0.00B         ~     (all equal)\r\nRouterGooglePlusAPIMisses-6                0.00B          0.00B         ~     (all equal)\r\nRouterParamsAndAnyAPI-6                    0.00B          0.00B         ~     (all equal)\r\npkg:github.com/labstack/echo/v4/middleware goos:linux goarch:amd64\r\nGzip-6                                    1.53kB ± 5%    1.53kB ± 3%    ~     (p=0.901 n=8+8)\r\nDecompress-6                              4.66kB ± 0%    4.66kB ± 0%    ~     (all equal)\r\nLoggerWithConfig_withoutMapFields-6       1.42kB ± 0%    1.42kB ± 0%    ~     (all equal)\r\nLoggerWithConfig_withMapFields-6          1.89kB ± 0%    1.89kB ± 0%    ~     (all equal)\r\nRequestLogger_withoutMapFields-6          1.34kB ± 0%    1.34kB ± 0%    ~     (all equal)\r\nRequestLogger_withMapFields-6             3.00kB ± 0%    3.00kB ± 0%    ~     (all equal)\r\n\r\nname                                    old allocs/op  new allocs/op  delta\r\npkg:github.com/labstack/echo/v4 goos:linux goarch:amd64\r\nBindbindDataWithTags-6                      51.0 ± 0%      51.0 ± 0%    ~     (all equal)\r\nDefaultBinder_BindInt64_single-6            2.00 ± 0%      2.00 ± 0%    ~     (all equal)\r\nValueBinder_BindInt64_single-6              0.00           0.00         ~     (all equal)\r\nRawFunc_Int64_single-6                      0.00           0.00         ~     (all equal)\r\nDefaultBinder_BindInt64_10_fields-6         13.0 ± 0%      13.0 ± 0%    ~     (all equal)\r\nValueBinder_BindInt64_10_fields-6           0.00           0.00         ~     (all equal)\r\nAllocJSONP-6                                4.00 ± 0%      4.00 ± 0%    ~     (all equal)\r\nAllocJSON-6                                 1.00 ± 0%      1.00 ± 0%    ~     (all equal)\r\nAllocXML-6                                  10.0 ± 0%      10.0 ± 0%    ~     (all equal)\r\nEchoStaticRoutes-6                          0.00           0.00         ~     (all equal)\r\nEchoStaticRoutesMisses-6                    0.00           0.00         ~     (all equal)\r\nEchoGitHubAPI-6                             0.00           0.00         ~     (all equal)\r\nEchoGitHubAPIMisses-6                       0.00           0.00         ~     (all equal)\r\nEchoParseAPI-6                              0.00           0.00         ~     (all equal)\r\nRouterStaticRoutes-6                        0.00           0.00         ~     (all equal)\r\nRouterStaticRoutesMisses-6                  0.00           0.00         ~     (all equal)\r\nRouterGitHubAPI-6                           0.00           0.00         ~     (all equal)\r\nRouterGitHubAPIMisses-6                     0.00           0.00         ~     (all equal)\r\nRouterParseAPI-6                            0.00           0.00         ~     (all equal)\r\nRouterParseAPIMisses-6                      0.00           0.00         ~     (all equal)\r\nRouterGooglePlusAPI-6                       0.00           0.00         ~     (all equal)\r\nRouterGooglePlusAPIMisses-6                 0.00           0.00         ~     (all equal)\r\nRouterParamsAndAnyAPI-6                     0.00           0.00         ~     (all equal)\r\npkg:github.com/labstack/echo/v4/middleware goos:linux goarch:amd64\r\nGzip-6                                      16.0 ± 0%      16.0 ± 0%    ~     (all equal)\r\nDecompress-6                                8.00 ± 0%      8.00 ± 0%    ~     (all equal)\r\nLoggerWithConfig_withoutMapFields-6         21.0 ± 0%      21.0 ± 0%    ~     (all equal)\r\nLoggerWithConfig_withMapFields-6            26.0 ± 0%      26.0 ± 0%    ~     (all equal)\r\nRequestLogger_withoutMapFields-6            14.0 ± 0%      14.0 ± 0%    ~     (all equal)\r\nRequestLogger_withMapFields-6               25.0 ± 0%      25.0 ± 0%    ~     (all equal)\r\n```"},{"ordinal":71,"create_number":2212,"create_updated_at":"2022-07-05T05:18:52Z","create_title":"Remove directory redirect from `echo.StaticDirectoryHandler`","create_spec":"This PR removes the directory redirect from `echo.StaticDirectoryHandler`.\r\n\r\nAs mentioned in the related issue (#2211), the redirect conflicts with the `RemoveTrailingSlash` middleware.\r\n\r\nI'm not sure why the redirect was added (maybe it's a legacy from earlier versions?) but I don't think its necessary.\r\n\r\nI'm tagging @aldas (git blame pointed to him) to correct me in case I'm missing something.\r\n","pr_number":2209,"pr":"Relates to issues  https://github.com/labstack/echo/issues/1726 and https://github.com/labstack/echo/issues/2201\r\n\r\nParameters and paths are now separated by methods within the same node. \r\nAdd new method Context, which store path, parameters, and handler.\r\nAdd test from https://github.com/labstack/echo/issues/1726. \r\n\r\nCC @aldas "},{"ordinal":72,"create_number":2208,"create_updated_at":"2022-06-29T12:50:09Z","create_title":"Allow different params' names in different methods","create_spec":"Relates to issues  https://github.com/labstack/echo/issues/1726 and https://github.com/labstack/echo/issues/2201\r\n\r\nParameters and paths are now separated by methods within the same node. \r\nAdd test from https://github.com/labstack/echo/issues/1726. \r\n\r\n\r\ncc @aldas ","pr_number":2207,"pr":"Closes issues #2201 and #1726\r\n\r\nWhat was done:\r\n1. Each node is now splitted into per-method handler with it's own ppath and pnames\r\n2. Test case from #1726 at [router_test.go](https://github.com/labstack/echo/pull/2207/files#diff-06b5e9693dbcea8c23abd4249e8b42aab7cb9affcc5366568e6a34b212c62ff5R824-R845)"},{"ordinal":73,"create_number":2206,"create_updated_at":"2022-07-05T05:04:18Z","create_title":"Add logger middleware template variables: `${time_unix_milli}` and `${time_unix_micro}`","create_spec":"This patch introduces two template variables `${time_unix_milli}` and `${time_unix_micro}` into the logger middleware.\r\n\r\nCurrently, there is no way to interpolate that UNIX milli and micro seconds timestamp in a log entry, and go 1.17 or later runtime supports the utility functions `time#UnixMilli()` and `time#UnixMicro()` so this patch adds them as well.\r\n\r\nsee also: https://github.com/golang/go/issues/44196","pr_number":2202,"pr":"Closed Echo pull request #2202 did not include a body."},{"ordinal":74,"create_number":2196,"create_updated_at":"2022-05-31T19:41:14Z","create_title":"refactor: defer gzip return to pool to prevent potential leaks","create_spec":"When using gzip compression, I have noticed much higher memory usage after some time. \r\n\r\nLooking at the code briefly, I think it is a good idea to defer `Put` in general\r\n\r\nps. other change is due to gofmt so please ignore","pr_number":2194,"pr":"refactor"},{"ordinal":75,"create_number":2191,"create_updated_at":"2022-05-27T16:44:52Z","create_title":"fix: basic auth invalid base64 string","create_spec":"fixes #2170","pr_number":2190,"pr":"This is the update for the issue #2188 \r\n\r\nAuthor should configure what key correspond with the value in context, like: `\"user_id\":\"${id_custom}\",`\r\nhttps://github.com/noritama73/echo/blob/9847b661126507d90d6cbe25b2f4425ad36dd2e6/middleware/logger_test.go#L185-L193\r\n\r\n### Discussion\r\n\r\n* Temporarily, the value in context is casted to string, but is this procedure correct?\r\nhttps://github.com/noritama73/echo/blob/9847b661126507d90d6cbe25b2f4425ad36dd2e6/middleware/logger.go#L214\r\n* Also now I casted the value to only `string`, but can this flamework support that the type is casted provided by author?\r\n```\r\n...\r\nif contextKey, ok := config.CustomContextMap[tag]; ok {\r\n\tcustomContext, valid := c.Get(contextKey).({any type provided by author})\r\n\tif valid {\r\n\t\treturn buf.WriteString(customContext)\r\n\t}\r\n}\r\n...\r\n```\r\n* To show the value is customed by author, should I add any prefix like \"header:\"?\r\nhttps://github.com/noritama73/echo/blob/9847b661126507d90d6cbe25b2f4425ad36dd2e6/middleware/logger.go#L200\r\n"},{"ordinal":76,"create_number":2187,"create_updated_at":"2022-07-12T19:03:32Z","create_title":"Timeout mw: rework how test waits for timeout. Using sleep as delay i…","create_spec":"Timeout mw: rework how test waits for timeout. Using sleep as delay is problematic when CI worker is slower than usual. Also make sure we wait logger middleware to be actually executed before we start asserting logger buffer contents.\r\n\r\nStill having problems with macos-latest @ Go 1.18 runs\r\n\r\n```\r\n2022-05-21 19:29:13.081172 +0000 UTC m=+0.616594200\r\n--- FAIL: TestTimeoutWithFullEchoStack (0.14s)\r\n    --- FAIL: TestTimeoutWithFullEchoStack/503_-_handler_timeouts,_write_response_in_timeout_middleware (0.08s)\r\n        timeout_test.go:417: \r\n            \tError Trace:\ttimeout_test.go:417\r\n            \tError:      \tShould be false\r\n            \tTest:       \tTestTimeoutWithFullEchoStack/503_-_handler_timeouts,_write_response_in_timeout_middleware\r\n```","pr_number":2186,"pr":"another try with #2185\r\n\r\nTimeout mw: fix datarace in tests when we are getting data from buffer (in test) and writing to logger at the same time."},{"ordinal":77,"create_number":2185,"create_updated_at":"2022-07-12T19:03:37Z","create_title":"Timeout mw: fix datarace in tests when we are getting data from logger buffer","create_spec":"Fix Timeout middleware \"full stack\" tests: there is datarace in tests when we are getting data from logger buffer. Run each test in their own server so multiple tests cases will not cause datarace getting data out of logger buffer.\r\n\r\nThis is quite brute force fix -  we do not spawn one server for all tests, we instead spawn server for each testcase.\r\n\r\n```\r\n==================\r\nWARNING: DATA RACE\r\nRead at 0x00c000134120 by goroutine 85:\r\n  bytes.(*Buffer).String()\r\n      /Users/runner/hostedtoolcache/go/1.18.2/x64/src/bytes/buffer.go:65 +0x35e\r\n  github.com/labstack/echo/v4/middleware.TestTimeoutWithFullEchoStack.func2()\r\n      /Users/runner/work/echo/echo/middleware/timeout_test.go:413 +0x3a9\r\n  testing.tRunner()\r\n      /Users/runner/hostedtoolcache/go/1.18.2/x64/src/testing/testing.go:1439 +0x213\r\n  testing.(*T).Run.func1()\r\n      /Users/runner/hostedtoolcache/go/1.18.2/x64/src/testing/testing.go:1486 +0x47\r\n\r\nPrevious write at 0x00c000134120 by goroutine 25:\r\n  bytes.(*Buffer).tryGrowByReslice()\r\n      /Users/runner/hostedtoolcache/go/1.18.2/x64/src/bytes/buffer.go:108 +0xb3\r\n  bytes.(*Buffer).Write()\r\n      /Users/runner/hostedtoolcache/go/1.18.2/x64/src/bytes/buffer.go:170 +0x18\r\n  github.com/labstack/echo/v4/middleware.LoggerWithConfig.func2.1()\r\n      /Users/runner/work/echo/echo/middleware/logger.go:216 +0x64c\r\n  github.com/labstack/echo/v4/middleware.echoHandlerFuncWrapper.ServeHTTP()\r\n      /Users/runner/work/echo/echo/middleware/timeout.go:164 +0x241\r\n  github.com/labstack/echo/v4/middleware.(*echoHandlerFuncWrapper).ServeHTTP()\r\n      <autogenerated>:1 +0xcd\r\n  net/http.(*timeoutHandler).ServeHTTP.func1()\r\n      /Users/runner/hostedtoolcache/go/1.18.2/x64/src/net/http/server.go:3374 +0xe1\r\n\r\nGoroutine 85 (running) created at:\r\n  testing.(*T).Run()\r\n      /Users/runner/hostedtoolcache/go/1.18.2/x64/src/testing/testing.go:1486 +0x724\r\n  github.com/labstack/echo/v4/middleware.TestTimeoutWithFullEchoStack()\r\n      /Users/runner/work/echo/echo/middleware/timeout_test.go:397 +0xa15\r\n  testing.tRunner()\r\n      /Users/runner/hostedtoolcache/go/1.18.2/x64/src/testing/testing.go:1439 +0x213\r\n  testing.(*T).Run.func1()\r\n      /Users/runner/hostedtoolcache/go/1.18.2/x64/src/testing/testing.go:1486 +0x47\r\n\r\nGoroutine 25 (finished) created at:\r\n  net/http.(*timeoutHandler).ServeHTTP()\r\n      /Users/runner/hostedtoolcache/go/1.18.2/x64/src/net/http/server.go:3368 +0x812\r\n  github.com/labstack/echo/v4/middleware.TimeoutConfig.ToMiddleware.func1.1()\r\n      /Users/runner/work/echo/echo/middleware/timeout.go:125 +0x467\r\n  github.com/labstack/echo/v4.(*Echo).ServeHTTP()\r\n      /Users/runner/work/echo/echo/echo.go:630 +0x844\r\n  net/http.serverHandler.ServeHTTP()\r\n      /Users/runner/hostedtoolcache/go/1.18.2/x64/src/net/http/server.go:2916 +0x896\r\n  net/http.(*conn).serve()\r\n      /Users/runner/hostedtoolcache/go/1.18.2/x64/src/net/http/server.go:1966 +0xbaa\r\n  net/http.(*Server).Serve.func3()\r\n      /Users/runner/hostedtoolcache/go/1.18.2/x64/src/net/http/server.go:3071 +0x58\r\n==================\r\n==================\r\nWARNING: DATA RACE\r\nRead at 0x00c000144240 by goroutine 85:\r\n  runtime.slicebytetostring()\r\n      /Users/runner/hostedtoolcache/go/1.18.2/x64/src/runtime/string.go:81 +0x0\r\n  bytes.(*Buffer).String()\r\n      /Users/runner/hostedtoolcache/go/1.18.2/x64/src/bytes/buffer.go:65 +0x3a8\r\n  github.com/labstack/echo/v4/middleware.TestTimeoutWithFullEchoStack.func2()\r\n      /Users/runner/work/echo/echo/middleware/timeout_test.go:413 +0x3a9\r\n  testing.tRunner()\r\n      /Users/runner/hostedtoolcache/go/1.18.2/x64/src/testing/testing.go:1439 +0x213\r\n  testing.(*T).Run.func1()\r\n      /Users/runner/hostedtoolcache/go/1.18.2/x64/src/testing/testing.go:1486 +0x47\r\n\r\nPrevious write at 0x00c000144240 by goroutine 25:\r\n  runtime.slicecopy()\r\n      /Users/runner/hostedtoolcache/go/1.18.2/x64/src/runtime/slice.go:295 +0x0\r\n  bytes.(*Buffer).Write()\r\n      /Users/runner/hostedtoolcache/go/1.18.2/x64/src/bytes/buffer.go:174 +0x126\r\n  github.com/labstack/echo/v4/middleware.LoggerWithConfig.func2.1()\r\n      /Users/runner/work/echo/echo/middleware/logger.go:216 +0x64c\r\n  github.com/labstack/echo/v4/middleware.echoHandlerFuncWrapper.ServeHTTP()\r\n      /Users/runner/work/echo/echo/middleware/timeout.go:164 +0x241\r\n  github.com/labstack/echo/v4/middleware.(*echoHandlerFuncWrapper).ServeHTTP()\r\n      <autogenerated>:1 +0xcd\r\n  net/http.(*timeoutHandler).ServeHTTP.func1()\r\n      /Users/runner/hostedtoolcache/go/1.18.2/x64/src/net/http/server.go:3374 +0xe1\r\n\r\nGoroutine 85 (running) created at:\r\n  testing.(*T).Run()\r\n      /Users/runner/hostedtoolcache/go/1.18.2/x64/src/testing/testing.go:1486 +0x724\r\n  github.com/labstack/echo/v4/middleware.TestTimeoutWithFullEchoStack()\r\n      /Users/runner/work/echo/echo/middleware/timeout_test.go:397 +0xa15\r\n  testing.tRunner()\r\n      /Users/runner/hostedtoolcache/go/1.18.2/x64/src/testing/testing.go:1439 +0x213\r\n  testing.(*T).Run.func1()\r\n      /Users/runner/hostedtoolcache/go/1.18.2/x64/src/testing/testing.go:1486 +0x47\r\n\r\nGoroutine 25 (finished) created at:\r\n  net/http.(*timeoutHandler).ServeHTTP()\r\n      /Users/runner/hostedtoolcache/go/1.18.2/x64/src/net/http/server.go:3368 +0x812\r\n  github.com/labstack/echo/v4/middleware.TimeoutConfig.ToMiddleware.func1.1()\r\n      /Users/runner/work/echo/echo/middleware/timeout.go:125 +0x467\r\n  github.com/labstack/echo/v4.(*Echo).ServeHTTP()\r\n      /Users/runner/work/echo/echo/echo.go:630 +0x844\r\n  net/http.serverHandler.ServeHTTP()\r\n      /Users/runner/hostedtoolcache/go/1.18.2/x64/src/net/http/server.go:2916 +0x896\r\n  net/http.(*conn).serve()\r\n      /Users/runner/hostedtoolcache/go/1.18.2/x64/src/net/http/server.go:1966 +0xbaa\r\n  net/http.(*Server).Serve.func3()\r\n      /Users/runner/hostedtoolcache/go/1.18.2/x64/src/net/http/server.go:3071 +0x58\r\n==================\r\n--- FAIL: TestTimeoutWithFullEchoStack (0.13s)\r\n    --- FAIL: TestTimeoutWithFullEchoStack/503_-_handler_timeouts,_write_response_in_timeout_middleware (0.12s)\r\n        timeout_test.go:418: \r\n            \tError Trace:\ttimeout_test.go:418\r\n            \tError:      \tShould be false\r\n            \tTest:       \tTestTimeoutWithFullEchoStack/503_-_handler_timeouts,_write_response_in_timeout_middleware\r\n        testing.go:1312: race detected during execution of test\r\n    testing.go:1312: race detected during execution of test\r\n{\"time\":\"2022-05-21T16:18:[27](https://github.com/aldas/echo/runs/6538053775?check_suite_focus=true#step:5:28).[28](https://github.com/aldas/echo/runs/6538053775?check_suite_focus=true#step:5:29)8656Z\",\"level\":\"-\",\"prefix\":\"echo\",\"file\":\"recover.go\",\"line\":\"113\",\"message\":\"[PANIC RECOVER] panic!!! goroutine 495 [running]:\\ngithub.com/labstack/echo/v4/middleware.RecoverWithConfig.func1.1.1()\\n\\t/Users/runner/work/echo/echo/middleware/recover.go:93 +0x2ba\\npanic({0x1833500, 0x1a00fc0})\\n\\t/Users/runner/hostedtoolcache/go/1.18.2/x64/src/runtime/panic.go:844 +0x258\\nnet/http.(*timeoutHandler).ServeHTTP(0xc0002182c0, {0x1a06210, 0xc0003a02a0}, 0xc000540800)\\n\\t/Users/runner/hostedtoolcache/go/1.18.2/x64/src/net/http/server.go:3379 +0xea5\\ngithub.com/labstack/echo/v4/middleware.TimeoutConfig.ToMiddleware.func1.1({0x1a0b9c0, 0xc000520280})\\n\\t/Users/runner/work/echo/echo/middleware/timeout.go:125 +0x468\\ngithub.com/labstack/echo/v4/middleware.RecoverWithConfig.func1.1({0x1a0b9c0, 0xc000520280})\\n\\t/Users/runner/work/echo/echo/middleware/recover.go:119 +0x165\\ngithub.com/labstack/echo/v4.(*Echo).ServeHTTP(0xc000103680, {0x1a06f60?, 0xc000218240}, 0xc000540800)\\n\\t/Users/runner/work/echo/echo/echo.go:6[30](https://github.com/aldas/echo/runs/6538053775?check_suite_focus=true#step:5:31) +0x845\\ngithub.com/labstack/echo/v4/middleware.TestTimeoutRecoversPanic.func2()\\n\\t/Users/runner/work/echo/echo/middleware/timeout_test.go:195 +0x54\\ngithub.com/stretchr/testify/assert.didPanic.func1(0xc0003c5ce0, 0xc0003c5ca6, 0xc0003c5cd0, 0xc0003a0280)\\n\\t/Users/runner/go/pkg/mod/github.com/stretchr/testify@v1.7.0/assert/assertions.go:1018 +0xa9\\ngithub.com/stretchr/testify/assert.didPanic(0xc00056e4e0?)\\n\\t/Users/runner/go/pkg/mod/github.com/stretchr/testify@v1.7.0/assert/assertions.go:1020 +0x51\\ngithub.com/stretchr/testify/assert.NotPanics({0x1a03680, 0xc00056e4e0}, 0xc0003a0280, {0x0, 0x0, 0x0})\\n\\t/Users/runner/go/pkg/mod/github.com/stretchr/testify@v1.7.0/assert/assertions.go:1091 +0x89\\ngithub.com/labstack/echo/v4/middleware.TestTimeoutRecoversPanic(0x0?)\\n\\t/Users/runner/work/echo/echo/middleware/timeout_test.go:194 +0x55e\\ntesting.tRunner(0xc00056e4e0, 0x1930948)\\n\\t/Users/runner/hostedtoolcache/go/1.18.2/x64/src/testing/testing.go:1439 +0x214\\ncreated by testing.(*T).Run\\n\\t/Users/runner/hostedtoolcache/go/1.18.2/x64/src/testing/testing.go:1486 +0x725\\n\\ngoroutine 1 [chan receive]:\\ntesting.tRunner.func1()\\n\\t/Users/runner/hostedtoolcache/go/1.18.2/x64/src/testing/testing.go:1405 +0x865\\ntesting.tRunner(0xc000140[34](https://github.com/aldas/echo/runs/6538053775?check_suite_focus=true#step:5:35)0, 0xc0000f9b68)\\n\\t/Users/runner/hostedtoolcache/go/1.18.2/x64/src/testing/testing.go:1445 +0x255\\ntesting.runTests(0xc0000c7d60?, {0x1d8a1c0, 0x7f, 0x7f}, {0x40?, 0xa5b13c0?, 0x1d8ef00?})\\n\\t/Users/runner/hostedtoolcache/go/1.18.2/x64/src/testing/testing.go:18[37](https://github.com/aldas/echo/runs/6538053775?check_suite_focus=true#step:5:38) +0x7e5\\ntesting.(*M).Run(0xc0000c7d60)\\n\\t/Users/runner/hostedtoolcache/go/1.18.2/x64/src/testing/testing.go:1719 +0xa72\\nmain.main()\\n\\t_testmain.go:415 +0x3aa\\n\\ngoroutine 489 [runnable]:\\ntime.Sleep(0x61a8)\\n\\t/Users/runner/hostedtoolcache/go/1.18.2/x64/src/runtime/time.go:194 +0x12e\\ngithub.com/labstack/echo/v4/middleware.TestTimeoutSkipper.func2({0x203000?, 0xc000[38](https://github.com/aldas/echo/runs/6538053775?check_suite_focus=true#step:5:39)ac30?})\\n\\t/Users/runner/work/echo/echo/middleware/timeout_test.go:38 +0x2f\\ngithub.com/labstack/echo/v4/middleware.TimeoutConfig.ToMiddleware.func1.1({0x1a0b9c0, 0xc0005741e0})\\n\\t/Users/runner/work/echo/echo/middleware/timeout.go:113 +0xe3\\ngithub.com/labstack/echo/v4/middleware.TestTimeoutSkipper(0x0?)\\n\\t/Users/runner/work/echo/echo/middleware/timeout_test.go:40 +0x655\\ntesting.tRunner(0xc000082d00, 0x1930960)\\n\\t/Users/runner/hostedtoolcache/go/1.18.2/x64/src/testing/testing.go:14[39](https://github.com/aldas/echo/runs/6538053775?check_suite_focus=true#step:5:40) +0x214\\ncreated by testing.(*T).Run\\n\\t/Users/runner/hostedtoolcache/go/1.18.2/x64/src/testing/testing.go:1486 +0x725\\n\\ngoroutine 205 [select]:\\nnet/http.(*persistConn).readLoop(0xc0002dab[40](https://github.com/aldas/echo/runs/6538053775?check_suite_focus=true#step:5:41))\\n\\t/Users/runner/hostedtoolcache/go/1.18.2/x64/src/net/http/transport.go:2213 +0x14d1\\ncreated by net/http.(*Transport).dialConn\\n\\t/Users/runner/hostedtoolcache/go/1.18.2/x64/src/net/http/transport.go:1750 +0x24c5\\n\\ngoroutine 206 [select]:\\nnet/http.(*persistConn).writeLoop(0xc0002dab40)\\n\\t/Users/runner/hostedtoolcache/go/1.18.2/x64/src/net/http/transport.go:2392 +0x1a5\\ncreated by net/http.(*Transport).dialConn\\n\\t/Users/runner/hostedtoolcache/go/1.18.2/x64/src/net/http/transport.go:1751 +0x2556\\n\\ngoroutine [49](https://github.com/aldas/echo/runs/6538053775?check_suite_focus=true#step:5:50)0 [runnable]:\\ntesting.(*T).Parallel(0xc000082ea0)\\n\\t/Users/runner/hostedtoolcache/go/1.18.2/x[64](https://github.com/aldas/echo/runs/6538053775?check_suite_focus=true#step:5:65)/src/testing/testing.go:[126](https://github.com/aldas/echo/runs/6538053775?check_suite_focus=true#step:5:127)4 +0x4e6\\ngithub.com/labstack/echo/v4/middleware.TestTimeoutWit\\n\"}\r\n```","pr_number":2182,"pr":"Some loadbalancers (eg citrix ADC / netscaler) add square brackets around the ipv6 address in the X-Forwarded-For header.\r\nThis PR removes them so that RealIP() and friends work correctly."},{"ordinal":78,"create_number":2176,"create_updated_at":"2022-05-27T17:15:58Z","create_title":"fix: duplicated findStaticChild process at findChildWithLabel","create_spec":"duplicated findStaticChild process at findChildWithLabel in router.go\r\n\r\nI think less duplicate processing more readable except  performance decrement.","pr_number":2173,"pr":"#1610 \r\n\r\nWebDAV support by adding method : `MKCOL`,`COPY `,`MOVE `,`LOCK`,`UNLOCK`,  `PROPPATCH`, no  dependencies added.\r\n\r\n**Working code using echo & golang.org/x/net/webdav**\r\n\r\n```\r\npackage main\r\n\r\nimport (\r\n\t\"flag\"\r\n\t\"fmt\"\r\n\t\"os\"\r\n\r\n\t\"github.com/labstack/echo/v4\"\r\n\t\"golang.org/x/net/webdav\"\r\n)\r\n\r\nvar (\r\n\tflagRootDir  = flag.String(\"dir\", \"\", \"webdav root dir\")\r\n\tflagHttpAddr = flag.String(\"http\", \":8080\", \"http or https address\")\r\n)\r\n\r\nfunc init() {\r\n\tflag.Usage = func() {\r\n\t\tfmt.Fprintf(os.Stderr, \"Usage of WebDAV Server\\n\")\r\n\t\tflag.PrintDefaults()\r\n\t}\r\n}\r\nfunc main() {\r\n\tflag.Parse()\r\n\tfs := &webdav.Handler{\r\n\t\tFileSystem: webdav.Dir(*flagRootDir),\r\n\t\tLockSystem: webdav.NewMemLS(),\r\n\t}\r\n\r\n\te := echo.New()\r\n\te.Pre(func(next echo.HandlerFunc) echo.HandlerFunc {\r\n\t\treturn func(c echo.Context) error {\r\n\t\t\tfmt.Println(c.Request())\r\n\t\t\treturn next(c)\r\n\t\t}\r\n\t})\r\n\techoHandle := echo.WrapHandler(fs)\r\n\te.Any(\"/*\", echoHandle)\r\n\tfmt.Println(e.Start(*flagHttpAddr))\r\n}\r\n\r\n```\r\n\r\nSigned-off-by: yixy <youzhilane01@gmail.com>"},{"ordinal":79,"create_number":2172,"create_updated_at":"2022-06-15T04:44:41Z","create_title":"Implement matching pattern routing . support AWS API","create_spec":" I want to use echo to support AWS API .  so I submitted these codes ","pr_number":2166,"pr":"Update Github CI flow to use Go 1.18, bump action versions.\r\n\r\nNote: Github flow used `go get -u golang.org/x/lint/golint@latest` but from Go 1.18 this way of installing commands is removed and `go install golang.org/x/lint/golint@latest` should be used. But go install is supported only from Go 1.16. So I dropped Go 1.15  from github flow at the moment."},{"ordinal":80,"create_number":2162,"create_updated_at":"2022-12-11T16:57:10Z","create_title":"Add route to request log","create_spec":"This PR adds `route` to tags used for request logging to be able to log the route pattern.\r\nThere is a usecase that we want to log the route to aggregate the request in monitoring.","pr_number":2160,"pr":"Add a third-party middleware library to the README to support generating Swagger API documentation in code.\r\n\r\nThis is a `coding` way to generate Swagger API documentation, there may be some groups who prefer coding instead of comments"},{"ordinal":81,"create_number":2159,"create_updated_at":"2024-03-06T20:05:54Z","create_title":"Able to set NoColor for echo.Logger and echo.middleware.Logger","create_spec":"close #1871 ","pr_number":2155,"pr":"Better to upgrade to JWT library v4 which is better than original one. Also, there is a deprecation of `StandardClaims`\r\n\r\n`\r\nStandardClaims are a structured version of the JWT Claims Set, as referenced at https://datatracker.ietf.org/doc/html/rfc7519#section-4. They do not follow the specification exactly, since they were based on an earlier draft of the specification and not updated. The main difference is that they only support integer-based date fields and singular audiences. This might lead to incompatibilities with other JWT implementations. The use of this is discouraged, instead the newer RegisteredClaims struct should be used.\r\n`"},{"ordinal":82,"create_number":2154,"create_updated_at":"2022-04-07T18:37:11Z","create_title":"Add httptest.Server to Echo struct","create_spec":"This pull request adds a pointer to an `httptest.Server` struct to the `Echo` struct, available via the `TestServer` field. This allows testing of handlers in a similar fashion to the typical httptest package. \r\n\r\n[Example usage.](https://gist.github.com/albrazeau/bc73d6126558451abf5f85b371a16c2e)\r\n\r\nI am not sure where would add an example/docs, but would be happy to clean up the above gist as needed and put it there.","pr_number":2145,"pr":"Hello, the four test items in Testbodylimit are actually tests on content length, not content read, and there is no difference between the first two test items and the latter two. After modification, the latter two test items can test content read."},{"ordinal":83,"create_number":2144,"create_updated_at":"2022-03-21T15:42:25Z","create_title":"Fix body_limit middleware unit test","create_spec":"Hello, the four test items in Testbodylimit are actually tests on content length, not content read, and there is no difference between the first two test items and the latter two. After modification, the latter two test items can test content read.","pr_number":2137,"pr":"Update version and changelog for 4.7.2"},{"ordinal":84,"create_number":2136,"create_updated_at":"2022-07-12T19:04:01Z","create_title":"fix CSRF middleware not being able to extract token from `multipart/form-data` form","create_spec":"Fix for #2135 CSRF middleware not being able to extract token from `multipart/form-data` form\r\n\r\nThis can be tested by hand:\r\n```go\r\nfunc main() {\r\n\te := echo.New()\r\n\te.Use(middleware.Logger())\r\n\te.Use(middleware.Recover())\r\n\r\n\te.Use(middleware.CSRFWithConfig(middleware.CSRFConfig{\r\n\t\tTokenLookup: \"form:csrf\",\r\n\t}))\r\n\r\n\te.POST(\"/form\", func(c echo.Context) error {\r\n\t\tcsrf, ok := c.Get(\"csrf\").(string)\r\n\t\tif !ok {\r\n\t\t\treturn echo.NewHTTPError(http.StatusBadRequest, \"missing CSRF value\")\r\n\t\t}\r\n\t\treturn c.String(http.StatusCreated, csrf)\r\n\t})\r\n\r\n\tif err := e.Start(\":8080\"); err != http.ErrServerClosed {\r\n\t\tlog.Fatal(err)\r\n\t}\r\n}\r\n```\r\n\r\n\r\n```bash\r\necho \"<div>hi</div>\" > test.html\r\n\r\ncurl --trace-ascii /dev/stdout \\\r\n    --cookie \"_csrf=test\" \\\r\n    -F csrf=test \\\r\n    -F upload=@test.html \\\r\n    http://localhost:8080/form\r\n```\r\n\r\noutput of curl after fix\r\n```bash\r\nx@x:~/code/echo/main$ curl --trace-ascii /dev/stdout --cookie \"_csrf=test\" -F csrf=test -F upload=@test.html http://localhost:8080/form\r\n== Info:   Trying 127.0.0.1:8080...\r\n== Info: Connected to localhost (127.0.0.1) port 8080 (#0)\r\n=> Send header, 210 bytes (0xd2)\r\n0000: POST /form HTTP/1.1\r\n0015: Host: localhost:8080\r\n002b: User-Agent: curl/7.74.0\r\n0044: Accept: */*\r\n0051: Cookie: _csrf=test\r\n0065: Content-Length: 299\r\n007a: Content-Type: multipart/form-data; boundary=--------------------\r\n00ba: ----a297ba70c335a0d7\r\n00d0: \r\n=> Send data, 299 bytes (0x12b)\r\n0000: --------------------------a297ba70c335a0d7\r\n002c: Content-Disposition: form-data; name=\"csrf\"\r\n0059: \r\n005b: test\r\n0061: --------------------------a297ba70c335a0d7\r\n008d: Content-Disposition: form-data; name=\"upload\"; filename=\"test.ht\r\n00cd: ml\"\r\n00d2: Content-Type: text/html\r\n00eb: \r\n00ed: <div>hi</div>.\r\n00fd: --------------------------a297ba70c335a0d7--\r\n== Info: We are completely uploaded and fine\r\n== Info: Mark bundle as not supporting multiuse\r\n<= Recv header, 22 bytes (0x16)\r\n0000: HTTP/1.1 201 Created\r\n<= Recv header, 41 bytes (0x29)\r\n0000: Content-Type: text/plain; charset=UTF-8\r\n<= Recv header, 63 bytes (0x3f)\r\n0000: Set-Cookie: _csrf=test; Expires=Wed, 16 Mar 2022 19:00:31 GMT\r\n<= Recv header, 14 bytes (0xe)\r\n0000: Vary: Cookie\r\n<= Recv header, 37 bytes (0x25)\r\n0000: Date: Tue, 15 Mar 2022 19:00:31 GMT\r\n<= Recv header, 19 bytes (0x13)\r\n0000: Content-Length: 4\r\n<= Recv header, 2 bytes (0x2)\r\n0000: \r\n<= Recv data, 4 bytes (0x4)\r\n0000: test\r\n== Info: Connection #0 to host localhost left intact\r\ntest\r\n```","pr_number":2134,"pr":"This is the fix for the issue #2133 \r\n\r\nThe fix is in explicit check in recover middleware defer function to re-throw (panic) the `http.ErrAbortHandler` error.\r\n\r\nThis specific error is recovered in `net/http/server.go` and per default ignored for logging.\r\nhttps://github.com/golang/go/blob/88be85f18bf0244a2470fdf6719e1b5ca5a5e50a/src/net/http/server.go#L1799"},{"ordinal":85,"create_number":2131,"create_updated_at":"2022-03-15T19:38:57Z","create_title":"fix nil pointer exception when calling Start again after address binding error","create_spec":"There is a nil pointer exception because the code assigns a nil value to an interface (`e.Listener`) which ends up creating a non-nil interface containing a nil pointer. Using this will result in the nil pointer exception and checking the `e.Listener` against `nil` will be `false` -- which is why the Listener is not recreated and overwritten the during the second call to `Start`.\r\n\r\nSmall code example that triggers this error (given an existing echo router):\r\n\r\n```go\r\nif err := e.Start(\":6060\"); err != nil {\r\n\t// retry with random port\r\n\tif r := e.Start(\":\"); r != nil {\r\n\t\tlog.Fatal(r)\r\n\t}\r\n}\r\n```\r\n\r\n","pr_number":2128,"pr":"Update version and changelog for 4.7.1\r\n\r\nSee: https://github.com/labstack/echo/issues/2117"},{"ordinal":86,"create_number":2127,"create_updated_at":"2022-07-12T19:04:30Z","create_title":"Add new value binding methods (UnixTimeMilli,TextUnmarshaler,JSONUnmarshaler) to Valuebinder","create_spec":"Add new value binding methods (UnixTimeMilli,TextUnmarshaler,JSONUnmarshaler) to Valuebinder. As it adds only new methods to struct it is backwards compatible change.\r\n\r\n* `UnixTimeMilli` binds `1647184410140` to `2022-03-13T15:13:30.140000000+00:00` useful when dealing with Javascript\r\n\r\n* `TextUnmarshaler` binds value to destination implementing `encoding.TextUnmarshaler` interface\r\n* `JSONUnmarshaler` binds value to destination implementing `json.Unmarshaler` interface\r\n\r\nRequested in https://github.com/labstack/echo/discussions/2000#discussioncomment-2315815","pr_number":2126,"pr":"Timeout middleware occasionally fails with data race in Github CI flow. I have taken code from `v5` branch which has a little bit reworked logic so we are not using `t.ctx.Error(err)` anymore\r\n\r\n```text\r\n==================\r\nWARNING: DATA RACE\r\nRead at 0x00c0001e8ca8 by goroutine 70:\r\n  bytes.(*Buffer).String()\r\n      /Users/runner/hostedtoolcache/go/1.15.15/x64/src/bytes/buffer.go:65 +0x3ca\r\n  github.com/labstack/echo/v4/middleware.TestTimeoutWithFullEchoStack.func2()\r\n      /Users/runner/work/echo/echo/middleware/timeout_test.go:408 +0x3b9\r\n  testing.tRunner()\r\n      /Users/runner/hostedtoolcache/go/1.15.15/x64/src/testing/testing.go:1123 +0x202\r\n\r\nPrevious write at 0x00c0001e8ca8 by goroutine 76:\r\n  bytes.(*Buffer).grow()\r\n      /Users/runner/hostedtoolcache/go/1.15.15/x64/src/bytes/buffer.go:147 +0x2d7\r\n  bytes.(*Buffer).Write()\r\n      /Users/runner/hostedtoolcache/go/1.15.15/x64/src/bytes/buffer.go:172 +0x184\r\n  github.com/labstack/echo/v4/middleware.LoggerWithConfig.func2.1()\r\n      /Users/runner/work/echo/echo/middleware/logger.go:216 +0x724\r\n  github.com/labstack/echo/v4/middleware.echoHandlerFuncWrapper.ServeHTTP()\r\n      /Users/runner/work/echo/echo/middleware/timeout.go:156 +0x22c\r\n  github.com/labstack/echo/v4/middleware.(*echoHandlerFuncWrapper).ServeHTTP()\r\n      <autogenerated>:1 +0xcc\r\n  net/http.(*timeoutHandler).ServeHTTP.func1()\r\n      /Users/runner/hostedtoolcache/go/1.15.15/x64/src/net/http/server.go:[32](https://github.com/labstack/echo/runs/5527738819?check_suite_focus=true#step:7:32)72 +0xb5\r\n\r\nGoroutine 70 (running) created at:\r\n  testing.(*T).Run()\r\n      /Users/runner/hostedtoolcache/go/1.15.15/x64/src/testing/testing.go:1168 +0x5bb\r\n  github.com/labstack/echo/v4/middleware.TestTimeoutWithFullEchoStack()\r\n      /Users/runner/work/echo/echo/middleware/timeout_test.go:392 +0xa3b\r\n  testing.tRunner()\r\n      /Users/runner/hostedtoolcache/go/1.15.15/x64/src/testing/testing.go:1123 +0x202\r\n\r\nGoroutine 76 (finished) created at:\r\n  net/http.(*timeoutHandler).ServeHTTP()\r\n      /Users/runner/hostedtoolcache/go/1.15.15/x64/src/net/http/server.go:3266 +0x38f\r\n  github.com/labstack/echo/v4/middleware.TimeoutWithConfig.func1.1()\r\n      /Users/runner/work/echo/echo/middleware/timeout.go:118 +0x449\r\n  github.com/labstack/echo/v4.(*Echo).ServeHTTP()\r\n      /Users/runner/work/echo/echo/echo.go:630 +0x1fd\r\n  net/http.serverHandler.ServeHTTP()\r\n      /Users/runner/hostedtoolcache/go/1.15.15/x64/src/net/http/server.go:2843 +0xca\r\n  net/http.(*conn).serve()\r\n      /Users/runner/hostedtoolcache/go/1.15.15/x64/src/net/http/server.go:1925 +0x84c\r\n==================\r\n==================\r\nWARNING: DATA RACE\r\nRead at 0x00c0001e8c90 by goroutine 70:\r\n  bytes.(*Buffer).String()\r\n      /Users/runner/hostedtoolcache/go/1.15.15/x64/src/bytes/buffer.go:65 +0x3e4\r\n  github.com/labstack/echo/v4/middleware.TestTimeoutWithFullEchoStack.func2()\r\n      /Users/runner/work/echo/echo/middleware/timeout_test.go:408 +0x3b9\r\n  testing.tRunner()\r\n      /Users/runner/hostedtoolcache/go/1.15.15/x64/src/testing/testing.go:1123 +0x202\r\n\r\nPrevious write at 0x00c0001e8c90 by goroutine 76:\r\n  bytes.(*Buffer).grow()\r\n      /Users/runner/hostedtoolcache/go/1.15.15/x64/src/bytes/buffer.go:144 +0x297\r\n  bytes.(*Buffer).Write()\r\n      /Users/runner/hostedtoolcache/go/1.15.15/x64/src/bytes/buffer.go:172 +0x184\r\n  github.com/labstack/echo/v4/middleware.LoggerWithConfig.func2.1()\r\n      /Users/runner/work/echo/echo/middleware/logger.go:216 +0x724\r\n  github.com/labstack/echo/v4/middleware.echoHandlerFuncWrapper.ServeHTTP()\r\n      /Users/runner/work/echo/echo/middleware/timeout.go:156 +0x22c\r\n  github.com/labstack/echo/v4/middleware.(*echoHandlerFuncWrapper).ServeHTTP()\r\n      <autogenerated>:1 +0xcc\r\n  net/http.(*timeoutHandler).ServeHTTP.func1()\r\n      /Users/runner/hostedtoolcache/go/1.15.15/x64/src/net/http/server.go:3272 +0xb5\r\n\r\nGoroutine 70 (running) created at:\r\n  testing.(*T).Run()\r\n      /Users/runner/hostedtoolcache/go/1.15.15/x64/src/testing/testing.go:1168 +0x5bb\r\n  github.com/labstack/echo/v4/middleware.TestTimeoutWithFullEchoStack()\r\n      /Users/runner/work/echo/echo/middleware/timeout_test.go:392 +0xa3b\r\n  testing.tRunner()\r\n      /Users/runner/hostedtoolcache/go/1.15.15/x64/src/testing/testing.go:1123 +0x202\r\n\r\nGoroutine 76 (finished) created at:\r\n  net/http.(*timeoutHandler).ServeHTTP()\r\n      /Users/runner/hostedtoolcache/go/1.15.15/x64/src/net/http/server.go:3266 +0x38f\r\n  github.com/labstack/echo/v4/middleware.TimeoutWithConfig.func1.1()\r\n      /Users/runner/work/echo/echo/middleware/timeout.go:118 +0x449\r\n  github.com/labstack/echo/v4.(*Echo).ServeHTTP()\r\n      /Users/runner/work/echo/echo/echo.go:630 +0x1fd\r\n  net/http.serverHandler.ServeHTTP()\r\n      /Users/runner/hostedtoolcache/go/1.15.15/x64/src/net/http/server.go:2843 +0xca\r\n  net/http.(*conn).serve()\r\n      /Users/runner/hostedtoolcache/go/1.15.15/x64/src/net/http/server.go:1925 +0x84c\r\n==================\r\n==================\r\nWARNING: DATA RACE\r\nRead at 0x00c000188000 by goroutine 70:\r\n  runtime.slicebytetostring()\r\n      /Users/runner/hostedtoolcache/go/1.15.15/x64/src/runtime/string.go:80 +0x0\r\n  bytes.(*Buffer).String()\r\n      /Users/runner/hostedtoolcache/go/1.15.15/x64/src/bytes/buffer.go:65 +0x437\r\n  github.com/labstack/echo/v4/middleware.TestTimeoutWithFullEchoStack.func2()\r\n      /Users/runner/work/echo/echo/middleware/timeout_test.go:408 +0x3b9\r\n  testing.tRunner()\r\n      /Users/runner/hostedtoolcache/go/1.15.15/x64/src/testing/testing.go:1123 +0x202\r\n\r\nPrevious write at 0x00c000188000 by goroutine 76:\r\n  runtime.slicecopy()\r\n      /Users/runner/hostedtoolcache/go/1.15.15/x64/src/runtime/slice.go:246 +0x0\r\n  bytes.(*Buffer).Write()\r\n      /Users/runner/hostedtoolcache/go/1.15.15/x64/src/bytes/buffer.go:174 +0x147\r\n  github.com/labstack/echo/v4/middleware.LoggerWithConfig.func2.1()\r\n      /Users/runner/work/echo/echo/middleware/logger.go:216 +0x724\r\n  github.com/labstack/echo/v4/middleware.echoHandlerFuncWrapper.ServeHTTP()\r\n      /Users/runner/work/echo/echo/middleware/timeout.go:156 +0x22c\r\n  github.com/labstack/echo/v4/middleware.(*echoHandlerFuncWrapper).ServeHTTP()\r\n      <autogenerated>:1 +0xcc\r\n  net/http.(*timeoutHandler).ServeHTTP.func1()\r\n      /Users/runner/hostedtoolcache/go/1.15.15/x64/src/net/http/server.go:3272 +0xb5\r\n\r\nGoroutine 70 (running) created at:\r\n  testing.(*T).Run()\r\n      /Users/runner/hostedtoolcache/go/1.15.15/x64/src/testing/testing.go:1168 +0x5bb\r\n  github.com/labstack/echo/v4/middleware.TestTimeoutWithFullEchoStack()\r\n      /Users/runner/work/echo/echo/middleware/timeout_test.go:392 +0xa3b\r\n  testing.tRunner()\r\n      /Users/runner/hostedtoolcache/go/1.15.15/x64/src/testing/testing.go:1123 +0x202\r\n\r\nGoroutine 76 (finished) created at:\r\n  net/http.(*timeoutHandler).ServeHTTP()\r\n      /Users/runner/hostedtoolcache/go/1.15.15/x64/src/net/http/server.go:3266 +0x38f\r\n  github.com/labstack/echo/v4/middleware.TimeoutWithConfig.func1.1()\r\n      /Users/runner/work/echo/echo/middleware/timeout.go:118 +0x449\r\n  github.com/labstack/echo/v4.(*Echo).ServeHTTP()\r\n      /Users/runner/work/echo/echo/echo.go:630 +0x1fd\r\n  net/http.serverHandler.ServeHTTP()\r\n      /Users/runner/hostedtoolcache/go/1.15.15/x64/src/net/http/server.go:2843 +0xca\r\n  net/http.(*conn).serve()\r\n      /Users/runner/hostedtoolcache/go/1.15.15/x64/src/net/http/server.go:1925 +0x84c\r\n==================\r\n--- FAIL: TestTimeoutWithFullEchoStack (0.12s)\r\n    --- FAIL: TestTimeoutWithFullEchoStack/503_-_handler_timeouts,_write_response_in_timeout_middleware (0.10s)\r\n        timeout_test.go:413: \r\n            \tError Trace:\ttimeout_test.go:413\r\n            \tError:      \tShould be false\r\n            \tTest:       \tTestTimeoutWithFullEchoStack/503_-_handler_timeouts,_write_response_in_timeout_middleware\r\nError:         testing.go:1038: race detected during execution of test\r\nError:     testing.go:1038: race detected during execution of test\r\n{\"time\":\"2022-03-13T13:06:20.137835Z\",\"level\":\"-\",\"prefix\":\"echo\",\"file\":\"recover.go\",\"line\":\"109\",\"message\":\"[PANIC RECOVER] panic!!! goroutine 494 [running]:\\ngithub.com/labstack/echo/v4/middleware.RecoverWithConfig.func1.1.1(0x19e6070, 0x1000, 0x0, 0x0, 0x1ab46c0, 0xc0001d6820)\\n\\t/Users/runner/work/echo/echo/middleware/recover.go:89 +0xa0f\\npanic(0x18f3f00, 0x1a8ff90)\\n\\t/Users/runner/hostedtoolcache/go/1.15.15/x64/src/runtime/panic.go:975 +0x47a\\nnet/http.(*timeoutHandler).ServeHTTP(0xc0000f3e40, 0x1aa7f80, 0xc0000f3dc0, 0xc0003ff900)\\n\\t/Users/runner/hostedtoolcache/go/1.15.15/x64/src/net/http/server.go:3277 +0xe8b\\ngithub.com/labstack/echo/v4/middleware.TimeoutWithConfig.func1.1(0x1ab46c0, 0xc0001d6820, 0xc00000000c, 0xc00018de00)\\n\\t/Users/runner/work/echo/echo/middleware/timeout.go:118 +0x44a\\ngithub.com/labstack/echo/v4/middleware.RecoverWithConfig.func1.1(0x1ab46c0, 0xc0001d6820, 0x0, 0x0)\\n\\t/Users/runner/work/echo/echo/middleware/recover.go:115 +0x1ab\\ngithub.com/labstack/echo/v4.(*Echo).ServeHTTP(0xc0006c2480, 0x1aa7f80, 0xc0000f3dc0, 0xc0003ff900)\\n\\t/Users/runner/work/echo/echo/echo.go:630 +0x1fe\\ngithub.com/labstack/echo/v4/middleware.TestTimeoutRecoversPanic.func2()\\n\\t/Users/runner/work/echo/echo/middleware/timeout_test.go:190 +0x65\\ngithub.com/stretchr/testify/assert.didPanic.func1(0xc00005bd08, 0xc00005bcd6, 0xc00005bcf8, 0xc000666ee0)\\n\\t/Users/runner/work/echo/pkg/mod/github.com/stretchr/testify@v1.7.0/assert/assertions.go:1018 +0x8c\\ngithub.com/stretchr/testify/assert.didPanic(0xc000666ee0, 0x1a9e5a0, 0xc0002ec300, 0x2fd9ad8, 0xc0002ec300, 0xc00005bd01)\\n\\t/Users/runner/work/echo/pkg/mod/github.com/stretchr/testify@v1.7.0/assert/assertions.go:1020 +0x6d\\ngithub.com/stretchr/testify/assert.NotPanics(0x1a9e5a0, 0xc0002ec300, 0xc000666ee0, 0x0, 0x0, 0x0, 0xc0003ff900)\\n\\t/Users/runner/work/echo/pkg/mod/github.com/stretchr/testify@v1.7.0/assert/assertions.go:1091 +0x85\\ngithub.com/labstack/echo/v4/middleware.TestTimeoutRecoversPanic(0xc0002ec300)\\n\\t/Users/runner/work/echo/echo/middleware/timeout_test.go:189 +0x57b\\ntesting.tRunner(0xc0002ec300, 0x19e68d0)\\n\\t/Users/runner/hostedtoolcache/go/1.15.15/x64/src/testing/testing.go:1123 +0x203\\ncreated by testing.(*T).Run\\n\\t/Users/runner/hostedtoolcache/go/1.15.15/x64/src/testing/testing.go:1168 +0x5bc\\n\\ngoroutine 1 [chan receive]:\\ntesting.tRunner.func1(0xc000082900)\\n\\t/Users/runner/hostedtoolcache/go/1.15.15/x64/src/testing/testing.go:1088 +0x[33](https://github.com/labstack/echo/runs/5527738819?check_suite_focus=true#step:7:33)3\\ntesting.tRunner(0xc000082900, 0xc0001b1c78)\\n\\t/Users/runner/hostedtoolcache/go/1.15.15/x64/src/testing/testing.go:1127 +0x22b\\ntesting.runTests(0xc0000b0b40, 0x1e32f60, 0x7d, 0x7d, 0xc0839988da1e9988, 0x8bb30c1c78, 0x1e378e0, 0xc0001c8190)\\n\\t/Users/runner/hostedtoolcache/go/1.15.15/x64/src/testing/testing.go:1437 +0x613\\ntesting.(*M).Run(0xc00018c180, 0x0)\\n\\t/Users/runner/hostedtoolcache/go/1.15.15/x64/src/testing/testing.go:1[34](https://github.com/labstack/echo/runs/5527738819?check_suite_focus=true#step:7:34)5 +0x3b4\\nmain.main()\\n\\t_testmain.go:407 +0x[35](https://github.com/labstack/echo/runs/5527738819?check_suite_focus=true#step:7:35)7\\n\\ngoroutine 215 [select]:\\nnet/http.(*persistConn).writeLoop(0xc00013fe60)\\n\\t/Users/runner/hostedtoolcache/go/1.15.15/x64/src/net/http/transport.go:2346 +0x1d4\\ncreated by net/http.(*Transport).dialConn\\n\\t/Users/runner/hostedtoolcache/go/1.15.15/x64/src/net/http/transport.go:1716 +0xc31\\n\\ngoroutine 214 [select]:\\nnet/http.(*persistConn).readLoop(0xc00013fe60)\\n\\t/Users/runner/hostedtoolcache/go/1.15.15/x64/src/net/http/transport.go:2167 +0xf3b\\ncreated by net/http.(*Transport).dialConn\\n\\t/Users/runner/hostedtoolcache/go/1.15.15/x64/src/net/http/transport.go:1715 +0xc0c\\n\\ngoroutine 495 [chan send]:\\ntesting.tRunner.func1(0xc0002ec480)\\n\\t/Users/runner/hostedtoolcache/go/1.15.15/x64/src/testing/testing.go:1113 +0x505\\ntesting.tRunner(0xc0002ec480, 0x19e68a8)\\n\\t/Users/runner/hostedtoolcache/go/1.15.15/x64/src/testing/testing.go:1127 +0x22b\\ncreated by testing.(*T).Run\\n\\t/Users/runner/hostedtoolcache/go/1.15.15/x64/src/testing/testing.go:1168 +0x5bc\\n\\ngoroutine 496 [select]:\\nnet/http.(*timeoutHandler).ServeHTTP(0xc0000f3d80, 0x1aa7f80, 0xc0000f3d00, 0xc0003ff700)\\n\\t/Users/runner/hostedtoolcache/go/1.15.15/x64/src/net/http/server.go:3275 +0x4bf\\ngithub.com/labstack/echo/v4/middleware.TimeoutWithConfig.func1.1(0x1ab46c0, 0xc0001d66e0, 0xc000034f60, 0x1e68[39](https://github.com/labstack/echo/runs/5527738819?check_suite_focus=true#step:7:39)0)\\n\\t/Users/runner/work/echo/echo/middleware/timeout.go:118 +0x[44](https://github.com/labstack/echo/runs/5527738819?check_suite_focus=true#step:7:44)a\\ngithub.com/labstack/echo/v4/\\n\"}\r\nFAIL\r\ncoverage: [92](https://github.com/labstack/echo/runs/5527738819?check_suite_focus=true#step:7:92).1% of statements\r\nFAIL\tgithub.com/labstack/echo/v4/middleware\t1.002s\r\n```"},{"ordinal":87,"create_number":2123,"create_updated_at":"2022-07-12T19:04:08Z","create_title":"Fix e.File() being picky with relative paths after 4.7.0 introduced echo.Fs support","create_spec":"Fix e.File() being picky with relative paths after 4.7.0 introduced echo.Fs support (Go 1.16+).\r\n\r\nSee https://github.com/labstack/echo/issues/2117#issuecomment-1063804825\r\n\r\nTLDR: `fs.Fs` and to be specific `os.DirFs.Open` does not like path to files starting with `./`, `../` and `/` but `os.Open` did not care about that and now old code has problems.\r\n\r\n```go\r\nfunc main() {\r\n\te := echo.New()\r\n\te.Use(middleware.Logger())\r\n\te.Use(middleware.Recover())\r\n\r\n\te.File(\"\", \"./index.html\")\r\n\r\n\te.Logger.Fatal(e.Start(\":8080\"))\r\n}\r\n```\r\n\r\n```bash\r\n./\r\n./main.go\r\n./index.html\r\n```","pr_number":2122,"pr":"There have been a bunch of improvements to the jwt dependency.\r\n\r\nAccording to the [golang-jwt/jwt Releases](https://github.com/golang-jwt/jwt/releases), \"any future `/v4` work is intended to be backwards-compatible with existing v3.x.y tags.\"\r\n\r\nThis appears to be true; all tests pass for recent versions.\r\n\r\n```\r\nbribera@flask:~/code/foss/echo 👻 $ git status\r\nOn branch abscondment/jwt-v4\r\nYour branch is up to date with 'abscondment/abscondment/jwt-v4'.\r\n\r\nnothing to commit, working tree clean\r\nbribera@flask:~/code/foss/echo 👻 $ git log -1\r\ncommit 130b4572b5bd24e75c8657d2699c04cf858c0bd6 (HEAD -> abscondment/jwt-v4, abscondment/abscondment/jwt-v4)\r\nAuthor: Brendan Ribera <bribera@axon.com>\r\nDate:   Wed Mar 9 11:16:37 2022 -0800\r\n\r\n    Upgrade to the v4 release of golang-jwt\r\n    \r\n    According to the [4.0.0 Version History](https://github.com/golang-jwt/jwt/blob/main/VERSION_HISTORY.md#400),\r\n    the v4 version is backwards compatible with v3.x\r\nbribera@flask:~/code/foss/echo 👻 $ go version\r\ngo version go1.17.6 linux/amd64\r\nbribera@flask:~/code/foss/echo 👻 $ go clean -testcache\r\nbribera@flask:~/code/foss/echo 👻 $ make test\r\nok  \tgithub.com/labstack/echo/v4\t0.206s\r\nok  \tgithub.com/labstack/echo/v4/middleware\t0.093s\r\nbribera@flask:~/code/foss/echo 👻 $ make race\r\nok  \tgithub.com/labstack/echo/v4\t0.484s\r\nok  \tgithub.com/labstack/echo/v4/middleware\t0.610s\r\nbribera@flask:~/code/foss/echo 👻 $ make test_version\r\nUnable to find image 'golang:1.15' locally\r\n1.15: Pulling from library/golang\r\n[... snip ...]\r\nok  \tgithub.com/labstack/echo/v4\t1.928s\r\nok  \tgithub.com/labstack/echo/v4/middleware\t0.672s\r\nbribera@flask:~/code/foss/echo 👻 $ make test_version goversion=1.14\r\nUnable to find image 'golang:1.14' locally\r\n1.14: Pulling from library/golang\r\n[...]\r\nok  \tgithub.com/labstack/echo/v4\t0.451s\r\nok  \tgithub.com/labstack/echo/v4/middleware\t0.740s\r\n```\r\n\r\nI noted that `make test_version` fail for go1.9.7 and go1.10.3 on the `master` branch, so I omitted them from my testing:\r\n\r\n```\r\nbribera@flask:~/code/foss/echo 👻 $ git status\r\nOn branch master\r\nYour branch is up to date with 'origin/master'.\r\n\r\nnothing to commit, working tree clean\r\nbribera@flask:~/code/foss/echo 👻 $ git log -1\r\ncommit 5ebed440aeec1abf7f08cca41cb02f6aaf0d7f6a (HEAD -> master, tag: v4.7.0, origin/master, origin/HEAD)\r\nAuthor: Roland Lammel <rl@neotel.at>\r\nDate:   Wed Mar 2 23:16:19 2022 +0100\r\n\r\n    Update version to v4.7.0\r\nbribera@flask:~/code/foss/echo 👻 $ make test_version goversion=1.9.7\r\nwarning: \"github.com/labstack/echo/...\" matched no packages\r\n# golang.org/x/tools/internal/typeparams\r\n/go/src/golang.org/x/tools/internal/typeparams/normalize.go:162:17: u.EmbeddedType undefined (type *types.Interface has no field or method EmbeddedType)\r\nMakefile:12: recipe for target 'init' failed\r\nmake: *** [init] Error 2\r\nmake: *** [Makefile:34: test_version] Error 2\r\nbribera@flask:~/code/foss/echo 👻 $ make test_version goversion=1.10.3\r\nwarning: \"github.com/labstack/echo/...\" matched no packages\r\n# golang.org/x/tools/internal/typeparams\r\n/go/src/golang.org/x/tools/internal/typeparams/normalize.go:162:17: u.EmbeddedType undefined (type *types.Interface has no field or method EmbeddedType)\r\nMakefile:12: recipe for target 'init' failed\r\nmake: *** [init] Error 2\r\nmake: *** [Makefile:34: test_version] Error 2\r\n```"},{"ordinal":88,"create_number":2116,"create_updated_at":"2022-03-13T13:31:40Z","create_title":"remove some unused code","create_spec":"Hi,\r\nthank you for this amazing repository , its really efficient, and i noticed some code is pretty old and have never been used\r\nso i think that maybe we may could remove them ?","pr_number":2115,"pr":"Closed Echo pull request #2115 did not include a body."},{"ordinal":89,"create_number":2109,"create_updated_at":"2024-06-17T01:45:10Z","create_title":"change request.URL in proxy","create_spec":"when I use this proxy middleware with target server that is also reverse proxy with nginx (k8s), it didn't work.\r\nso I change request.URL to target.URL.Host","pr_number":2103,"pr":"this commit will add cache-control and connection headers"},{"ordinal":90,"create_number":2102,"create_updated_at":"2022-02-23T11:28:20Z","create_title":"remove unused notFoundHandler in echo struct","create_spec":"Hi !\r\nthanks you guys for this great repo,i really like using it.\r\ni pull your repo and notice there is some go-staticcheck warning, so i did some changes that remove unused notFoundHandler in echo struct","pr_number":2101,"pr":"Hi !\r\nthanks you guys for this great repo,i really like using it.\r\ni pull your repo and notice there is some go-staticcheck warning, so i did some changes and `make(chan,0)` and `make(chan)` is exactly the same. acorrding to [go-dev.doc](https://go.dev/doc/effective_go#:~:text=The%20default%20is%20zero%2C%20for%20an%20unbuffered%20or%20synchronous%20channel.\r\n)"},{"ordinal":91,"create_number":2096,"create_updated_at":"2022-03-08T02:30:37Z","create_title":"add Routers() call in echo.test","create_spec":"Hi !\r\nthanks you guys for this great repo,i really like using it.\r\ni noticed that you call `Route()` in echo_test.go only once at `echo_test.go:62` for testing it,but did not call `Routers()` in echo.test for testing it which implement just below `Route()` in `echo.go:364`\r\ni think that we may add this call in `echo_test.go` for consistency or something like this ?","pr_number":2085,"pr":"Add Deflate middleware which is very similar to the current Gzip middleware's implementation, I think it will be nice to have deflate as well if someone also needs it.\r\n\r\nI didn't change any public function and struct in `compress.go`, and didn't touch existing test cases in `compress_test.go` as well, so I think this PR won't break backward compatibility.\r\n\r\n"},{"ordinal":92,"create_number":2082,"create_updated_at":"2022-02-04T06:26:57Z","create_title":"Reset the routers","create_spec":"Thank this featue we will able to do that our apps has routes dynamic.","pr_number":2078,"pr":"The `Retry-After` header is often sent together with `429`s and `503`s to indicate the waiting time for the client. Having a constant for this header would be nice."},{"ordinal":93,"create_number":2077,"create_updated_at":"2022-01-23T20:37:14Z","create_title":"Add pagoda (full-stack web dev kit) to the README.","create_spec":"I recently built and released [pagoda](https://github.com/mikestefanello/pagoda) which is a full-stack web development starter kit built with _Echo_ and _Ent_. I thought you may want to include it in the README. There's only a section for third-party middleware, so I included it in there. If you want to include the project but want it elsewhere, please let me know.","pr_number":2072,"pr":"I've come across the situation when I need to distinguish log levels for different recovered values. \r\n\r\nFor example, I want to handle all panics as ERRORs except situations when I've got `panic(ErrAbortHandler)`. It may happen in the `ProxyWithConfig` middleware when the client aborts the request.\r\n\r\nFor now, we have the `LogLevel` and `LogLevelSetter` both in the `RecoverConfig` to not break compatibility and introduce a new feature to the recover middleware.\r\n\r\n**Usage**:\r\n\r\n```go\r\nconfig := DefaultRecoverConfig\r\nconfig.LogLevelSetter = func(value interface{}) log.Lvl {\r\n  if err, ok := value.(error); ok {\r\n    if errors.Is(err, http.ErrAbortHandler) {\r\n      return log.WARN\r\n    }\r\n  }\r\n  return log.ERROR\r\n}\r\n```"},{"ordinal":94,"create_number":2069,"create_updated_at":"2022-01-13T08:21:47Z","create_title":"Add Souin middleware into third-party-middlewares","create_spec":"Closes #2045 ","pr_number":2067,"pr":"Fix Echo version number which was not incremented with Release 4.6.2 Now bumped to 4.6.3\r\n\r\nRelates to https://github.com/labstack/echo/issues/2066\r\nWhen I release 4.6.2 I did not use latest master branch commit but took  4 commits older commit https://github.com/labstack/echo/commit/6b5e62b27ea0bc459843e67014360dd35ae8147b as release point. Ofcourse this point version number was not bumped to `4.6.2` and is/was still `4.6.1`\r\n\r\nI have created separate branch under labstack/echo repository to hold this fix commit as different pullable branch. But to include this change into master history this commit needs to be added into master (after those 4 commits which are not included in 4.6.2 and are already in master).\r\n\r\nI will release 4.6.3 from this branch `fix_branch_4_6_2` so `4.6.3` would only have this version number fix.\r\n\r\nSorry for the mess."},{"ordinal":95,"create_number":2065,"create_updated_at":"2022-07-12T19:04:17Z","create_title":"Add list of (3rd party) middlewares to readme","create_spec":"* Add list of (3rd party) middlewares to readme. Relates to #2051\r\n* Removes gitter links","pr_number":2064,"pr":"Improve filesystem support.  Relates to #2059 I have taken some code from `v5` and made it `v4` compatible\r\n\r\nUsable if you have Go 1.16+, older Go versions will use old implementation\r\n\r\n* Add field echo.Filesystem, default filesystem that is set on `echo.New()` emulates how previously `os.Open` worked \r\n* Add methods: \r\n  * echo.FileFS(path, file string, filesystem fs.FS, m ...MiddlewareFunc), \r\n  * echo.StaticFS(pathPrefix string, fileSystem fs.FS), \r\n  * group.FileFS(path, file string, filesystem fs.FS, m ...MiddlewareFunc), \r\n  * group.StaticFS(pathPrefix string, fileSystem fs.FS). \r\n  * MustSubFS(currentFs fs.FS, fsRoot string) fs.FS\r\n* Following methods will use internally echo.Filesystem to serve files: \r\n  * echo.File, \r\n  * echo.Static, \r\n  * group.File, \r\n  * group.Static, \r\n  * Context.File\r\n\r\n\r\nNote: `embed.FS` embedds files with full paths that include that same embedded directory name as prefix. In that case you can use helper function to create sub fs with that prefix `echo.MustSubFS(embedded, rootDirectory)`.  That \"root prefix\" was not added to the method signature as there could be `fs.FS` implementations that do not act that way.\r\n\r\nExample with `embed.FS`:\r\n```go\r\n//go:embed testdata\r\nvar embedded embed.FS\r\n\r\n// assuming we have directory structure\r\n// ./  <- execution dir\r\n// ./testdata/  <- directory we are embedding\r\n// ./testdata/index.html   <- on of the embedded files\r\n\r\nfunc main() {\r\n\te := echo.New()\r\n\r\n\te.Use(middleware.Logger())\r\n\te.Use(middleware.Recover())\r\n\r\n\te.StaticFS(\"/assets\", echo.MustSubFS(embedded, \"testdata\"))\r\n\r\n\tlog.Fatal(e.Start(\":8080\"))\r\n}\r\n```"},{"ordinal":96,"create_number":2060,"create_updated_at":"2022-07-12T19:04:14Z","create_title":"JWT, KeyAuth, CSRF multivalue extractors","create_spec":"* Adds to JWT, KeyAuth, CSRF support for multivalue extractor - This is mostly useful with headers but this PR adds this other variants also. Usecase - You are using JWT middleware and expect `Authentication` header with value `Bearer xxxx` but your intracture has upstream proxy that adds  Basic authentication also. Now even if you fill basic auth in browser and your application sends requests with JWT token you would be in trouble as  previously JWT middleware knows only to extract first `Authentication` header value - which could be JWT token but could be also Basic Auth. This change allows extractor to return all those header values and run JWT token checks  or validation for Keyauth on them.\r\n* Add flag `NoErrorContinuesExecution` to JWT and KeyAuth middleware to allow continuing execution `next(c)` when error handler decides to swallow the error (returns nil). Usecase: This is useful in cases when portion of your site/api is publicly accessible and has extra features for authorized users. In that case you can use ErrorHandlerWithContext to set default public JWT token value to request and continue with handler chain.  Note: this is similar to #2048\r\n\r\np.s. there are ugly error handling parts just to preserve similar/same errors that those middlewares previously returned. Unfortunately all these 3 are quite inconsistent how they do error handling - JWT has 2 generic types. Keyauth has specific error values.","pr_number":2057,"pr":"Add a ProxyConfig field to modify the outgoing requests.\r\n\r\nTo implement  labstack/echo#1133"},{"ordinal":97,"create_number":2049,"create_updated_at":"2022-01-24T16:09:49Z","create_title":"build: upgrade `go` directive in `go.mod` to 1.17","create_spec":"This PR upgrades the `go` directive in `go.mod` file by running `go mod tidy -go=1.17` to enable [module graph pruning](https://golang.org/ref/mod#graph-pruning) and [lazy module loading](https://golang.org/ref/mod#lazy-loading).\r\n\r\n**Note 1:** This does not prevent users with earlier Go versions from successfully building packages from this module.\r\n\r\n**Note 2:** The additional `require` directive is used to record indirect dependencies for Go 1.17 or higher, see https://go.dev/ref/mod#go-mod-file-go.","pr_number":2048,"pr":"This PR adds a `CredentialsOptional` configuration option to the JWT middleware, allowing it to gracefully fail. It allows the next handler to be called, even when there is no valid JWT token present.\r\n\r\nThis was brought up before in https://github.com/labstack/echo/issues/1039, but I think it might have gotten lost in the PR that was ultimately merged. This PR reintroduces the flag and adds some tests to validate the behavior."},{"ordinal":98,"create_number":2047,"create_updated_at":"2022-07-12T19:04:21Z","create_title":"fix: route containing escaped colon is not actually matched to the request path","create_spec":"Route containing escaped colon should be matchable to request path with colon but is not actually matched (fixes #2046)","pr_number":2044,"pr":"1. The Accept-Encoding header parser is more precise. Previously it had been a simple sub-string match, not aware of the possible header grammar.\r\n\r\n2. Downstream middleware or handlers will not gzip the response twice now. This makes the middleware easier to use with packages such as Prometheus (which also implements gzipping).\r\n\r\n3. Vary header is handled more cleanly. It is not sent unless it is needed. It is now removed from 204 or other empty responses. Although this was not strictly wrong, it was unnecessary and wasteful."},{"ordinal":99,"create_number":2043,"create_updated_at":"2024-04-15T18:18:05Z","create_title":"Proxy middleware: add ModifyRequest function to modify http.Request before proxy","create_spec":"The proxy middleware currently does not allow any modification to the request, only ModifyResponse. This PR adds that option so caller can pass a custom function to be applied to requests before it hits reverse proxy.\r\n\r\nI have also added passing test. \r\n\r\nPlease consider merging this as it would allow for more flexibility when proxy with Echo. \r\n\r\nThanks!","pr_number":2040,"pr":"This PR adds the ability to choose what header to target for the X-Request-ID. Before this PR, the header used by the middleware was hardcoded for `X-Request-ID`, and exposing it via configuration will allow, among others, to set for `X-Correlation-ID`.\r\n\r\nSetting a different header key is helpful for distributed traceability since `X-Request-ID` is used to trace single requests. In contrast, `X-Correlation-ID`(or others) is [commonly used](https://en.wikipedia.org/wiki/List_of_HTTP_header_fields) to trace multiple servers' transactions.\r\n\r\nThe default is set as the previous hardcoded header (`X-Request-ID`), which is a backward-compatible change.\r\n\r\nExample for the new configuration:\r\n\r\n```Golang\r\n(...)\r\n\r\nrid = RequestIDWithConfig(RequestIDConfig{\r\n\t\tTargetHeader: echo.HeaderXCorrelationID,\r\n})\r\n\r\n(...)\r\n```\r\n\r\nThis way they can be combined and the same middleware can be used for both use-cases. \r\n\r\nAlso updated documentation: https://github.com/labstack/echox/pull/239\r\n"},{"ordinal":100,"create_number":2039,"create_updated_at":"2022-07-12T19:04:24Z","create_title":"`Allow` header support in Router, MethodNotFoundHandler (405) and CORS middleware","create_spec":"This PR adds support for `Allow` header to \r\n* http OPTIONS method responses\r\n* status 405 (method not found) responses\r\n* [CORS](https://developer.mozilla.org/en-US/docs/Glossary/CORS) middleware\r\n\r\n`Allow` header lists all method types registered for given routed url path.\r\n\r\nRelated RFCs:\r\n* `Allow` RFC https://datatracker.ietf.org/doc/html/rfc7231#section-7.4.1 all 405 should have `Allow` header listing all methods that router has registered for given path.\r\n* OPTIONS RFC https://httpwg.org/specs/rfc7231.html#OPTIONS `Allow` is optional but useful header to have\r\n\r\nImplementation notes:\r\n* Although in case of OPTIONS method router now add special options method handler instead of MethodNotFound handler as found/matched handler, we can not rely on that for CORS middleware. In CORS middleware we can not remove IF conditions for OPTIONS as when browser sends OPTIONS request it does not (by default) include cookie / authentication headers and therefore if we would blindly use `next(c)` and hope to meet our router optionshandler we probably not succeed because of different kinds of authentication middlewares (ala JWT or BasicAuth or KeyAuth) which check for stuff like that.\r\n* The reason for adding `Allow` value to context `echo.ContextKeyHeaderAllow` is because default 405 handler needs to be able to access that value and possibly  CORS middleware is interested in that value.  As `echo.MethodNotAllowedHandler` is part of public API we can not change how that method is created due backwards compatibility. \r\n* `optionsMethodHandler` is kept private as it is router specific implementation detail. If you want your own then you can add them with `e.OPTIONS()` method for paths you choose.\r\n\r\nUsing context values in Echo core is so far unprecedented and potentially controversial decision. I did not want to introduce new field into `echo.context` struct as it is quite specific case and I know that Go standard library `http.Server` is using `context.Context` to add some info to each request context - so it is not so unheard of but probably should be avoided:\r\n\r\nSee: \r\n* [http.ServerContextKey](https://github.com/golang/go/blob/3396878af43752a7c25406dabd525754f80a1c40/src/net/http/server.go#L3037)\r\n* [http.LocalAddrContextKey](https://github.com/golang/go/blob/3396878af43752a7c25406dabd525754f80a1c40/src/net/http/server.go#L1819)\r\n\r\nLet's have a discussion","pr_number":2035,"pr":"possible error should be rigorous handled  before \"FIXME\" will supported."}]$echo_pr_pairs$::jsonb)
            as pr(
                ordinal int,
                create_number int,
                create_updated_at timestamp with time zone,
                create_title text,
                create_spec text,
                pr_number int,
                pr text
            )
        order by pr.ordinal
    loop
        select pg_temp.fn_demo_change_insert(
            _project_id,
            array['feature']::text[],
            (
                select id
                from public.epic
                where project_id = _project_id
                  and name = _epic_names[((_pr.ordinal - 1) % array_length(_epic_names, 1)) + 1]
            ),
            _pr.create_title,
            _pr.create_spec
        ) into _change_id;

        call pg_temp.sp_demo_change_pr_update(
            _change_id,
            _pr.pr,
            format('https://github.com/labstack/echo/pull/%s', _pr.create_number)
        );
    end loop;
end;
$$;

do
$$
declare
    _change record;
    _epic record;
begin
    -- Demo board distribution: 40% backlog, the remaining 60% spread across active phases.
    with ranked as (
        select
            c.id,
            row_number() over (order by c.ref) as rn,
            count(*) over () as total_count
        from public.change c
        join public.project p on p.id = c.project_id
        where p.name = 'demo1'
    )
    update public.change c
    set change_phase = case
        when ranked.rn <= floor(ranked.total_count * 0.40)::int then 'backlog'
        else (array['progress', 'review', 'staging', 'production', 'rejected'])[
            ((ranked.rn - floor(ranked.total_count * 0.40)::int - 1) % 5) + 1
        ]
    end
    from ranked
    where c.id = ranked.id;

    -- Keep 30% of demo Changes standalone; distribute the rest across Echo epics.
    with ranked as (
        select
            c.id,
            row_number() over (order by c.ref) as rn,
            count(*) over () as total_count
        from public.change c
        join public.project p on p.id = c.project_id
        where p.name = 'demo1'
    ), epics as (
        select
            e.id,
            row_number() over (order by e.name) as rn,
            count(*) over () as total_count
        from public.epic e
        join public.project p on p.id = e.project_id
        where p.name = 'demo1'
    )
    update public.change c
    set epic_id = case
        when ranked.rn <= floor(ranked.total_count * 0.30)::int then null
        else epics.id
    end
    from ranked
    left join epics
        on epics.rn = ((ranked.rn - floor(ranked.total_count * 0.30)::int - 1) % epics.total_count) + 1
    where c.id = ranked.id;

    -- Keep optional artifact/default states visible in demo data.
    update public.change c
    set change_types = array[]::text[]
    from public.project p
    where p.id = c.project_id
      and p.name = 'demo1'
      and c.ref % 10 = 0;

    -- Existing demo Changes predate the scraped PR timestamp batch, so keep them recent and varied.
    update public.change c
    set modified = now() - (random() * interval '10 days')
    from public.project p
    where p.id = c.project_id
      and p.name = 'demo1'
      and c.ref <= 300;

    insert into public.test_case (change_id, scenario, done)
    select c.id, seed.scenario, seed.done
    from public.change c
    join public.project p on p.id = c.project_id
    cross join lateral (values
        ('Review the Change spec and confirm it states the user-visible behavior.', true),
        ('Exercise the primary success path for this Change.', (c.ref % 3) <> 0),
        ('Verify the documented failure or boundary path for this Change.', (c.ref % 4) = 0)
    ) as seed(scenario, done)
    where p.name = 'demo1';

    for _change in
        select c.id
        from public.change c
        join public.project p on p.id = c.project_id
        where p.name = 'demo1'
    loop
        call public.sp_change_test_case_recalculate(_change.id);
    end loop;

    for _epic in
        select e.id
        from public.epic e
        join public.project p on p.id = e.project_id
        where p.name = 'demo1'
    loop
        call public.sp_epic_test_case_recalculate(_epic.id);
    end loop;
end;
$$;

commit;
