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

DOCKER_IMAGE_NAME := $(DOCKER_REGISTRY)$(DOCKER_ORG)k8sta

ifdef HELM_REGISTRY
	HELM_REGISTRY := $(HELM_REGISTRY)/
endif

ifdef HELM_ORG
	HELM_ORG := $(HELM_ORG)/
endif

HELM_CHART_PREFIX := $(HELM_REGISTRY)$(HELM_ORG)

ifdef VERSION
	MUTABLE_DOCKER_TAG := latest
else
	VERSION            := $(GIT_VERSION)
	MUTABLE_DOCKER_TAG := edge
endif

IMMUTABLE_DOCKER_TAG := $(VERSION)

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
	grype $(DOCKER_IMAGE_NAME):$(IMMUTABLE_DOCKER_TAG) -f medium

.PHONY: generate-sbom
generate-sbom:
	syft $(DOCKER_IMAGE_NAME):$(IMMUTABLE_DOCKER_TAG) \
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
		-t $(DOCKER_IMAGE_NAME):$(IMMUTABLE_DOCKER_TAG) \
		-t $(DOCKER_IMAGE_NAME):$(MUTABLE_DOCKER_TAG) \
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
		-t $(DOCKER_IMAGE_NAME):$(IMMUTABLE_DOCKER_TAG) \
		-t $(DOCKER_IMAGE_NAME):$(MUTABLE_DOCKER_TAG) \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(GIT_VERSION) \
		--platform linux/amd64,linux/arm64 \
		--push \
		.

.PHONY: sign
sign:
	docker pull $(DOCKER_IMAGE_NAME):$(IMMUTABLE_DOCKER_TAG)
	docker pull $(DOCKER_IMAGE_NAME):$(MUTABLE_DOCKER_TAG)
	docker trust sign $(DOCKER_IMAGE_NAME):$(IMMUTABLE_DOCKER_TAG)
	docker trust sign $(DOCKER_IMAGE_NAME):$(MUTABLE_DOCKER_TAG)
	docker trust inspect --pretty $(DOCKER_IMAGE_NAME):$(IMMUTABLE_DOCKER_TAG)
	docker trust inspect --pretty $(DOCKER_IMAGE_NAME):$(MUTABLE_DOCKER_TAG)

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
	helm repo list | grep argo || helm repo add argo https://argoproj.github.io/argo-helm
	helm repo update
	helm upgrade argocd argo/argo-cd \
		--version 3.35.4 \
		--install \
		--create-namespace \
		--namespace argocd \
		--values hack/argo-cd-config/values.yaml \
		--wait \
		--timeout 300s
	kubectl apply -f hack/argo-cd-apps/apps.yaml

.PHONY: hack-kind-down
hack-kind-down:
	ctlptl delete -f hack/kind/cluster.yaml
