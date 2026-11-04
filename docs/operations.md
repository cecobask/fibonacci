# Operations

## Configuration

The service follows [twelve-factor](https://12factor.net/config) principles:
all configuration comes from environment variables, and it is validated at
startup. If any value is invalid, the process exits with code 1 and a message
listing every problem.

| Variable           | Default | Description                                          |
|--------------------|---------|------------------------------------------------------|
| `PORT`             | `8080`  | TCP port to listen on                                |
| `LOG_LEVEL`        | `info`  | `debug`, `info`, `warn` or `error`                   |
| `LOG_FORMAT`       | `json`  | `json` for machines, `text` for humans               |
| `SHUTDOWN_DELAY`   | `0s`    | Time to keep serving after a signal, before draining  |
| `SHUTDOWN_TIMEOUT` | `15s`   | Maximum time to wait for in-flight requests to finish |

The Kubernetes manifests also set `GOMEMLIMIT` from the container's memory
limit, so the Go garbage collector works harder before the pod gets
OOM-killed.

## Logs

Logs are structured, one JSON object per line on stdout. Every line has a
`service` attribute. Request logs look like this:

```json
{"time":"2026-11-04T18:21:07.848Z","level":"INFO","msg":"request","service":"fibonacci",
 "request_id":"19a0a084ace5326b9a88e1c6da037c83","method":"GET","path":"/v1/fibonacci",
 "query":"n=5","status":200,"bytes":12,"duration":195042,"remote_addr":"10.0.0.12:51234",
 "user_agent":"curl/8.7.1"}
```

- `request_id` is taken from the `X-Request-ID` request header if present and
  valid, and generated otherwise. It is always echoed back in the response, so
  a user's report can be matched to log lines.
- `duration` is in nanoseconds.
- Probe requests (`/healthz`, `/readyz`) are logged at `debug` level to keep
  production logs readable.
- Server errors (5xx) and panics are logged at `error` level together with the
  request ID. Panics include a stack trace.

## Health probes

| Endpoint   | Meaning                                  | Fails when                                   |
|------------|------------------------------------------|----------------------------------------------|
| `/healthz` | Liveness: the process can serve HTTP      | the process is wedged (the kubelet restarts it) |
| `/readyz`  | Readiness: send traffic to this instance | during startup and once shutdown has begun   |

Neither probe checks downstream dependencies, because there are none. If you
add a dependency later, add it to readiness only. A dependency outage should
take an instance out of rotation, not restart it.

## Graceful shutdown

On `SIGTERM` or `SIGINT` the service:

1. logs `shutdown started` and starts failing `/readyz` with 503;
2. keeps serving normally for `SHUTDOWN_DELAY`, so Kubernetes has time to
   remove the pod from Service endpoints and load balancers stop sending it
   new connections;
3. calls `http.Server.Shutdown`, which stops accepting connections and waits
   up to `SHUTDOWN_TIMEOUT` for in-flight requests to finish;
4. logs `shutdown complete` and exits 0. If the timeout expires first, it
   exits 1.

A second signal during shutdown kills the process immediately.

In Kubernetes, `terminationGracePeriodSeconds` (30s) must be larger than
`SHUTDOWN_DELAY + SHUTDOWN_TIMEOUT` (5s + 15s). Otherwise the kubelet sends
`SIGKILL` while requests are still draining.

## HTTP server limits

| Setting             | Value  | Protects against                       |
|---------------------|--------|----------------------------------------|
| `ReadHeaderTimeout` | 5s     | Slowloris-style slow header attacks    |
| `ReadTimeout`       | 10s    | slow request bodies                    |
| `WriteTimeout`      | 10s    | stuck clients holding goroutines       |
| `IdleTimeout`       | 120s   | idle keep-alive connections piling up  |
| `MaxHeaderBytes`    | 64 KiB | oversized headers                      |

These values are defined in `internal/server/server.go`.

## Troubleshooting

| Symptom                                    | Likely cause / action                                                   |
|--------------------------------------------|-------------------------------------------------------------------------|
| Pod in `CrashLoopBackOff` right after deploy | Invalid env var. Run `kubectl logs --previous`; the error names the variable. |
| 503 from `/readyz` while the pod is running  | The pod is shutting down. Look for `shutdown started` in the logs.        |
| Requests fail during rollouts              | Check that `SHUTDOWN_DELAY` > 0 and that the readiness probe is configured. |
| `shutting down: context deadline exceeded` | Requests outlived `SHUTDOWN_TIMEOUT`. Investigate the slow requests, or raise both the timeout and the grace period. |
| Need more detail                           | Set `LOG_LEVEL=debug`. This also logs probe requests and rejected input. |
