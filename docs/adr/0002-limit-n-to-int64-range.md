# 0002. Limit n to the int64 range

**Status:** accepted

## Context

Fibonacci numbers grow exponentially. F(92) = 7540113804746346429 is the
largest that fits in an int64; F(93) overflows. Without a bound, a single
request could also ask for millions of arbitrary-precision numbers and
exhaust memory and CPU.

## Decision

`n` must be between 0 and 93 inclusive, so the largest value returned is
F(92). Values are `int64`, serialised as plain JSON integers to match the
documented `[0, 1, 1, 2, 3]` format. Requests outside the range get
`400 Bad Request`.

## Consequences

- The cost of every request is bounded and tiny, so no rate limiting or
  pagination is needed for resource protection.
- JavaScript clients using `JSON.parse` lose precision above 2^53 (after
  F(78)). Clients that need exact large values must use a big-integer-aware
  JSON parser.
- Supporting larger values (e.g. `math/big`, string-encoded numbers) would
  change the response format, so it would be a new API version (`/v2`).
