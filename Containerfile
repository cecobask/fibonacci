# The build stage always runs on the native platform of the build host and
# cross-compiles for the target platform, so multi-arch builds need no emulation.
FROM --platform=$BUILDPLATFORM docker.io/library/golang:1.27.3-alpine@sha256:fa3d596898d9f126af7c01d14ee7597bab3f39cccf392ef36b69fe77cd681f18 AS build

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev

WORKDIR /src

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=bind,source=go.mod,target=go.mod \
    --mount=type=bind,source=go.sum,target=go.sum \
    go mod download

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=bind,target=. \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/fibonacci ./cmd/fibonacci

FROM gcr.io/distroless/static-debian13:nonroot@sha256:25ac8f0c3b2334a9eb05452266bd4697864a8c8ef0ef71e5e6480d7f92868329

COPY --from=build /out/fibonacci /fibonacci

# Numeric UID/GID of the distroless "nonroot" user, so that Kubernetes can
# verify runAsNonRoot without resolving user names.
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/fibonacci"]
