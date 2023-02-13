SHELL ?= /bin/bash

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
lint:
	golangci-lint run --config golangci.yaml

.PHONY: test-unit
test-unit:
	go test \
		-v \
		-timeout=120s \
		-race \
		-coverprofile=coverage.txt \
		-covermode=atomic \
		./...

.PHONY: lint-chart
lint-chart:
	cd charts/kargo && \
	helm dep up && \
	helm lint .

################################################################################
# Code generation: To be run after modifications to API types                  #
################################################################################

.PHONY: codegen
codegen:
	controller-gen \
		rbac:roleName=manager-role \
		crd \
		webhook \
		paths=./... \
		output:crd:artifacts:config=charts/kargo/crds && \
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

.PHONY: hack-test-unit
hack-test-unit: hack-build-dev-tools
	$(DOCKER_CMD) make test-unit

.PHONY: hack-lint-chart
hack-lint-chart: hack-build-dev-tools
	$(DOCKER_CMD) make lint-chart

.PHONY: hack-codegen
hack-codegen: hack-build-dev-tools
	$(DOCKER_CMD) make codegen

.PHONY: hack-build
hack-build:
	docker build . -t kargo:dev

.PHONY: hack-kind-up
hack-kind-up:
	ctlptl apply -f hack/kind/cluster.yaml
	make hack-install-argocd

.PHONY: hack-k3d-up
hack-k3d-up:
	ctlptl apply -f hack/k3d/cluster.yaml
	make hack-install-argocd

.PHONY: hack-kind-down
hack-kind-down:
	ctlptl delete -f hack/kind/cluster.yaml

.PHONY: hack-k3d-down
hack-k3d-down:
	ctlptl delete -f hack/k3d/cluster.yaml

.PHONY: hack-install-argocd
hack-install-argocd:
	helm upgrade argo-cd argo-cd \
		--repo https://argoproj.github.io/argo-helm \
		--version 5.5.6 \
		--install \
		--create-namespace \
		--namespace argo-cd \
		--set 'configs.secret.argocdServerAdminPassword=$$2a$$10$$5vm8wXaSdbuff0m9l21JdevzXBzJFPCi8sy6OOnpZMAG.fOXL7jvO' \
		--set server.service.type=NodePort \
		--set server.service.nodePortHttp=30080 \
		--wait

.PHONY: hack-add-rollouts
hack-add-rollouts:
	helm upgrade argo-cd argo-cd \
		--repo https://argoproj.github.io/argo-helm \
		--version 5.5.6 \
		--namespace argo-cd \
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
