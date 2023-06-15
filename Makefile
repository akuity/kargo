SHELL ?= /bin/bash

ARGO_CD_CHART_VERSION := 5.21.0
BUF_LINT_ERROR_FORMAT ?= text
GO_LINT_ERROR_FORMAT ?= colored-line-number
CERT_MANAGER_CHART_VERSION := 1.11.0

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
	go mod vendor
	buf lint api --error-format=$(BUF_LINT_ERROR_FORMAT)

.PHONY: lint-charts
lint-charts:
	cd charts/kargo && \
	helm dep up && \
	helm lint .
	cd charts/kargo-kit && \
	helm dep up && \
	helm lint .
	cd charts/argocd-kit && \
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
# Code generation: To be run after modifications to API types                  #
################################################################################

.PHONY: codegen
codegen:
	go mod vendor
	./hack/generate-proto.sh
	buf generate api
	./hack/api/apply-patches.sh
	go mod tidy
	rm -rf ./vendor
	controller-gen \
		rbac:roleName=manager-role \
		crd \
		webhook \
		paths=./... \
		output:crd:artifacts:config=charts/kargo/crds
	rm -rf charts/kargo-kit/crds
	cp -R charts/kargo/crds charts/kargo-kit/crds
	controller-gen \
		object:headerFile=hack/boilerplate.go.txt \
		paths=./... \

################################################################################
# Hack: Targets to help you hack                                               #
#                                                                              #
# These targets minimize required developer setup by executing in a container  #
# that is pre-loaded with required tools.                                      #
################################################################################

DOCKER_CMD := docker run \
	-it \
	--rm \
	-v gomodcache:/go/pkg/mod \
	-v $(dir $(realpath $(firstword $(MAKEFILE_LIST)))):/workspaces/kargo \
	-w /workspaces/kargo \
	kargo:dev-tools

.PHONY: hack-build-dev-tools
hack-build-dev-tools:
	docker build -f Dockerfile.dev -t kargo:dev-tools .

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

.PHONY: hack-build
hack-build:
	docker build \
		--build-arg GIT_COMMIT=$(shell git rev-parse HEAD) \
		--build-arg GIT_TREE_STATE=$(shell if [ -z "`git status --porcelain`" ]; then echo "clean" ; else echo "dirty"; fi) \
		--tag kargo:dev \
		.

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
