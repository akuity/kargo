# This Makefile contains targets for installing various development tools.
# The tools are installed in a local bin directory, making it easy to manage
# project-specific tool versions without affecting the system-wide installation.
#
# NOTE: Go-based tools (controller-gen, goimports, swag, go-swagger,
# oapi-codegen, ctlptl, kind) are no longer installed here. They are declared
# as `tool` directives in the main go.mod and invoked via `go tool <name>`.
# Only tools distributed as prebuilt binaries remain below.

################################################################################
# Directory and file path variables                                            #
################################################################################

HACK_DIR        ?= $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
GO_MOD_FILE     := $(HACK_DIR)/../go.mod
BIN_DIR			?= $(HACK_DIR)/bin

# Detect OS and architecture
OS		:= $(shell uname -s | tr A-Z a-z)
ARCH	:= $(shell uname -m)

################################################################################
# Tool versions                                                                #
################################################################################

# golangci-lint is intentionally installed from its official prebuilt binaries
# (not `go tool`): building it from source is slow and, because it bundles many
# linters, a source build can produce different results than the release binary.
# It is bumped manually as new lints surface, so the version is pinned here.
GOLANGCI_LINT_VERSION	?= v2.10.1
# Helm is distributed as a prebuilt binary, but we keep its version in sync with
# the helm.sh/helm/v3 module pinned in the main go.mod.
HELM_VERSION            ?= $(shell grep -m 1 -E '^[[:space:]]+helm.sh/helm/v3 v' $(GO_MOD_FILE) | awk '{print $$2}')
QUILL_VERSION			?= v0.5.1
TILT_VERSION			?= v0.36.3
K3D_VERSION				?= v5.8.3
JQ_VERSION				?= 1.8.1

################################################################################
# Tool targets                                                                 #
################################################################################

GOLANGCI_LINT	:= $(BIN_DIR)/golangci-lint-$(OS)-$(ARCH)-$(GOLANGCI_LINT_VERSION)
HELM            := $(BIN_DIR)/helm-$(OS)-$(ARCH)-$(HELM_VERSION)
QUILL		   	:= $(BIN_DIR)/quill-$(OS)-$(ARCH)-$(QUILL_VERSION)
TILT            := $(BIN_DIR)/tilt-$(OS)-$(ARCH)-$(TILT_VERSION)
K3D             := $(BIN_DIR)/k3d-$(OS)-$(ARCH)-$(K3D_VERSION)
JQ              := $(BIN_DIR)/jq-$(OS)-$(ARCH)-$(JQ_VERSION)

$(GOLANGCI_LINT):
	$(call install-golangci-lint,$@,$(GOLANGCI_LINT_VERSION))

$(HELM):
	$(call install-helm,$@,$(HELM_VERSION))

$(QUILL):
	$(call install-quill,$@,$(QUILL_VERSION))

$(TILT):
	$(call install-tilt,$@,$(TILT_VERSION))

$(K3D):
	$(call install-k3d,$@,$(K3D_VERSION))

$(JQ):
	$(call install-jq,$@,$(JQ_VERSION))

################################################################################
# Symlink targets                                                              #
################################################################################

GOLANGCI_LINT_LINK 	:= $(BIN_DIR)/golangci-lint
HELM_LINK 			:= $(BIN_DIR)/helm
QUILL_LINK			:= $(BIN_DIR)/quill
TILT_LINK			:= $(BIN_DIR)/tilt
K3D_LINK			:= $(BIN_DIR)/k3d
JQ_LINK				:= $(BIN_DIR)/jq

.PHONY: $(GOLANGCI_LINT_LINK)
$(GOLANGCI_LINT_LINK): $(GOLANGCI_LINT)
	$(call create-symlink,$(GOLANGCI_LINT),$(GOLANGCI_LINT_LINK))

.PHONY: $(HELM_LINK)
$(HELM_LINK): $(HELM)
	$(call create-symlink,$(HELM),$(HELM_LINK))

.PHONY: $(QUILL_LINK)
$(QUILL_LINK): $(QUILL)
	$(call create-symlink,$(QUILL),$(QUILL_LINK))

.PHONY: $(TILT_LINK)
$(TILT_LINK): $(TILT)
	$(call create-symlink,$(TILT),$(TILT_LINK))

.PHONY: $(K3D_LINK)
$(K3D_LINK): $(K3D)
	$(call create-symlink,$(K3D),$(K3D_LINK))

.PHONY: $(JQ_LINK)
$(JQ_LINK): $(JQ)
	$(call create-symlink,$(JQ),$(JQ_LINK))

################################################################################
# Alias targets                                                                #
################################################################################

TOOLS := install-golangci-lint install-helm install-quill install-tilt install-k3d install-jq

.PHONY: install-tools
install-tools: $(TOOLS)

.PHONY: install-golangci-lint
install-golangci-lint: $(GOLANGCI_LINT) $(GOLANGCI_LINT_LINK)

.PHONY: install-helm
install-helm: $(HELM) $(HELM_LINK)

.PHONY: install-quill
install-quill: $(QUILL) $(QUILL_LINK)

.PHONY: install-tilt
install-tilt: $(TILT) $(TILT_LINK)

.PHONY: install-k3d
install-k3d: $(K3D) $(K3D_LINK)

.PHONY: install-jq
install-jq: $(JQ) $(JQ_LINK)

################################################################################
# Clean up targets                                                             #
################################################################################

