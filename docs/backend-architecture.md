# Backend Architecture

This document defines the target architecture for the backend. Existing code
must be brought into alignment with these rules; existing deviations are not
precedents. The aim is boring, predictable code that uses the same patterns
across modules.

## Scope and organization

Paths below are relative to `backend/` unless stated otherwise.

| Location | Responsibility |
| --- | --- |
| `cmd/<executable>/` | Application startup, dependency construction, wiring, and shutdown. |
| `config/<environment>.yaml` | Environment-specific application configuration. |
| `internal/<module>/` | A business capability, organized into `api.go`, `service.go`, and `repo.go`. |
| `internal/domain/` | Shared business models, requests, responses, and other data types passed between layers. |
| `internal/error/` | All backend error definitions and public error-handling functions. |
| `internal/health/` | Health checks; exempt from the business-module layer structure. |
| `pkg/<connector>/` | Configuration loaders and focused integrations with external libraries or infrastructure. |

The API/service/repository structure applies to business modules. `domain`,
`error`, and `health` have their own responsibilities and do not need that
structure. Layers are files within a module, not a requirement to create three
separate Go packages.

The backend architecture and its implementation are the scope of this work.
Frontend and CLI changes are outside that scope.

## Database authority

The database is the source of truth for persisted structure and integrity.
Align Go models and repository queries with its tables, views, columns,
relationships, and constraints.

Every API operation and business field must be traceable to database data or
database-backed behavior. Derived representations of stored data, such as
rendered document content, still have a database source. Remove obsolete fields
and operations that have no such source, even when this breaks the existing API
contract. Do not retain compatibility placeholders or invent database structures
to preserve obsolete API behavior.

Remove tests that exist only to enforce removed behavior. Update tests for
retained behavior to match the database contract; do not discard those tests
merely because the implementation changes.

Reuse existing database business logic through its functions and procedures
wherever it supports the operation. Do not duplicate that logic in Go.
Services own user-facing validation and business logic that the database does
not already provide. They coordinate multiple operations only when the workflow
cannot be expressed through separate API calls.

### Transactions

Remove all explicit Go-managed database transactions as part of this refactor,
including transaction wrappers and transaction-dependent helpers. Use existing
database functions or procedures for atomic workflows where available.

Individual SQL statements and calls to database functions retain PostgreSQL's
normal transactional behavior. Removing Go-managed transactions does not mean
disabling database atomicity.

Do not introduce or restore a Go-managed transaction without explicit user
approval. Any restoration will be decided during review. Multiple independent
database calls must not be described or treated as one atomic operation.

### Project configuration

Project configuration belongs to the `project` module and is exposed through a
project endpoint. Remove the `options` module and its endpoints.

Resolve configuration using the project's configured slug in `public.config`.
If that configuration does not exist, return an error through `internal/error`.
Do not fall back to the `default` configuration.

Use the database configuration for document types, change phases, colors, and
change types. Keep this project data distinct from application configuration
loaded from `config/<environment>.yaml`.

## Explicit API operations

Do not create convenience APIs that combine operations or return additional
data merely to save the caller another request. Keep each endpoint's purpose,
side effects, and result explicit.

- A create operation returns HTTP `201 Created` with a domain ID DTO containing
  the created ID, not the full created entity.
- A successful update or delete returns HTTP `204 No Content`, with no response
  body. Its service and repository methods return only an error, which is `nil`
  on success; they do not load or return an entity for the API to discard.
- Callers use a separate details endpoint when they need current entity data.
- Do not reload an entity after a mutation merely to enrich the response.
- Do not return related entities, refreshed lists, or rendered documents as
  side results of a mutation. Expose those reads through separate endpoints.
- Decompose workflows into separate API operations wherever possible. An API
  handler must not conceal the same composition by calling multiple services
  to assemble a convenience response.

Service-level composition is a last resort, allowed only when the required
behavior has no viable decomposition into separate API calls. An existing
database function may implement one business operation involving several
database steps; reuse it without adding convenience reads to its API response.

## Business-module layers

The normal call direction is API → service → repository. Dependencies are
provided explicitly through constructors. A service may also receive another
module's service or repository when needed. Keep these dependencies minimal
and visible in the constructor; avoid circular dependencies and hidden lookups.

### API: `api.go`

The API layer owns HTTP handling and code that requires Echo or
`github.com/gookit/validate/v2`:

- Register routes and bind requests into domain types.
- Validate request format using `validate`.
- Pass the request context and domain request to the service.
- Write HTTP responses using the centralized error package when handling errors.

