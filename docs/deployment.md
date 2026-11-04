# Deployment

## Container image

The image is built from the [`Containerfile`](../Containerfile) with
[podman](https://podman.io):

- **Build stage**: the official Go Alpine image. It always runs on the build
  host's native platform and cross-compiles a static binary (`CGO_ENABLED=0`)
  for the target platform via `GOOS`/`GOARCH`. Multi-arch builds therefore
  need no QEMU emulation and are fast.
- **Final stage**: `gcr.io/distroless/static-debian13:nonroot`. It contains
  the binary, CA certificates and timezone data, and nothing else: no shell
  and no package manager. The image is about 10 MB.
- **Non-root**: it runs as UID/GID `65532`. The UID is numeric so that
  Kubernetes can enforce `runAsNonRoot`.
- **Pinned bases**: both base images are pinned by digest for reproducible
  builds. Dependabot opens PRs when new digests are published.

```sh
make image                 # local platform, tagged ghcr.io/cecobask/fibonacci:<git describe>
make image-run             # build and run on :8080
make image-multiarch       # linux/amd64 + linux/arm64 manifest list
make image-push            # push the manifest list (requires podman login ghcr.io)
```

Override the defaults with `IMAGE=…`, `VERSION=…` and `PLATFORMS=…`.

## Releasing

Releases are cut from `main` by pushing a semver tag:

```sh
git tag -a v0.2.0 -m "v0.2.0"
git push origin v0.2.0
```

The [release workflow](../.github/workflows/release.yml) then:

1. builds the linux/amd64 and linux/arm64 images with podman;
2. pushes the manifest list as `ghcr.io/cecobask/fibonacci:0.2.0` and
   `:sha-<commit>`;
3. publishes a signed SLSA build provenance attestation for the image digest.
   Verify it with `gh attestation verify oci://ghcr.io/cecobask/fibonacci:0.2.0 --owner cecobask`.

Releasing does not deploy anything. Deployment is a separate, reviewed step,
described below.

## Kubernetes manifests

```
deploy/kubernetes/
├── base/                       environment-agnostic resources
│   ├── deployment.yaml         probes, resources, hardened securityContext
│   ├── service.yaml            ClusterIP on port 80 → http (8080)
│   ├── serviceaccount.yaml     no API token mounted
│   └── poddisruptionbudget.yaml
└── overlays/
    ├── staging/                namespace fibonacci-staging, 1 replica, debug logs
    └── production/             namespace fibonacci, HPA (3–10 replicas)
```

Render an overlay locally with `make k8s-render` or
`kubectl kustomize deploy/kubernetes/overlays/production`.

Notes on the base:

- **Pod Security.** Pods meet the
  [restricted Pod Security Standard](https://kubernetes.io/docs/concepts/security/pod-security-standards/#restricted):
  non-root, read-only root filesystem, all capabilities dropped,
  `RuntimeDefault` seccomp. Each overlay's namespace enforces `restricted`.
- **Resources.** The memory limit equals what the service may use, and
  `GOMEMLIMIT` is derived from it. There is no CPU limit, which avoids CFS
  throttling; the CPU request drives scheduling and the HPA.
- **Rollouts.** The Deployment uses `maxUnavailable: 0`, readiness gating and
  graceful shutdown, so rollouts never drop requests.
- **Spreading.** Topology spread constraints spread replicas across nodes.
- **Production replicas.** The production overlay removes `spec.replicas`
  because the HPA owns it. The HPA needs
  [metrics-server](https://github.com/kubernetes-sigs/metrics-server) in the
  cluster.
- **Exposure.** Ingress/Gateway resources are intentionally not included,
  because they depend on the cluster. Add an `HTTPRoute` or `Ingress` that
  points at the `fibonacci` Service on port 80.

## Continuous delivery with Flux

[Flux](https://fluxcd.io) applies the overlays straight from this repository.
The resources in [`deploy/flux/`](../deploy/flux) belong in your cluster's
fleet repository, the one `flux bootstrap` manages:

- `gitrepository.yaml`: watches `main` of this repo and fetches only
  `deploy/kubernetes/`
- `staging.yaml`: a Flux `Kustomization` that applies the staging overlay
- `production.yaml`: applies the production overlay, but only after staging
  is healthy (`dependsOn`)

Both Kustomizations use `prune: true`, so removing a manifest from git deletes
it from the cluster. They also use `wait: true`, so a reconciliation only
succeeds when every workload is ready.

### Promoting a release

Image tags are pinned in each overlay's `kustomization.yaml` (`images[].newTag`)
and changed only through pull requests. Every deployment is therefore
reviewed, auditable and revertible with `git revert`.

1. Tag a release as described above and wait for the image to be published.
2. Open a PR that bumps `newTag` in `deploy/kubernetes/overlays/staging`.
   Merge it, and Flux rolls out staging within its sync interval. To sync
   immediately, run
   `flux reconcile kustomization fibonacci-staging --with-source`.
3. Verify staging, then open a PR that bumps `newTag` in the production
   overlay.

To roll back, revert the bump commit.

Useful commands:

```sh
flux get kustomizations
flux get sources git
kubectl -n fibonacci rollout status deployment/fibonacci
kubectl -n fibonacci logs -l app.kubernetes.io/name=fibonacci --tail=50
```
