SHELL ?= /bin/bash

.DEFAULT_GOAL := build

################################################################################
# Version details                                                              #
################################################################################

# This will reliably return the short SHA1 of HEAD or, if the working directory
# is dirty, will return that + "-dirty"
GIT_VERSION = $(shell git describe --always --abbrev=7 --dirty --match=NeVeRmAtCh)

################################################################################
# Containerized development environment-- or lack thereof                      #
################################################################################

ifneq ($(SKIP_DOCKER),true)
	PROJECT_ROOT := $(dir $(realpath $(firstword $(MAKEFILE_LIST))))
	GO_DEV_IMAGE := brigadecore/go-tools:v0.8.0

	GO_DOCKER_CMD := docker run \
		-it \
		--rm \
		-e SKIP_DOCKER=true \
		-e GOCACHE=/workspaces/k8sta/.gocache \
		-v $(PROJECT_ROOT):/workspaces/k8sta \
		-w /workspaces/k8sta \
		$(GO_DEV_IMAGE)

	HELM_IMAGE := brigadecore/helm-tools:v0.4.0

	HELM_DOCKER_CMD := docker run \
	  -it \
		--rm \
		-e SKIP_DOCKER=true \
		-e HELM_PASSWORD=$${HELM_PASSWORD} \
		-v $(PROJECT_ROOT):/workspaces/k8sta \
		-w /workspaces/k8sta \
		$(HELM_IMAGE)
endif

################################################################################
# Docker images and charts we build and publish                                #
################################################################################

ifdef DOCKER_REGISTRY
	DOCKER_REGISTRY := $(DOCKER_REGISTRY)/
endif

ifdef DOCKER_ORG
	DOCKER_ORG := $(DOCKER_ORG)/
endif

ifndef VERSION
	VERSION            := $(GIT_VERSION)
endif

DOCKER_IMAGE_NAME := $(DOCKER_REGISTRY)$(DOCKER_ORG)k8sta:$(VERSION)

ifdef HELM_REGISTRY
	HELM_REGISTRY := $(HELM_REGISTRY)/
endif

ifdef HELM_ORG
	HELM_ORG := $(HELM_ORG)/
endif

HELM_CHART_PREFIX := $(HELM_REGISTRY)$(HELM_ORG)

################################################################################
# Tests                                                                        #
################################################################################

.PHONY: lint
lint:
	$(GO_DOCKER_CMD) golangci-lint run --config golangci.yaml

.PHONY: test-unit
test-unit:
	$(GO_DOCKER_CMD) go test \
		-v \
		-timeout=60s \
		-race \
		-coverprofile=coverage.txt \
		-covermode=atomic \
		./...

.PHONY: lint-chart
lint-chart:
	$(HELM_DOCKER_CMD) sh -c ' \
		cd charts/k8sta && \
		helm dep up && \
		helm lint . \
	'

################################################################################
# Image security                                                               #
################################################################################

.PHONY: scan
scan:
	grype $(DOCKER_IMAGE_NAME) -f high

.PHONY: generate-sbom
generate-sbom:
	syft $(DOCKER_IMAGE_NAME) \
		-o spdx-json \
		--file ./artifacts/k8sta-$(VERSION)-SBOM.json

.PHONY: publish-sbom
publish-sbom: generate-sbom
	ghr \
		-u $(GITHUB_ORG) \
		-r $(GITHUB_REPO) \
		-c $$(git rev-parse HEAD) \
		-t $${GITHUB_TOKEN} \
		-n ${VERSION} \
		${VERSION} ./artifacts/k8sta-$(VERSION)-SBOM.json

################################################################################
# Build                                                                        #
################################################################################

.PHONY: build
build:
	docker buildx build \
		-t $(DOCKER_IMAGE_NAME) \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(GIT_VERSION) \
		--platform linux/amd64,linux/arm64 \
		.

################################################################################
# Publish                                                                      #
################################################################################

.PHONY: publish
publish: push push-chart

