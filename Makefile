SHELL := /bin/bash
VERSION ?= dev
REGISTRY := ghcr.io/bartlettc22
DOCKER_IMAGE := $(REGISTRY)/image-inquisitor:$(VERSION)
GO_VERSION ?= $(shell grep '^go ' go.mod | awk '{print $$2}')
GOLANGCI_VERSION := v2.3.1
GO_FILES_NO_VENDOR := $(shell find ./* -name "*.go" -not -path "./vendor/*")
TRIVY_VERSION ?= 0.65.0
DOCKER_RUN_FLAGS := --rm \
  -u $$(id -u $${USER}):$$(id -g $${USER}) \
  -v $$PWD:/src -w /src \
  -v $$HOME/.cache/go-build:/.cache/go-build \
  -v $$HOME/go:/go \
  -e GOCACHE=/.cache/go-build \
  -e GOPATH-/go \
  -v $$HOME/.cache/golangci-lint:/.cache/golangci-lint

.PHONY: build-image
build-image:
	DOCKER_BUILDKIT=1 docker build \
	--build-arg VERSION=$(VERSION) \
	--build-arg GO_VERSION=$(GO_VERSION) \
	--build-arg TRIVY_VERSION=$(TRIVY_VERSION) \
	-t $(DOCKER_IMAGE) \
	.

publish-image: build-image
	docker push $(DOCKER_IMAGE)

.PHONY: dev-run
dev-run: 
	go run ./... \
	--log-level=debug \
	--log-json=true \
	--image-source file \
	--report-outputs=summary,summaryImageCombined,summaryRegistry,imageSummary,imageRegistry,imageVulnerabilities,imageKubernetes \
	--include-kubernetes-namespaces=prometheus \
	--image-source-file-path=$$(pwd)/test/images.txt

.PHONY: dev-run-docker
dev-run-docker: build-image
	mkdir -p $$HOME/.cache/imginq/trivy
	mkdir -p ./.reports/dev-run
	docker run \
	-u $$(id -u $${USER}):$$(id -g $${USER}) \
	-v $$HOME/.cache/imginq/trivy:/trivy/ \
	-v $$PWD/.reports/dev-run/:/reports/ \
	-v $$PWD/test/images2.txt:/images2.txt \
	$(DOCKER_IMAGE) \
	run \
	--log-level=debug \
	--source-id=testsrc \
	--source=file \
	--source-file-path=/images2.txt \
	--report-location=file:///reports \
	--reports InventoryReport,SummaryReport,ImageSummaryReport

# List files whose formatting differs from gofmts
.PHONY: fmt-check
fmt-check: 
	test -z $$(docker run $(DOCKER_RUN_FLAGS) golang:$(GO_VERSION) gofmt -l $(GO_FILES_NO_VENDOR))

.PHONY: mkdirs
mkdirs:
	mkdir -p $$HOME/go
	mkdir -p $$HOME/.cache/go-build
	mkdir -p $$HOME/.cache/golangci-lint

## Run golangci-lint checker
.PHONY: lint-check
lint-check: mkdirs 
	docker run $(DOCKER_RUN_FLAGS) golangci/golangci-lint:$(GOLANGCI_VERSION) golangci-lint run --timeout=4m

# Tidy and vendor dependencies
.PHONY: tidy
tidy: mkdirs
	docker run $(DOCKER_RUN_FLAGS) golang:$(GO_VERSION) sh -c "go mod tidy && go mod vendor"

.PHONY: clean
clean:
	rm -rf ./test/trivy-results

# Run unit tests in a docker container
.PHONY: test
test: mkdirs
	docker run $(DOCKER_RUN_FLAGS) golang:$(GO_VERSION) go test -v -covermode=atomic -cover ./...

# Run integration tests in a docker container
.PHONY: integration-test
integration-test: mkdirs
	docker run $(DOCKER_RUN_FLAGS) golang:$(GO_VERSION) go test -v -covermode=atomic -cover -tags=integration ./...