# Clean up all installed tools and symlinks
.PHONY: clean-tools
clean-tools:
	rm -rf $(BIN_DIR)/*

# Update all tools
.PHONY: update-tools
update-tools: clean-tools install-tools

################################################################################
# Helper functions                                                             #
################################################################################

# install-golangci-lint installs golangci-lint from its official prebuilt
# binaries.
#
# $(1) binary path
# $(2) version
define install-golangci-lint
	@[ -f $(1) ] || { \
	set -e ;\
	TMP_DIR=$$(mktemp -d) ;\
	cd $$TMP_DIR ;\
	echo "Installing golangci-lint $(2) to $(1)" ;\
	curl -fsSL -o install.sh https://golangci-lint.run/install.sh ;\
	chmod 0700 install.sh ;\
	./install.sh -b $$TMP_DIR $(2) ;\
	mkdir -p $(dir $(1)) ;\
	mv $$TMP_DIR/golangci-lint $(1) ;\
	rm -rf $$TMP_DIR ;\
	}
endef

# install-helm installs Helm.
#
# $(1) binary path
# $(2) version
define install-helm
	@[ -f $(1) ] || { \
	set -e ;\
	TMP_DIR=$$(mktemp -d) ;\
	cd $$TMP_DIR ;\
	echo "Installing helm $(2) to $(1)" ;\
	curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 ;\
	chmod 0700 get_helm.sh ;\
	PATH="$$TMP_DIR:$$PATH" HELM_INSTALL_DIR=$$TMP_DIR USE_SUDO="false" DESIRED_VERSION="$(2)" ./get_helm.sh ;\
	mkdir -p $(dir $(1)) ;\
	mv $$TMP_DIR/helm $(1) ;\
	rm -rf $$TMP_DIR ;\
	}
endef

# install-quill installs Quill.
#
# $(1) binary path
# $(2) version
define install-quill
	@[ -f $(1) ] || { \
	set -e ;\
	TMP_DIR=$$(mktemp -d) ;\
	cd $$TMP_DIR ;\
	echo "Installing quill $(2) to $(1)" ;\
	curl -fsSL -o install.sh https://get.anchore.io/quill ;\
	chmod 0700 install.sh ;\
	./install.sh -b $$TMP_DIR $(2) ;\
	mkdir -p $(dir $(1)) ;\
	mv $$TMP_DIR/quill $(1) ;\
	rm -rf $$TMP_DIR ;\
	}
endef

# TILT_OS and TILT_ARCH are used to determine the platform-specific tarball to
# download for tilt.
TILT_OS ?= $(OS)
ifeq ($(TILT_OS),darwin)
	TILT_OS := mac
endif

TILT_ARCH ?= $(ARCH)
ifeq ($(TILT_ARCH),aarch64)
	override TILT_ARCH = arm64
endif

# install-tilt installs Tilt.
#
# $(1) binary path
# $(2) version
define install-tilt
	@[ -f $(1) ] || { \
	set -e ;\
	TMP_DIR=$$(mktemp -d) ;\
	cd $$TMP_DIR ;\
	echo "Installing tilt $(2) to $(1)" ;\
	curl -fsSL -o tilt.tar.gz https://github.com/tilt-dev/tilt/releases/download/$(2)/tilt.$(patsubst v%,%,$(2)).$(TILT_OS).$(TILT_ARCH).tar.gz ;\
	tar xzf tilt.tar.gz ;\
	mkdir -p $(dir $(1)) ;\
	mv $$TMP_DIR/tilt $(1) ;\
	rm -rf $$TMP_DIR ;\
	}
endef

# K3D_ARCH is used to determine the platform-specific binary to download for k3d.
K3D_ARCH ?= $(ARCH)
ifeq ($(K3D_ARCH),x86_64)
	override K3D_ARCH = amd64
else ifeq ($(K3D_ARCH),aarch64)
	override K3D_ARCH = arm64
endif

# install-k3d installs k3d.
#
# $(1) binary path
# $(2) version
define install-k3d
	@[ -f $(1) ] || { \
	set -e ;\
	echo "Installing k3d $(2) to $(1)" ;\
	mkdir -p $(dir $(1)) ;\
	curl -fsSL -o $(1) https://github.com/k3d-io/k3d/releases/download/$(2)/k3d-$(OS)-$(K3D_ARCH) ;\
	chmod 0755 $(1) ;\
	}
endef

# JQ_OS and JQ_ARCH are used to determine the platform-specific binary to
# download for jq.
JQ_OS ?= $(OS)
ifeq ($(JQ_OS),darwin)
	JQ_OS := macos
endif

JQ_ARCH ?= $(ARCH)
ifeq ($(JQ_ARCH),x86_64)
	override JQ_ARCH = amd64
else ifeq ($(JQ_ARCH),aarch64)
	override JQ_ARCH = arm64
endif

# install-jq installs jq.
#
# $(1) binary path
# $(2) version
define install-jq
	@[ -f $(1) ] || { \
	set -e ;\
	echo "Installing jq $(2) to $(1)" ;\
	mkdir -p $(dir $(1)) ;\
	curl -fsSL -o $(1) https://github.com/jqlang/jq/releases/download/jq-$(2)/jq-$(JQ_OS)-$(JQ_ARCH) ;\
	chmod 0755 $(1) ;\
	}
endef

# create-symlink creates a relative symlink to the platform-specific binary.
#
# $(1) platform-specific binary path
# $(2) symlink path
define create-symlink
	@if [ ! -e $(2) ] || [ "$$(readlink $(2))" != "$$(basename $(1))" ]; then \
		echo "Creating symlink: $(2) -> $$(basename $(1))"; \
		ln -sf $$(basename $(1)) $(2); \
	fi
endef
