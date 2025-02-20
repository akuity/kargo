include $(CURDIR)/hack/tools.mk

SHELL	      ?= /bin/bash
EXTENDED_PATH ?= $(CURDIR)/hack/bin:$(PATH)

ARGO_CD_CHART_VERSION		:= 7.7.0
ARGO_ROLLOUTS_CHART_VERSION := 2.37.7
CERT_MANAGER_CHART_VERSION 	:= 1.16.1

BUF_LINT_ERROR_FORMAT	?= text
GO_LINT_ERROR_FORMAT 	?= colored-line-number

VERSION_PACKAGE := github.com/akuity/kargo/internal/version

# Default to docker, but support alternative container runtimes that are CLI-compatible with Docker
CONTAINER_RUNTIME ?= docker

IMAGE_REPO 			?= kargo
LOCAL_REG_PORT			?= 5001
BASE_IMAGE 			?= localhost:$(LOCAL_REG_PORT)/$(IMAGE_REPO)-base
IMAGE_TAG 			?= dev
IMAGE_PUSH 			?= false
IMAGE_PLATFORMS 	?=
DOCKER_BUILD_OPTS 	=

DOCS_PORT ?= 3000

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
ifeq ($(GOARCH), x86_64)
	override GOARCH = amd64
endif

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
lint: lint-go lint-proto lint-charts lint-ui

.PHONY: format
format: format-go format-ui

.PHONY: lint-go
lint-go: install-golangci-lint
	$(GOLANGCI_LINT) run --out-format=$(GO_LINT_ERROR_FORMAT)

.PHONY: format-go
format-go:
	$(GOLANGCI_LINT) run --fix

.PHONY: lint-proto
lint-proto: install-buf
	# Vendor go dependencies to build protobuf definitions
	go mod vendor
	$(BUF) lint api --error-format=$(BUF_LINT_ERROR_FORMAT)

.PHONY: lint-charts
lint-charts: install-helm
	cd charts/kargo && \
	$(HELM) dep up && \
	$(HELM) lint .

.PHONY: lint-ui
lint-ui:
	pnpm --dir=ui install --dev
	pnpm --dir=ui run lint

.PHONY: typecheck-ui
typecheck-ui:
	pnpm --dir=ui install
	pnpm --dir=ui run typecheck

.PHONY: format-ui
format-ui:
	pnpm --dir=ui install --dev
	pnpm --dir=ui run lint:fix

.PHONY: test-unit
test-unit: install-helm
	PATH=$(EXTENDED_PATH) go test \
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

################################################################################
# Build: Targets to help build                                                 #
################################################################################

.PHONY: clean
clean:
	rm -rf build

.PHONY: build-base-image
build-base-image:
	mkdir -p build
	cp kargo-base.apko.yaml build
	docker run \
		--rm \
		-v $(dir $(realpath $(firstword $(MAKEFILE_LIST))))build:/build \
		-w /build \
		cgr.dev/chainguard/apko \
		build kargo-base.apko.yaml $(BASE_IMAGE) kargo-base.tar.gz
	docker image load -i build/kargo-base.tar.gz

.PHONY: build-cli
build-cli:
	CGO_ENABLED=0 go build \
		-ldflags "-w -X $(VERSION_PACKAGE).version=$(VERSION) -X $(VERSION_PACKAGE).buildDate=$$(date -u +'%Y-%m-%dT%H:%M:%SZ') -X $(VERSION_PACKAGE).gitCommit=$(GIT_COMMIT) -X $(VERSION_PACKAGE).gitTreeState=$(GIT_TREE_STATE)" \
		-o bin/kargo-$(GOOS)-$(GOARCH)$(shell [ ${GOOS} = windows ] && echo .exe) \
		./cmd/cli

################################################################################
# Used for Nighty/Unstable builds                                              #
################################################################################

.PHONY: build-nightly-cli
build-nightly-cli:
	CGO_ENABLED=0 go build \
		-ldflags "-w -X $(VERSION_PACKAGE).version=$(VERSION) -X $(VERSION_PACKAGE).buildDate=$$(date -u +'%Y-%m-%dT%H:%M:%SZ') -X $(VERSION_PACKAGE).gitCommit=$(GIT_COMMIT) -X $(VERSION_PACKAGE).gitTreeState=$(GIT_TREE_STATE)" \
		-o bin/kargo-cli/${VERSION}/${GOOS}/${GOARCH}/kargo$(shell [ ${GOOS} = windows ] && echo .exe) ./cmd/cli


################################################################################
# Used to build the CLI with the UI embedded                                   #
################################################################################

