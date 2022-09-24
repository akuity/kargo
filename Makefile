SHELL ?= /bin/bash

ifneq ($(SKIP_DOCKER),true)
	DOCKER_CMD := docker run \
		-it \
		--rm \
		-e SKIP_DOCKER=true \
		-v gomodcache:/go/pkg/mod \
		-v $(dir $(realpath $(firstword $(MAKEFILE_LIST)))):/workspaces/k8sta \
		-w /workspaces/k8sta \
		ghcr.io/akuityio/k8sta-tools:v0.2.0
endif

################################################################################
# Tests                                                                        #
################################################################################

.PHONY: lint
lint:
	$(DOCKER_CMD) golangci-lint run --config golangci.yaml

.PHONY: test-unit
test-unit:
	$(DOCKER_CMD) go test \
		-v \
		-timeout=120s \
		-race \
		-coverprofile=coverage.txt \
		-covermode=atomic \
		./...

.PHONY: lint-chart
lint-chart:
	$(DOCKER_CMD) sh -c ' \
		cd charts/k8sta && \
		helm dep up && \
		helm lint . \
	'

################################################################################
# Code generation: To be run after modifications to API types                  #
################################################################################

.PHONY: generate
generate:
	$(DOCKER_CMD) sh -c ' \
		controller-gen \
			rbac:roleName=manager-role \
			crd \
			webhook \
			paths=./... \
			output:crd:artifacts:config=charts/k8sta/crds && \
		controller-gen \
			object:headerFile=hack/boilerplate.go.txt \
			paths=./... \
	'

################################################################################
# Helm chart                                                                   #
################################################################################

ifdef HELM_REGISTRY
	HELM_REGISTRY := $(HELM_REGISTRY)/
endif

ifdef HELM_ORG
	HELM_ORG := $(HELM_ORG)/
endif

.PHONY: push-chart
push-chart:
	sh -c ' \
		cd charts/k8sta && \
		helm dep up && \
		helm package . --version $(VERSION) --app-version $(VERSION) && \
		helm push k8sta-$(VERSION).tgz oci://$(HELM_REGISTRY)$(HELM_ORG) \
	'

################################################################################
# Hack: Manage a kind cluster with Istio, Argo CD, and Argo Rollouts           #
# pre-installed                                                                #
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
