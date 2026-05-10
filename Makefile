BINARY        := gforce
IMAGE         := ghcr.io/gforce/gforce
TAG           ?= dev
GOPATH_BIN    := $(shell go env GOPATH)/bin
TOOLS_BIN     := $(HOME)/go-packages/bin
LOCALBIN      := $(shell pwd)/bin
ENVTEST_K8S   := 1.29.x
CONTROLLER_GEN := $(TOOLS_BIN)/controller-gen
AIR            := $(TOOLS_BIN)/air
GOLANGCI_LINT  := $(TOOLS_BIN)/golangci-lint
ENVTEST        := go run sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

$(LOCALBIN):
	mkdir -p $(LOCALBIN)

.PHONY: build test test-integration lint docker-build generate manifests migrate dev envtest-assets help

## build: Compile all Go binaries.
build:
	go build ./...

## test: Run unit tests with race detection.
test:
	go test ./... -race -count=1 -timeout=120s \
		--ignore=./operator/controllers/...

## test-integration: Run operator controller tests (requires envtest binaries).
test-integration: envtest-assets
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S) --bin-dir $(LOCALBIN) -p path)" \
		go test ./operator/controllers/... -v -race -timeout=5m

## lint: Run golangci-lint.
lint:
	$(GOLANGCI_LINT) run ./...

## docker-build: Build the container image.
docker-build:
	docker build -t $(IMAGE):$(TAG) .

## generate: Regenerate DeepCopy methods from kubebuilder markers.
generate:
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./operator/api/..."

## manifests: Generate CRD and RBAC manifests.
manifests:
	$(CONTROLLER_GEN) \
		crd \
		rbac:roleName=gforce-operator-role \
		paths="./operator/..." \
		output:crd:artifacts:config=config/crd/bases

## migrate: Apply database migrations against GFORCE_DB_DSN.
migrate:
	@echo "Applying migrations from internal/store/migrations/"
	@for f in internal/store/migrations/*.sql; do \
		echo "  Applying $$f ..."; \
		psql "$$GFORCE_DB_DSN" -f "$$f"; \
	done

## envtest-assets: Download envtest binaries for controller integration tests.
envtest-assets: $(LOCALBIN)
	$(ENVTEST) use $(ENVTEST_K8S) --bin-dir $(LOCALBIN) -p path

## dev: Run the server locally with live reload via air.
dev:
	$(AIR) -c .air.toml

## help: Show this help message.
help:
	@grep -E '^##' $(MAKEFILE_LIST) | sed 's/## //'