.PHONY: build-ui
build-ui:
	cd ui && NODE_ENV=production BUILD_TARGET_PATH=../internal/api/ui pnpm run build --emptyOutDir
	touch internal/api/ui/.keep

.PHONY: build-cli-with-ui
build-cli-with-ui: build-ui build-cli

################################################################################
# Code generation: To be run after modifications to API types                  #
################################################################################

.PHONY: codegen
codegen: codegen-proto codegen-controller codegen-directive-configs codegen-ui codegen-docs

.PHONY: codegen-proto
codegen-proto: install-protoc install-go-to-protobuf install-protoc-gen-gogo install-goimports install-buf
	./hack/codegen/proto.sh

.PHONY: codegen-controller
codegen-controller: install-controller-gen
	$(CONTROLLER_GEN) \
		rbac:roleName=manager-role \
		crd \
		webhook \
		paths=./api/v1alpha1/... \
		output:crd:artifacts:config=charts/kargo/resources/crds
	$(CONTROLLER_GEN) \
		object:headerFile=hack/boilerplate.go.txt \
		paths=./...

.PHONY: codegen-directive-configs
codegen-directive-configs:
	npm install -g quicktype
	./hack/codegen/directive-configs.sh

.PHONY: codegen-ui
codegen-ui:
	pnpm --dir=ui install --dev
	pnpm --dir=ui run generate:schema

.PHONY: codegen-docs
codegen-docs:
	npm install -g @bitnami/readme-generator-for-helm
	bash hack/helm-docs/helm-docs.sh

################################################################################
# Hack: Targets to help you hack                                               #
#                                                                              #
# These targets minimize required developer setup by executing in a container  #
# that is pre-loaded with required tools.                                      #
################################################################################

# Prevents issues with vcs stamping within docker containers. 
GOFLAGS="-buildvcs=false"

DOCKER_OPTS := -it \
	--rm \
	-e GOFLAGS=$(GOFLAGS) \
	-v gomodcache:/home/user/gocache \
	-v $(dir $(realpath $(firstword $(MAKEFILE_LIST)))):/workspaces/kargo \
	-v /workspaces/kargo/ui/node_modules \
	-w /workspaces/kargo

DOCKER_CMD := $(CONTAINER_RUNTIME) run $(DOCKER_OPTS) kargo:dev-tools

DEV_TOOLS_BUILD_OPTS =

# Intelligently choose to load the image into the local Docker daemon
# depending on whether or not Docker Buildx is available.
BUILDX_AVAILABLE ?= $(shell $(CONTAINER_RUNTIME) buildx inspect >/dev/null 2>&1 && echo true || echo false)
ifeq ($(BUILDX_AVAILABLE),true)
	override DEV_TOOLS_BUILD_OPTS += --load
endif

ifeq ($(GOOS),linux)
	override DEV_TOOLS_BUILD_OPTS += --build-arg USER_ID=$(shell id -u)
	override DEV_TOOLS_BUILD_OPTS += --build-arg GROUP_ID=$(shell id -g)
endif

.PHONY: hack-build-dev-tools
hack-build-dev-tools:
	$(CONTAINER_RUNTIME) build $(DEV_TOOLS_BUILD_OPTS) \
 		-f Dockerfile.dev -t kargo:dev-tools .

.PHONY: hack-lint
hack-lint: hack-build-dev-tools
	$(DOCKER_CMD) make lint

.PHONY: hack-format
hack-format: hack-build-dev-tools
	$(DOCKER_CMD) make format

.PHONY: hack-lint-go
hack-lint-go: hack-build-dev-tools
	$(DOCKER_CMD) make lint-go

.PHONY: hack-lint-proto
hack-lint-proto: hack-build-dev-tools
	$(DOCKER_CMD) make lint-proto

.PHONY: hack-lint-charts
hack-lint-charts: hack-build-dev-tools
	$(DOCKER_CMD) make lint-charts

.PHONY: hack-lint-ui
hack-lint-ui: hack-build-dev-tools
	$(DOCKER_CMD) make lint-ui

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
# Build a linux/amd64 image with a docker build option to not re-use docker build cache
# 	make hack-build IMAGE_PLATFORMS=linux/amd64 DOCKER_BUILD_OPTS=--no-cache
.PHONY: hack-build
hack-build: build-base-image
	{ \
		$(CONTAINER_RUNTIME) run -d -p $(LOCAL_REG_PORT):5000 --name tmp-registry registry:2; \
		trap '$(CONTAINER_RUNTIME) rm -f tmp-registry' EXIT; \
		docker push $(BASE_IMAGE):latest-amd64; \
		docker push $(BASE_IMAGE):latest-arm64; \
		$(CONTAINER_RUNTIME) buildx build \
			$(DOCKER_BUILD_OPTS) \
			--network host \
			--build-arg BASE_IMAGE=$(BASE_IMAGE) \
			--build-arg GIT_COMMIT=$(shell git rev-parse HEAD) \
			--build-arg GIT_TREE_STATE=$(shell if [ -z "`git status --porcelain`" ]; then echo "clean" ; else echo "dirty"; fi) \
			--tag $(IMAGE_REPO):$(IMAGE_TAG) \
			.;\
	}

