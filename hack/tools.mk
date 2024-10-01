# This Makefile contains targets for installing various development tools.
# The tools are installed in a local bin directory, making it easy to manage
# project-specific tool versions without affecting the system-wide installation.

################################################################################
# Directory and file path variables                                            #
################################################################################

HACK_DIR        ?= $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
TOOLS_MOD_FILE	:= $(HACK_DIR)/tools/go.mod
BIN_DIR			?= $(HACK_DIR)/bin
INCLUDE_DIR		?= $(HACK_DIR)/include

# Detect OS and architecture
OS		:= $(shell uname -s | tr A-Z a-z)
ARCH	:= $(shell uname -m)

################################################################################
# Tool versions                                                                #
################################################################################

GOLANGCI_LINT_VERSION	?= $(shell grep github.com/golangci/golangci-lint $(TOOLS_MOD_FILE) | awk '{print $$2}')
HELM_VERSION            ?= $(shell grep helm.sh/helm/v3 $(TOOLS_MOD_FILE) | awk '{print $$2}')
GOIMPORTS_VERSION       ?= $(shell grep golang.org/x/tools $(TOOLS_MOD_FILE) | awk '{print $$2}')
CODE_GENERATOR_VERSION	?= $(shell grep k8s.io/code-generator $(TOOLS_MOD_FILE) | awk '{print $$2}')
CONTROLLER_GEN_VERSION	?= $(shell grep k8s.io/controller-tools $(TOOLS_MOD_FILE) | awk '{print $$2}')
PROTOC_VERSION			?= v25.3
BUF_VERSION				?= $(shell grep github.com/bufbuild/buf $(TOOLS_MOD_FILE) | awk '{print $$2}')

################################################################################
# Tool targets                                                                 #
################################################################################

GOLANGCI_LINT	:= $(BIN_DIR)/golangci-lint-$(OS)-$(ARCH)-$(GOLANGCI_LINT_VERSION)
HELM            := $(BIN_DIR)/helm-$(OS)-$(ARCH)-$(HELM_VERSION)
GOIMPORTS       := $(BIN_DIR)/goimports-$(OS)-$(ARCH)-$(GOIMPORTS_VERSION)
GO_TO_PROTOBUF  := $(BIN_DIR)/go-to-protobuf-$(OS)-$(ARCH)-$(CODE_GENERATOR_VERSION)
PROTOC_GEN_GO   := $(BIN_DIR)/protoc-gen-gogo-$(OS)-$(ARCH)-$(CODE_GENERATOR_VERSION)
CONTROLLER_GEN  := $(BIN_DIR)/controller-gen-$(OS)-$(ARCH)-$(CONTROLLER_GEN_VERSION)
PROTOC          := $(BIN_DIR)/protoc-$(OS)-$(ARCH)-$(PROTOC_VERSION)
BUF             := $(BIN_DIR)/buf-$(OS)-$(ARCH)-$(BUF_VERSION)

$(GOLANGCI_LINT):
	$(call install-golangci-lint,$@,$(GOLANGCI_LINT_VERSION))

$(HELM):
	$(call install-helm,$@,$(HELM_VERSION))

$(GOIMPORTS):
	$(call go-install-tool,$@,golang.org/x/tools/cmd/goimports,$(GOIMPORTS_VERSION))

$(GO_TO_PROTOBUF):
	$(call go-install-tool,$@,k8s.io/code-generator/cmd/go-to-protobuf,$(CODE_GENERATOR_VERSION))

$(PROTOC_GEN_GO):
	$(call go-install-tool,$@,k8s.io/code-generator/cmd/go-to-protobuf/protoc-gen-gogo,$(CODE_GENERATOR_VERSION))

$(CONTROLLER_GEN):
	$(call go-install-tool,$@,sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_GEN_VERSION))

$(PROTOC):
	$(call install-protoc,$@,$(PROTOC_VERSION),$(HACK_DIR)/include)

$(BUF):
	$(call go-install-tool,$@,github.com/bufbuild/buf/cmd/buf,$(BUF_VERSION))

################################################################################
# Symlink targets                                                              #
################################################################################

GOLANGCI_LINT_LINK 	:= $(BIN_DIR)/golangci-lint
HELM_LINK 			:= $(BIN_DIR)/helm
GOIMPORTS_LINK 		:= $(BIN_DIR)/goimports
GO_TO_PROTOBUF_LINK	:= $(BIN_DIR)/go-to-protobuf
PROTOC_GEN_GO_LINK	:= $(BIN_DIR)/protoc-gen-gogo
CONTROLLER_GEN_LINK	:= $(BIN_DIR)/controller-gen
PROTOC_LINK			:= $(BIN_DIR)/protoc
BUF_LINK			:= $(BIN_DIR)/buf

.PHONY: $(GOLANGCI_LINT_LINK)
$(GOLANGCI_LINT_LINK): $(GOLANGCI_LINT)
	$(call create-symlink,$(GOLANGCI_LINT),$(GOLANGCI_LINT_LINK))

.PHONY: $(HELM_LINK)
$(HELM_LINK): $(HELM)
	$(call create-symlink,$(HELM),$(HELM_LINK))

.PHONY: $(GOIMPORTS_LINK)
$(GOIMPORTS_LINK): $(GOIMPORTS)
	$(call create-symlink,$(GOIMPORTS),$(GOIMPORTS_LINK))