.PHONY: push
push:
	docker buildx build \
		-t $(DOCKER_IMAGE_NAME) \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(GIT_VERSION) \
		--platform linux/amd64,linux/arm64 \
		--push \
		.

.PHONY: sign
sign:
	docker pull $(DOCKER_IMAGE_NAME)
	docker trust sign $(DOCKER_IMAGE_NAME)
	docker trust inspect --pretty $(DOCKER_IMAGE_NAME)

.PHONY: push-chart
push-chart:
	$(HELM_DOCKER_CMD) sh	-c ' \
		helm registry login $(HELM_REGISTRY) -u $(HELM_USERNAME) -p $${HELM_PASSWORD} && \
		cd charts/k8sta && \
		helm dep up && \
		helm package . --version $(VERSION) --app-version $(VERSION) && \
		helm push k8sta-$(VERSION).tgz oci://$(HELM_REGISTRY)$(HELM_ORG) \
	'

################################################################################
# Targets to facilitate hacking on K8sTA                                       #
################################################################################

.PHONY: hack-kind-up
hack-kind-up:
	ctlptl apply -f hack/kind/cluster.yaml
	helm upgrade istio-base base \
		--repo https://istio-release.storage.googleapis.com/charts \
		--version 1.15.0-beta.0 \
		--install \
		--create-namespace \
		--namespace istio-system \
		--wait
	helm upgrade istiod istiod \
		--repo https://istio-release.storage.googleapis.com/charts \
		--version 1.15.0-beta.0 \
		--install \
		--namespace istio-system \
		--wait
	kubectl apply -f https://raw.githubusercontent.com/istio/istio/release-1.14/samples/addons/prometheus.yaml
	kubectl get namespace istio-ingress || kubectl create namespace istio-ingress
	kubectl label namespace istio-ingress istio-injection=enabled --overwrite
	helm upgrade istio-ingress gateway \
		--repo https://istio-release.storage.googleapis.com/charts \
		--version 1.15.0-beta.0 \
		--install \
		--namespace istio-ingress \
		--set service.type=NodePort \
		--set 'service.ports[0].name=status-port' \
		--set 'service.ports[0].port=15021' \
		--set 'service.ports[0].protocol=TCP' \
		--set 'service.ports[0].targetPort=15021' \
		--set 'service.ports[1].name=http2' \
		--set 'service.ports[1].port=80' \
		--set 'service.ports[1].protocol=TCP' \
		--set 'service.ports[1].targetPort=80' \
		--set 'service.ports[1].nodePort=30080' \
		--set 'service.ports[2].name=https' \
		--set 'service.ports[2].port=443' \
		--set 'service.ports[2].protocol=TCP' \
		--set 'service.ports[2].targetPort=443' \
		--wait
	helm upgrade argo-cd argo-cd \
		--repo https://argoproj.github.io/argo-helm \
		--version 4.10.5 \
		--install \
		--create-namespace \
		--namespace argo-cd \
		--set 'configs.secret.argocdServerAdminPassword=$$2a$$10$$5vm8wXaSdbuff0m9l21JdevzXBzJFPCi8sy6OOnpZMAG.fOXL7jvO' \
		--set server.extensions.enabled=true \
		--set server.extensions.contents[0].name=argo-rollouts \
		--set server.extensions.contents[0].url=https://github.com/argoproj-labs/rollout-extension/releases/download/v0.2.0/extension.tar \
		--set server.service.type=NodePort \
		--set server.service.nodePortHttp=30081 \
		--wait
	helm upgrade argo-rollouts argo-rollouts \
		--repo https://argoproj.github.io/argo-helm \
		--version 2.18.0 \
		--install \
		--create-namespace \
		--namespace argo-rollouts \
		--wait

.PHONY: hack-kind-down
hack-kind-down:
	ctlptl delete -f hack/kind/cluster.yaml

################################################################################
# Kubebuilder stuffs                                                           #
################################################################################

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen

## Tool Versions
CONTROLLER_TOOLS_VERSION ?= v0.9.0

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	echo $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=charts/k8sta/crds

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."