.PHONY: hack-build-cli
hack-build-cli: hack-build-dev-tools
	@# Local values of GOOS and GOARCH get passed into the container.
	$(DOCKER_CMD) sh -c 'GOOS=$(GOOS) GOARCH=$(GOARCH) make build-cli'

.PHONY: hack-kind-up
hack-kind-up:
	ctlptl apply -f hack/kind/cluster.yaml
	make hack-install-prereqs

.PHONY: hack-k3d-up
hack-k3d-up:
	ctlptl apply -f hack/k3d/cluster.yaml
	make hack-install-prereqs

.PHONY: hack-kind-down
hack-kind-down:
	ctlptl delete -f hack/kind/cluster.yaml

.PHONY: hack-k3d-down
hack-k3d-down:
	ctlptl delete -f hack/k3d/cluster.yaml

.PHONY: hack-install-prereqs
hack-install-prereqs: hack-install-cert-manager hack-install-argocd hack-install-argo-rollouts

.PHONY: hack-install-cert-manager
hack-install-cert-manager: install-helm
	$(HELM) upgrade cert-manager cert-manager \
		--repo https://charts.jetstack.io \
		--version $(CERT_MANAGER_CHART_VERSION) \
		--install \
		--create-namespace \
		--namespace cert-manager \
		--set crds.enabled=true \
		--wait

.PHONY: hack-install-argocd
hack-install-argocd: install-helm
	$(HELM) upgrade argocd argo-cd \
		--repo https://argoproj.github.io/argo-helm \
		--version $(ARGO_CD_CHART_VERSION) \
		--install \
		--create-namespace \
		--namespace argocd \
		--set 'configs.secret.argocdServerAdminPassword=$$2a$$10$$5vm8wXaSdbuff0m9l21JdevzXBzJFPCi8sy6OOnpZMAG.fOXL7jvO' \
		--set 'configs.params."application\.namespaces"=*' \
		--set server.service.type=NodePort \
		--set server.service.nodePortHttp=30080 \
		--set server.extensions.enabled=true \
		--set server.extensions.contents[0].name=argo-rollouts \
		--set server.extensions.contents[0].url=https://github.com/argoproj-labs/rollout-extension/releases/download/v0.3.3/extension.tar \
		--wait

.PHONY: hack-install-argo-rollouts
hack-install-argo-rollouts: install-helm
	$(HELM) upgrade argo-rollouts argo-rollouts \
		--repo https://argoproj.github.io/argo-helm \
		--version $(ARGO_ROLLOUTS_CHART_VERSION) \
		--install \
		--create-namespace \
		--namespace argo-rollouts \
		--wait

.PHONY: hack-uninstall-prereqs
hack-uninstall-prereqs: hack-uninstall-argo-rollouts hack-uninstall-argocd hack-uninstall-cert-manager

.PHONY: hack-uninstall-argo-rollouts
hack-uninstall-argo-rollouts: install-helm
	$(HELM) delete argo-rollouts --namespace argo-rollouts

.PHONY: hack-uninstall-argocd
hack-uninstall-argocd: install-helm
	$(HELM) delete argocd --namespace argocd

.PHONY: hack-uninstall-cert-manager
hack-uninstall-cert-manager: install-helm
	$(HELM) delete cert-manager --namespace cert-manager

.PHONY: start-api-local
start-api-local:
	./hack/start-api.sh

.PHONY: start-controller-local
start-controller-local:
	KUBECONFIG=~/.kube/config \
	ARGOCD_KUBECONFIG=~/.kube/config \
    	go run ./cmd/controlplane controller

################################################################################
# Docs                                                                         #
#                                                                              #
# Convenience targets for building and running the documentation site natively #
# or in a container.                                                           #
################################################################################

.PHONY: hack-serve-docs
hack-serve-docs: hack-build-dev-tools
	$(CONTAINER_RUNTIME) run $(DOCKER_OPTS) -p $(DOCS_PORT):$(DOCS_PORT) kargo:dev-tools make serve-docs

.PHONY: serve-docs
serve-docs:
	cd docs && pnpm install && pnpm start
