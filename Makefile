BINARY      := gforce
IMAGE       := ghcr.io/gforce/gforce
TAG         ?= dev
GOPATH_BIN  := $(shell go env GOPATH)/bin

.PHONY: build test lint docker-build generate migrate dev help

## build: Compile all Go binaries.
build:
	go build ./...

## test: Run the full test suite with race detection.
test:
	go test ./... -race -count=1 -timeout=120s

## lint: Run golangci-lint.
lint:
	$(GOPATH_BIN)/golangci-lint run ./...

## docker-build: Build the container image.
docker-build:
	docker build -t $(IMAGE):$(TAG) .

## generate: Regenerate CRD manifests and DeepCopy methods via controller-gen.
generate:
	$(GOPATH_BIN)/controller-gen \
		object:headerFile="hack/boilerplate.go.txt" \
		rbac:roleName=gforce-operator \
		crd \
		paths="./operator/..." \
		output:crd:artifacts:config=charts/gforce/templates/crds

## migrate: Apply database migrations.
migrate:
	@echo "Applying migrations from internal/store/migrations/"
	@for f in internal/store/migrations/*.sql; do \
		echo "  Applying $$f ..."; \
		psql "$$GFORCE_DB_DSN" -f "$$f"; \
	done

## dev: Run the server locally with live reload via air.
dev:
	$(GOPATH_BIN)/air -c .air.toml

## help: Show this help message.
help:
	@grep -E '^##' $(MAKEFILE_LIST) | sed 's/## //'
