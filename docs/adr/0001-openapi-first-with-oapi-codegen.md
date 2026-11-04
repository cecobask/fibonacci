# 0001. OpenAPI first, with oapi-codegen

**Status:** accepted

## Context

The API is expected to grow over several years and to be consumed by clients
we do not control. Hand-written handlers and hand-written docs tend to drift
apart, and nothing catches it.

## Decision

`api/openapi.yaml` is the source of truth. Server interfaces and models are
generated with [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen)
using the `std-http-server` and `strict-server` generators:

- `std-http-server` registers routes on the standard `http.ServeMux`, so no
  router framework is needed.
- `strict-server` makes handlers return typed responses, so they cannot emit
  undeclared status codes or content types.
- oapi-codegen is pinned as a `tool` dependency in `go.mod`. The generated
  code is committed, and CI fails if it is stale.
- `embedded-spec` is disabled because it pulls in `kin-openapi` at runtime.

## Consequences

- Changing the API always starts with the spec, and the compiler points at
  every handler that has to change.
- The generated code depends on `github.com/oapi-codegen/runtime` for
  parameter binding. That is the service's only runtime dependency.
- Bounds such as `minimum` and `maximum` in the spec are not enforced at
  runtime, because we skip the validation middleware to avoid its
  dependencies. Handlers validate their input in the domain layer, and tests
  keep the two in sync.
