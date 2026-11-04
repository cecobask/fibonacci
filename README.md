# fibonacci

[![CI](https://github.com/cecobask/fibonacci/actions/workflows/ci.yml/badge.svg)](https://github.com/cecobask/fibonacci/actions/workflows/ci.yml)

A small HTTP API that returns the first `n` Fibonacci numbers, starting from 0.

```console
$ curl -s 'localhost:8080/v1/fibonacci?n=5'
[0,1,1,2,3]
```

It is written in Go using only the standard library plus the code generated from
an [OpenAPI spec](api/openapi.yaml). It ships as a ~10 MB multi-arch container
image and runs on Kubernetes via Kustomize and Flux.

## Quick start

Requirements: Go 1.27+, [golangci-lint](https://golangci-lint.run) v2, and
optionally [podman](https://podman.io) and kubectl.

```sh
make run           # start the server on :8080 with human-readable logs
make test          # run unit and integration tests
make lint          # run golangci-lint
make help          # list all targets
```

Or run it in a container:

```sh
make image-run
```

## API

| Method | Path                | Description                              |
|--------|---------------------|------------------------------------------|
| GET    | `/v1/fibonacci?n=N` | First `N` Fibonacci numbers, `0 ≤ N ≤ 93` |
| GET    | `/healthz`          | Liveness probe                           |
| GET    | `/readyz`           | Readiness probe                          |

`n` is capped at 93 because the 93rd number, F(92) = 7540113804746346429, is
the largest Fibonacci number that fits in a signed 64-bit integer. See
[ADR 0002](docs/adr/0002-limit-n-to-int64-range.md).

Invalid input returns `400 Bad Request` with an
[RFC 9457](https://www.rfc-editor.org/rfc/rfc9457) problem document:

```console
$ curl -s 'localhost:8080/v1/fibonacci?n=-1'
{"detail":"n must not be negative","status":400,"title":"Bad Request","type":"about:blank"}
```

The full contract lives in [`api/openapi.yaml`](api/openapi.yaml).

## Configuration

All configuration is read from environment variables:

| Variable           | Default | Description                                         |
|--------------------|---------|-----------------------------------------------------|
| `PORT`             | `8080`  | TCP port to listen on                               |
| `LOG_LEVEL`        | `info`  | `debug`, `info`, `warn` or `error`                  |
| `LOG_FORMAT`       | `json`  | `json` or `text`                                    |
| `SHUTDOWN_DELAY`   | `0s`    | Time to keep serving after a signal, before draining |
| `SHUTDOWN_TIMEOUT` | `15s`   | Maximum time to wait for in-flight requests         |

## Documentation

New to the project? Read these in order:

1. [Architecture](docs/architecture.md): how the code is organised and why
2. [Development](docs/development.md): day-to-day workflow, including how to change the API
3. [Operations](docs/operations.md): configuration, logs, probes and graceful shutdown
4. [Deployment](docs/deployment.md): container images, releases, Kustomize and Flux
5. [Architecture decision records](docs/adr/): the reasoning behind key choices
