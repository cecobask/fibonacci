IMAGE         ?= ghcr.io/cecobask/fibonacci
VERSION       ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
PLATFORMS     ?= linux/amd64,linux/arm64
GOLANGCI_LINT ?= golangci-lint
BIN           := bin/fibonacci
LDFLAGS       := -s -w -X main.version=$(VERSION)

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z0-9_-]+:.*##/ {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

##@ Development

.PHONY: generate
generate: ## Generate code from api/openapi.yaml
	go generate ./...

.PHONY: build
build: ## Build the binary into bin/
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/fibonacci

.PHONY: run
run: ## Run the service locally with human-readable logs
	LOG_FORMAT=text LOG_LEVEL=debug go run ./cmd/fibonacci

.PHONY: test
test: ## Run all tests with the race detector
	go test -race -count=1 ./...

.PHONY: cover
cover: ## Run tests and open the coverage report
	go test -race -count=1 -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

.PHONY: fuzz
fuzz: ## Fuzz the fibonacci package for 30s
	go test -run='^$$' -fuzz=FuzzSequence -fuzztime=30s ./internal/fibonacci

.PHONY: bench
bench: ## Run benchmarks
	go test -run='^$$' -bench=. -benchmem ./...

.PHONY: lint
lint: ## Run golangci-lint
	$(GOLANGCI_LINT) run ./...

.PHONY: fmt
fmt: ## Format code
	$(GOLANGCI_LINT) fmt ./...

.PHONY: tidy
tidy: ## Tidy go.mod and go.sum
	go mod tidy

.PHONY: check
check: generate tidy lint test ## Run everything CI runs and fail on uncommitted changes
	git diff --exit-code

##@ Containers

.PHONY: image
image: ## Build the container image for the local platform
	podman build --build-arg VERSION=$(VERSION) -t $(IMAGE):$(VERSION) .

.PHONY: image-multiarch
image-multiarch: ## Build a multi-arch manifest list for $(PLATFORMS)
	-podman manifest rm $(IMAGE):$(VERSION) 2>/dev/null
	podman build --build-arg VERSION=$(VERSION) --platform $(PLATFORMS) --manifest $(IMAGE):$(VERSION) .

.PHONY: image-push
image-push: ## Push the multi-arch manifest list to the registry
	podman manifest push --all $(IMAGE):$(VERSION)

.PHONY: image-run
image-run: image ## Build and run the container image locally
	podman run --rm -p 8080:8080 $(IMAGE):$(VERSION)

##@ Kubernetes

.PHONY: k8s-render
k8s-render: ## Render the Kustomize overlays
	kubectl kustomize deploy/kubernetes/overlays/staging
	kubectl kustomize deploy/kubernetes/overlays/production

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf bin coverage.out