.PHONY: $(GO_TO_PROTOBUF_LINK)
$(GO_TO_PROTOBUF_LINK): $(GO_TO_PROTOBUF)
	$(call create-symlink,$(GO_TO_PROTOBUF),$(GO_TO_PROTOBUF_LINK))

.PHONY: $(PROTOC_GEN_GO_LINK)
$(PROTOC_GEN_GO_LINK): $(PROTOC_GEN_GO)
	$(call create-symlink,$(PROTOC_GEN_GO),$(PROTOC_GEN_GO_LINK))

.PHONY: $(CONTROLLER_GEN_LINK)
$(CONTROLLER_GEN_LINK): $(CONTROLLER_GEN)
	$(call create-symlink,$(CONTROLLER_GEN),$(CONTROLLER_GEN_LINK))

.PHONY: $(PROTOC_LINK)
$(PROTOC_LINK): $(PROTOC)
	$(call create-symlink,$(PROTOC),$(PROTOC_LINK))

.PHONY: $(BUF_LINK)
$(BUF_LINK): $(BUF)
	$(call create-symlink,$(BUF),$(BUF_LINK))

################################################################################
# Alias targets                                                                #
################################################################################

TOOLS := install-golangci-lint install-helm install-goimports install-go-to-protobuf install-protoc-gen-gogo install-controller-gen install-protoc install-buf

.PHONY: install-tools
install-tools: $(TOOLS)

.PHONY: install-golangci-lint
install-golangci-lint: $(GOLANGCI_LINT) $(GOLANGCI_LINT_LINK)

.PHONY: install-helm
install-helm: $(HELM) $(HELM_LINK)

.PHONY: install-goimports
install-goimports: $(GOIMPORTS) $(GOIMPORTS_LINK)

.PHONY: install-go-to-protobuf
install-go-to-protobuf: $(GO_TO_PROTOBUF) $(GO_TO_PROTOBUF_LINK)

.PHONY: install-protoc-gen-gogo
install-protoc-gen-gogo: $(PROTOC_GEN_GO) $(PROTOC_GEN_GO_LINK)

.PHONY: install-controller-gen
install-controller-gen: $(CONTROLLER_GEN) $(CONTROLLER_GEN_LINK)

.PHONY: install-protoc
install-protoc: $(PROTOC) $(PROTOC_LINK)

.PHONY: install-buf
install-buf: $(BUF) $(BUF_LINK)

################################################################################
# Clean up targets                                                             #
################################################################################

# Clean up all installed tools and symlinks
.PHONY: clean-tools
clean-tools:
	rm -rf $(BIN_DIR)/*
	rm -rf $(INCLUDE_DIR)/*

# Update all tools
.PHONY: update-tools
update-tools: clean-tools install-tools

################################################################################
# Helper functions                                                             #
################################################################################

# go-install-tool installs a Go tool.
#
# $(1) binary path
# $(2) repo URL
# $(3) version
define go-install-tool
	@[ -f $(1) ] || { \
	set -e ;\
	TMP_DIR=$$(mktemp -d) ;\
	cd $$TMP_DIR ;\
	echo "Installing $(2)@$(3) to $(1)" ;\
	go mod init tmp ;\
	GOBIN=$$TMP_DIR go install $(2)@$(3) ;\
	mkdir -p $(dir $(1)) ;\
	mv $$TMP_DIR/$$(basename $(2)) $(1) ;\
	rm -rf $$TMP_DIR ;\
	}
endef

# install-golangci-lint installs golangci-lint.
#
# $(1) binary path
# $(2) version
define install-golangci-lint
	@[ -f $(1) ] || { \
	set -e ;\
	TMP_DIR=$$(mktemp -d) ;\
	cd $$TMP_DIR ;\
	echo "Installing golangci-lint $(2) to $(1)" ;\
	curl -fsSL -o install.sh https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh ;\
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

# PROTOC_OS and PROTOC_ARCH are used to determine the platform-specific zip file
# to download for protoc.
PROTOC_OS ?= $(shell uname -s | tr '[:upper:]' '[:lower:]')
ifeq ($(PROTOC_OS),darwin)
	PROTOC_OS := osx
endif

PROTOC_ARCH ?= $(shell uname -m)
ifeq ($(PROTOC_ARCH),amd64)
	override PROTOC_ARCH = x86_64
else ifeq ($(PROTOC_ARCH),aarch64)
	override PROTOC_ARCH = aarch_64
else ifeq ($(PROTOC_ARCH),arm64)
	override PROTOC_ARCH = aarch_64
endif

# install-protoc installs protoc.
#
# $(1) binary path
# $(2) version
# $(3) include path
define install-protoc
	@[ -f $(1) ] || { \
	set -e ;\
	TMP_DIR=$$(mktemp -d) ;\
	cd $$TMP_DIR ;\
	curl -fsSL -o protoc.zip https://github.com/protocolbuffers/protobuf/releases/download/$(2)/protoc-$(patsubst v%,%,$(2))-$(PROTOC_OS)-$(PROTOC_ARCH).zip ;\
	unzip -q protoc.zip ;\
	rm -rf $(3) ;\
	mkdir -p $(dir $(1)) $(3) ;\
	mv $$TMP_DIR/bin/protoc $(1) ;\
	mv -f $$TMP_DIR/include/* $(3) ;\
	rm -rf $$TMP_DIR ;\
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
