# Development

## Prerequisites

| Tool          | Version | Purpose                                       |
|---------------|---------|-----------------------------------------------|
| Go            | 1.27+   | build, test, code generation                  |
| golangci-lint | v2.15+  | linting and formatting                        |
| podman        | 5+      | building and running container images         |
| kubectl       | 1.30+   | rendering Kustomize overlays (`make k8s-render`) |

You don't need to install oapi-codegen. It is pinned in `go.mod` as a
[tool dependency](https://go.dev/doc/modules/managing-dependencies#tools) and
runs via `go tool oapi-codegen`, so everyone uses exactly the same version.

## Everyday workflow

```sh
make run        # run locally (text logs, debug level)
make test       # all tests with -race
make lint       # golangci-lint
make fmt        # gofmt + goimports
make check      # what CI does: generate, tidy, lint, test, then fail on a dirty tree
```

Run `make check` before you push. If it passes locally, CI will pass too.

Try the API:

```sh
curl -i 'localhost:8080/v1/fibonacci?n=10'
curl -i 'localhost:8080/v1/fibonacci?n=-1'
```

## Changing the API

The spec in `api/openapi.yaml` is the source of truth. Never edit
`internal/api/api.gen.go` by hand.

As an example, here is how you would add `GET /v1/fibonacci/{index}`, which
returns a single number:

1. **Update the spec.** Add the path, its parameters and its responses to
   `api/openapi.yaml`. Reuse the existing `Problem` responses for errors.
2. **Regenerate.** Run `make generate`. `api.StrictServerInterface` gains a
   new method, and the build breaks until you implement it. That is intended:
   the compiler tells you exactly what is missing.
3. **Implement the domain logic** in `internal/fibonacci`, with unit tests.
4. **Implement the handler** in `internal/server/handler.go`. Return the
   generated response types, such as `api.GetFibonacciIndex200JSONResponse`,
   and map domain errors to the 400 response type.
5. **Add integration tests** in `internal/server/server_test.go`.
6. **Commit the spec and the generated code together.** CI regenerates the
   code and fails if the committed version differs.

Some rules for evolving the API:

- **Additive changes** are safe within `/v1`: new endpoints, new optional
  parameters, new response fields.
- **Breaking changes** need a new version prefix (`/v2/...`), served next to
  `/v1` until clients have migrated. Breaking changes include renaming or
  removing fields, changing types, and tightening validation.
- **Bump `info.version`** in the spec whenever it changes.

## Testing

| Kind        | Location                          | What it covers                                     |
|-------------|-----------------------------------|----------------------------------------------------|
| Unit        | `internal/fibonacci`              | sequence values, bounds, int64 boundary            |
| Fuzz        | `FuzzSequence`                    | sequence invariants for arbitrary `n` (`make fuzz`) |
| Unit        | `internal/config`                 | env parsing and validation                         |
| Unit        | `internal/server/middleware_test` | panic recovery, request IDs, access log            |
| Integration | `internal/server/server_test`     | full handler chain via `httptest.Server`           |
| Integration | `cmd/fibonacci/main_test`         | real listener, real requests, graceful shutdown    |
| Benchmark   | `BenchmarkSequence`               | `make bench`                                       |

Conventions:

- Tests are table-driven and call `t.Parallel()`.
- Tests use only the standard library; there is no assertion library.
- Requests use `t.Context()`, so they are cancelled when the test ends.

## Linting

The configuration is in `.golangci.yml` (v2 format). Besides the standard
linters it enables, among others:

- `gosec` for security issues
- `errorlint` for correct error wrapping
- `sloglint`, which enforces consistent `slog` usage: snake_case keys, the
  `*Context` variants, and no global logger
- `bodyclose` and `noctx` for HTTP client hygiene

Generated code is excluded. If you must silence a finding, use a
`//nolint:<linter> // reason` comment. `nolintlint` rejects directives that
lack a specific linter or an explanation.

## Logging conventions

- Use `log/slog` with typed attributes (`slog.String`, `slog.Int`, …), never
  loose key/value pairs.
- Write keys in `snake_case`.
- Always pass a context (`logger.InfoContext(ctx, ...)`).
- Log messages are static strings. Variable data goes into attributes.
- Include `request_id` on any log line written while handling a request.

## Commits and pull requests

- Keep commits small and focused, with an imperative subject line ("Add X",
  "Fix Y").
- CI must be green before you merge. Dependabot opens weekly grouped PRs for
  Go modules, GitHub Actions and base images.