Move business logic and any work that does not require the API layer into the
service. The API must not access the repository or database directly.

### Service: `service.go`

The service layer owns business logic and unavoidable orchestration:

- Enforce business validity independently of the API. A direct service caller
  must not bypass business validation.
- Apply business rules that are not already implemented by the database.
- Coordinate repository calls and injected collaborators only when the
  operation cannot be decomposed into separate API calls.
- Perform necessary mapping between domain types.

Services must not import Echo, `validate`, or `pgx`. They must not depend on HTTP
contexts, HTTP responses, database connections, or database transactions.

### Repository: `repo.go`

The repository layer owns code that requires `pgx`:

- Execute SQL and call existing database functions or procedures.
- Bind query parameters and scan database results into domain types.
- Handle database errors through the public functions in `internal/error`.

Move business decisions, domain-to-domain mapping, and unavoidable orchestration
into the service. Keep repository methods direct and consistent across modules.
Prefer straightforward, repeated database code over unnecessary helper
functions, generic query machinery, or an ORM-like abstraction.

`List`, `Details`, `Create`, `Update`, and `Remove` are recommended method names,
not mandatory names or a required CRUD interface. Preserve specialized
operations, including the separate change update methods. Do not add unused
operations merely for symmetry.

## Domain types

All business inputs and outputs crossing API, service, or repository boundaries
must use types defined in `internal/domain`. Standard infrastructure types such
as `context.Context` and `error` remain standard types; dependency constructors
accept the collaborators they need.

Pass the same domain request through API → service → repository wherever
possible. For example, use `domain.ChangeIDRequest` throughout instead of
unpacking it into an `int` argument at the service/repository boundary.

When different representations are necessary, define both types in
`internal/domain` and perform the mapping in the service. Binding an HTTP
request and scanning database columns are responsibilities of their respective
boundary layers, not reasons to introduce duplicate transport or storage DTOs.

Private data types are allowed only when strictly local to one layer, such as
an implementation detail inside `api.go` or `repo.go`. They must not cross layer
boundaries. Keep error definitions in `internal/error`, not in `domain` or
individual business modules.

## Errors

`internal/error` owns all backend error definitions and public functions for
creating, wrapping, and interpreting errors. This applies backend-wide,
including startup code, connectors, health checks, and business modules.

- Use the package's public functions instead of defining errors, creating
  ad hoc wrappers, or constructing error responses independently elsewhere.
- Handle errors from external libraries through these functions, preserving
  underlying errors for inspection.
- Once an error has been handled by the package, higher layers may return it
  unchanged. Call another error-package function only when additional context
  or interpretation is needed.
- Keep error classification and interpretation consistent across modules.
  HTTP output remains an API responsibility and uses the error package's
  interpretation.

## Dependency injection

Construct dependencies at application startup and inject them through
constructors. Each component receives only what it uses. For example, an API
receives its service, a service receives its repository and other collaborators,
and a repository receives its database pool.

Use concrete dependencies by default. Introduce an interface only when it must
be mocked to support testing, such as a service's repository dependency. Do not
create interfaces for every layer merely to follow a pattern.

Configuration loaders and connectors in `pkg` are injectable collaborators.
Do not pass configuration to components that do not need it. Avoid hidden
dependency lookup and unnecessary dependency-injection abstractions.

## Libraries

Use these libraries for their respective responsibilities wherever applicable.
Do not add new dependencies without user approval.

| Library | Responsibility |
| --- | --- |
| `github.com/gookit/validate/v2` | API request validation. |
| `github.com/gofrs/uuid/v5` | UUID handling. |
| `github.com/gookit/config/v2` | Application configuration. |
| `github.com/jackc/pgx/v5` | PostgreSQL access. |
| `github.com/labstack/echo/v5` | HTTP routing and handling. |
| `github.com/microcosm-cc/bluemonday` | HTML sanitization. |
| `github.com/rs/zerolog` | Logging. |
| `github.com/stretchr/testify` | Test assertions and test support. |
| `github.com/yuin/goldmark` | Markdown parsing and rendering. |

## Verification and maintenance

Follow the repository's unit-test and coverage requirements: cover every
acceptance-criterion bullet, test production code, and maintain unit-test
coverage of at least 95%. API integration coverage must be at least 90%.
Additional tests must increase coverage or prove a specific
acceptance criterion.

Run the applicable backend lint, vet, and race checks through `make check`.
Verify database-dependent behavior against the database contract as well as
through unit tests.

Keep code readable, maintainable, and consistent. Add abstractions only to
solve an existing need. When a proposed design requires an exception to these
rules, raise the concrete decision with the user.
