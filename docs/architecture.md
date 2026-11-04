# Architecture

This service is deliberately small, but it is structured like a service that
will grow: a contract-first API, a transport-agnostic domain package and a
thin HTTP layer in between.

## Repository layout

```
api/openapi.yaml        API contract: the single source of truth
cmd/fibonacci/          main package: config, logger, signal handling, lifecycle
internal/api/           code generated from the spec (never edit api.gen.go)
internal/config/        environment variable parsing and validation
internal/fibonacci/     domain logic: no HTTP, no logging, no I/O
internal/server/        HTTP handlers, middleware, probes and http.Server setup
deploy/kubernetes/      Kustomize base and per-environment overlays
deploy/flux/            Flux resources that deploy the overlays
docs/                   this documentation and the ADRs
```

Everything except `cmd/` lives under `internal/`, so no other module can
import it and the packages can be refactored freely.

## Request flow

```
client
  │
  ▼
http.Server            timeouts, header size limit, error log routed to slog
  │
  ▼
requestID              reads or generates X-Request-ID and stores it in the context
accessLog              one structured log line per request
recoverer              turns a panic into a 500 problem response
securityHeaders        X-Content-Type-Options: nosniff
  │
  ▼
http.ServeMux          Go 1.22+ method and path patterns
  ├── GET /healthz, GET /readyz          internal/server/health.go
  └── GET /v1/fibonacci                  registered by generated code
        │
        ▼
      api.ServerInterfaceWrapper         parses and type-checks query parameters
        │  (on a parse error → handleRequestError → 400 problem)
        ▼
      api strict handler                 turns typed responses into HTTP
        │
        ▼
      server.handler.GetFibonacci        maps domain errors to API responses
        │
        ▼
      fibonacci.Sequence                 pure function
```

## Key design choices

**Contract first.** The OpenAPI document is written first, and the Go server
interface, request and response types are generated from it with
[oapi-codegen](https://github.com/oapi-codegen/oapi-codegen). The compiler
then catches any handler that drifts from the contract, and the spec can be
handed to API consumers or used to generate clients.
See [ADR 0001](adr/0001-openapi-first-with-oapi-codegen.md).

**Strict server mode.** Handlers implement `api.StrictServerInterface`. They
receive typed request objects and return typed response objects, for example
`api.GetFibonacci200JSONResponse`. They never touch `http.ResponseWriter`, so
it is impossible to return a status code or content type that the spec does
not declare.

**Domain isolation.** `internal/fibonacci` knows nothing about HTTP. It
returns sentinel errors (`ErrNegative`, `ErrTooLarge`), and the HTTP layer
decides how to represent them. A future gRPC or CLI entry point could reuse it
unchanged.

**Errors follow RFC 9457.** Every error response is
`application/problem+json`. Client errors carry a `detail` message. Server
errors deliberately do not; the details go to the logs together with the
request ID instead.

**Standard library only.** The only runtime dependency is
`github.com/oapi-codegen/runtime`, which the generated parameter-binding code
needs. Routing, logging (`log/slog`), configuration and testing all use the
standard library. See [ADR 0003](adr/0003-standard-library-first.md).

**Testable `main`.** `main` only calls `run(ctx, getenv, stdout)`. Because the
environment and output are injected, the integration test in
`cmd/fibonacci/main_test.go` starts the real server, sends requests and checks
that graceful shutdown completes, all without touching global state.

## Known limitations and future work

- **Metrics and tracing.** Prometheus metrics and OpenTelemetry tracing would
  need third-party modules. The request ID and structured access logs cover
  basic observability for now.
- **Responses for unknown routes.** 404 and 405 responses for unknown routes
  or methods come from `http.ServeMux` as plain text, not problem documents.
- **Large Fibonacci numbers.** Values beyond F(92) would need arbitrary
  precision and probably a new API version. See
  [ADR 0002](adr/0002-limit-n-to-int64-range.md).
