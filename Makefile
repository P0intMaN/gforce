BINARY        := gforce
IMAGE         := gforce
TAG           ?= latest
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

.PHONY: build test test-integration lint docker-build kind-load helm-install helm-uninstall \
        kind-setup kind-teardown port-forward helm-lint helm-template helm-deps \
        generate manifests migrate dev envtest-assets help

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

## kind-load: Load the image into the kind cluster.
kind-load:
	kind load docker-image $(IMAGE):$(TAG) --name kind

## helm-deps: Update Helm chart dependencies.
helm-deps:
	helm dependency update ./charts/gforce

## helm-lint: Lint the Helm chart.
helm-lint: helm-deps
	helm lint ./charts/gforce

## helm-template: Render Helm templates to stdout.
helm-template: helm-deps
	helm template gforce ./charts/gforce --namespace gforce-system | less

## helm-install: Install GForce via Helm.
helm-install: helm-deps
	helm upgrade --install gforce ./charts/gforce \
		--namespace gforce-system \
		--create-namespace \
		--wait \
		--timeout 5m \
		--set server.jwtSecret=$(shell openssl rand -hex 32)

## helm-uninstall: Uninstall GForce from the cluster.
helm-uninstall:
	helm uninstall gforce -n gforce-system

## kind-setup: Build image, load into kind, and install via Helm.
kind-setup:
	./scripts/kind-setup.sh

## kind-teardown: Remove GForce from kind cluster.
kind-teardown:
	./scripts/kind-teardown.sh

## port-forward: Forward HTTP and SSH ports from the cluster.
port-forward:
	kubectl port-forward -n gforce-system svc/gforce 8080:80 &
	kubectl port-forward -n gforce-system svc/gforce 2222:2222 &

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
	cp config/crd/bases/*.yaml charts/gforce/crds/

## migrate: Apply database migrations against GFORCE_DB_DSN.
migrate:
	go run ./cmd/gforce migrate

## envtest-assets: Download envtest binaries for controller integration tests.
envtest-assets: $(LOCALBIN)
	$(ENVTEST) use $(ENVTEST_K8S) --bin-dir $(LOCALBIN) -p path

## dev: Run the server locally with live reload via air.
dev:
	$(AIR) -c .air.toml

## help: Show this help message.
help:
	@grep -E '^##' $(MAKEFILE_LIST) | sed 's/## //'
