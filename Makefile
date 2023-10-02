SHELL ?= /bin/bash

ARGO_CD_CHART_VERSION := 5.46.6
BUF_LINT_ERROR_FORMAT ?= text
GO_LINT_ERROR_FORMAT ?= colored-line-number
CERT_MANAGER_CHART_VERSION := 1.11.5

VERSION_PACKAGE := github.com/akuity/kargo/internal/version


CONTAINER_RUNTIME := $(shell ./check_container.sh)


IMAGE_REPO ?= kargo
IMAGE_TAG ?= dev
IMAGE_PUSH ?= false
IMAGE_PLATFORMS =
DOCKER_BUILD_OPTS =

# Intelligently choose to build a multi-arch image if the intent is to push to a
# container registry (IMAGE_PUSH=true). If not pushing, build an single-arch
# image for the local architecture. Honor IMAGE_PLATFORMS above all.
ifeq ($(strip $(IMAGE_PUSH)),true)
  override DOCKER_BUILD_OPTS += --push
  ifeq ($(strip $(IMAGE_PLATFORMS)),)
    override DOCKER_BUILD_OPTS += --platform=linux/amd64,linux/arm64
  endif
endif
ifneq ($(strip $(IMAGE_PLATFORMS)),)
  override DOCKER_BUILD_OPTS += --platform=$(IMAGE_PLATFORMS)
endif

# These enable cross-compiling the CLI binary for any desired OS and CPU
# architecture. Even if building inside a container, they will default to the
# developer's native OS and CPU architecture.
#
# Note: We use `uname` instead of `go env` because if a developer intends to
# build inside a container, it's possible they may not have Go installed on the
# host machine.
#
# This only works on Linux and macOS. Windows users are advised to undertake
# Kargo development activities inside WSL2.
GOOS ?= $(shell uname -s | tr '[:upper:]' '[:lower:]')
GOARCH ?= $(shell uname -m)

################################################################################
# Tests                                                                        #
#                                                                              #
# These targets are used by our continuous integration processes. Use these    #
# directly at your own risk -- they assume required tools (and correct         #
# versions thereof) to be present on your system.                              #
#                                                                              #
# If you prefer to execute these tasks in a container that is pre-loaded with  #
# required tools, refer to the hacking section toward the bottom of this file. #
################################################################################

.PHONY: lint
lint: lint-go lint-proto lint-charts

.PHONY: lint-go
lint-go:
	golangci-lint run --out-format=$(GO_LINT_ERROR_FORMAT)

.PHONY: lint-proto
lint-proto:
	buf lint api --error-format=$(BUF_LINT_ERROR_FORMAT)

.PHONY: lint-charts
lint-charts:
	cd charts/kargo && \
	helm dep up && \
	helm lint .

.PHONY: test-unit
test-unit:
	go test \
		-v \
		-timeout=300s \
		-race \
		-coverprofile=coverage.txt \
		-covermode=atomic \
		./...

################################################################################
# Builds                                                                       #
#                                                                              #
# These targets are used by our continuous integration and release processes.  #
# Use these directly at your own risk -- they assume required tools            #
# correct versions thereof) to be present on your system.                      #
#                                                                              #
# If you prefer to execute these tasks in a container that is pre-loaded with  #
# required tools, refer to the hacking section toward the bottom of this file. #
################################################################################

.PHONY: build-cli
build-cli:
	CGO_ENABLED=0 go build \
		-ldflags "-w -X $(VERSION_PACKAGE).version=$(VERSION) -X $(VERSION_PACKAGE).buildDate=$$(date -u +'%Y-%m-%dT%H:%M:%SZ') -X $(VERSION_PACKAGE).gitCommit=$(GIT_COMMIT) -X $(VERSION_PACKAGE).gitTreeState=$(GIT_TREE_STATE)" \
		-o bin/kargo-$(GOOS)-$(GOARCH)$(shell [ ${GOOS} = windows ] && echo .exe) \
		./cmd/cli

################################################################################
# Code generation: To be run after modifications to API types                  #
################################################################################

.PHONY: codegen
codegen:
	buf generate api
	controller-gen \
		rbac:roleName=manager-role \
		crd \
		webhook \
		paths=./... \
		output:crd:artifacts:config=charts/kargo/crds
	controller-gen \
		object:headerFile=hack/boilerplate.go.txt \
		paths=./...
	pnpm --dir=ui install --dev
	pnpm --dir=ui run generate:schema
	npm install -g @bitnami/readme-generator-for-helm
	bash hack/helm-docs/helm-docs.sh

################################################################################
# Hack: Targets to help you hack                                               #
#                                                                              #
# These targets minimize required developer setup by executing in a container  #
# that is pre-loaded with required tools.                                      #
################################################################################

