# 0003. Standard library first

**Status:** accepted

## Context

Every dependency must be tracked, updated and audited for its whole lifetime.
Over a five-year horizon, abandoned or breaking dependencies are a real
maintenance cost. Since Go 1.21 and 1.22, the standard library covers
structured logging (`log/slog`) and method/path routing (`net/http`).

## Decision

Use the standard library unless a dependency is clearly required, or building
our own would make no sense. Adding a third-party module needs explicit
agreement from the maintainers.

Approved dependencies:

| Module | Why |
|--------|-----|
| `github.com/oapi-codegen/oapi-codegen/v2` | code generation tool, not linked into the binary ([0001](0001-openapi-first-with-oapi-codegen.md)) |
| `github.com/oapi-codegen/runtime` | required by the generated parameter-binding code |

## Consequences

- The binary has one runtime dependency, and Dependabot PRs are few and easy
  to review.
- Some features take more of our own code, such as request IDs and access
  logging middleware. Each of these is small and covered by tests.
- Metrics (Prometheus) and tracing (OpenTelemetry) are not implemented yet.
  Adopting them will mean revisiting this decision.
