SHELL := /bin/bash
VERSION ?= $(shell cat ./VERSION)
REGISTRY := ghcr.io/bartlettc22
DOCKER_IMAGE := $(REGISTRY)/image-inquisitor:$(VERSION)
GO_VERSION ?= 1.22.3
GOLANGCI_VERSION := golangci/golangci-lint:v1.60.3
GO_FILES_NO_VENDOR := $(shell find ./* -name "*.go" -not -path "./vendor/*")
TRIVY_VERSION ?= 0.54.1
DOCKER_RUN_FLAGS := --rm -u $$(id -u $${USER}):$$(id -g $${USER})

.PHONY: build-docker
build-docker:
	DOCKER_BUILDKIT=1 docker build \
	--build-arg VERSION=$(VERSION) \
	--build-arg GO_VERSION=$(GO_VERSION) \
	--build-arg TRIVY_VERSION=$(TRIVY_VERSION) \
	-t $(DOCKER_IMAGE) \
	.

.PHONY: dev-run
dev-run:
	go run ./... \
	--log-level=debug \
	--log-json=false \
	--run-trivy=true \
	--run-registry=true \
	--image-source kubernetes \
	--include-kubernetes-namespaces=prometheus \
	--image-source-file-path=/test/images.txt

.PHONY: fmt-check
fmt-check: ## List files whose formatting differs from gofmt's.
	test -z $(shell gofmt -l $(GO_FILES_NO_VENDOR))

.PHONY: lint-check
lint-check: ## Run golangci-lint linter.
	mkdir -p $$HOME/.cache/go-build; \
	mkdir -p -v $$HOME/.cache/golangci-lint; \
	docker run $(DOCKER_RUN_FLAGS) \
		-v $$PWD:/src \
		-w /src \
		-v $$HOME/.cache/go-build:/.cache/go-build \
		-v $$HOME/.cache/golangci-lint:/.cache/golangci-lint \
		-e GOLANGCI_LINT_CACHE=/.cache/golangci-lint \
		-e GOCACHE=/.cache/go-build \
		$(GOLANGCI_VERSION) golangci-lint run --timeout=4m

.PHONY: vendor
vendor: ## Tidy and vendor dependencies.
	go mod tidy
	go mod vendor

.PHONY: clean
clean:
	rm -rf ./test/trivy-results

.PHONY: test
test: ## Run unit and integration tests.
	go test -v -covermode=atomic -cover ./...