DOCKER_CMD := ${CONTAINER_RUNTIME} run \
	-it \
	--rm \
	-v gomodcache:/go/pkg/mod \
	-v $(dir $(realpath $(firstword $(MAKEFILE_LIST)))):/workspaces/kargo \
	-w /workspaces/kargo \
	kargo:dev-tools

.PHONY: hack-build-dev-tools
hack-build-dev-tools:
	${CONTAINER_RUNTIME} build -f Dockerfile.dev -t kargo:dev-tools .

.PHONY: hack-lint
hack-lint: hack-build-dev-tools
	$(DOCKER_CMD) make lint

.PHONY: hack-lint-go
hack-lint-go: hack-build-dev-tools
	$(DOCKER_CMD) make lint-go

.PHONY: hack-lint-proto
hack-lint-proto: hack-build-dev-tools
	$(DOCKER_CMD) make lint-proto

.PHONY: hack-lint-charts
hack-lint-charts: hack-build-dev-tools
	$(DOCKER_CMD) make lint-charts

.PHONY: hack-test-unit
hack-test-unit: hack-build-dev-tools
	$(DOCKER_CMD) make test-unit

.PHONY: hack-codegen
hack-codegen: hack-build-dev-tools
	$(DOCKER_CMD) make codegen

# Build an image. Example usages:
#
# Build image for local architecture (kargo:dev)
#   make hack-build
#
# Push a multi-arch image to a personal repository (myusername/kargo:latest)
#   make hack-build IMAGE_REPO=myusername/kargo IMAGE_PUSH=true IMAGE_TAG=latest
#
# Build a linux/amd64 image with a ${CONTAINER_RUNTIME} build option to not re-use ${CONTAINER_RUNTIME} build cache
# 	make hack-build IMAGE_PLATFORMS=linux/amd64 DOCKER_BUILD_OPTS=--no-cache
.PHONY: hack-build
hack-build:
	${CONTAINER_RUNTIME} buildx build \
		$(DOCKER_BUILD_OPTS) \
		--build-arg GIT_COMMIT=$(shell git rev-parse HEAD) \
		--build-arg GIT_TREE_STATE=$(shell if [ -z "`git status --porcelain`" ]; then echo "clean" ; else echo "dirty"; fi) \
		--tag $(IMAGE_REPO):$(IMAGE_TAG) \
		.

.PHONY: hack-build-cli
hack-build-cli: hack-build-dev-tools
	@# Local values of GOOS and GOARCH get passed into the container.
	$(DOCKER_CMD) sh -c 'GOOS=$(GOOS) GOARCH=$(GOARCH) make build-cli'

.PHONY: hack-kind-up
hack-kind-up:
	ctlptl apply -f hack/kind/cluster.yaml
	make hack-install-cert-manager
	make hack-install-argocd

.PHONY: hack-k3d-up
hack-k3d-up:
	ctlptl apply -f hack/k3d/cluster.yaml
	make hack-install-cert-manager
	make hack-install-argocd

.PHONY: hack-kind-down
hack-kind-down:
	ctlptl delete -f hack/kind/cluster.yaml

.PHONY: hack-k3d-down
hack-k3d-down:
	ctlptl delete -f hack/k3d/cluster.yaml

.PHONY: hack-install-cert-manager
hack-install-cert-manager:
	helm upgrade cert-manager cert-manager \
		--repo https://charts.jetstack.io \
		--version $(CERT_MANAGER_CHART_VERSION) \
		--install \
		--create-namespace \
		--namespace cert-manager \
		--set installCRDs=true \
		--wait

.PHONY: hack-install-argocd
hack-install-argocd:
	helm upgrade argocd argo-cd \
		--repo https://argoproj.github.io/argo-helm \
		--version $(ARGO_CD_CHART_VERSION) \
		--install \
		--create-namespace \
		--namespace argocd \
		--set 'configs.secret.argocdServerAdminPassword=$$2a$$10$$5vm8wXaSdbuff0m9l21JdevzXBzJFPCi8sy6OOnpZMAG.fOXL7jvO' \
		--set 'configs.params."application\.namespaces"=*' \
		--set server.service.type=NodePort \
		--set server.service.nodePortHttp=30080 \
		--wait

.PHONY: hack-add-rollouts
hack-add-rollouts:
	helm upgrade argocd argo-cd \
		--repo https://argoproj.github.io/argo-helm \
		--version $(ARGO_CD_CHART_VERSION) \
		--namespace argocd \
		--reuse-values \
		--set server.extensions.enabled=true \
		--set server.extensions.contents[0].name=argo-rollouts \
		--set server.extensions.contents[0].url=https://github.com/argoproj-labs/rollout-extension/releases/download/v0.2.0/extension.tar \
		--wait
	helm upgrade argo-rollouts argo-rollouts \
		--repo https://argoproj.github.io/argo-helm \
		--version 2.20.0 \
		--install \
		--create-namespace \
		--namespace argo-rollouts \
		--wait
