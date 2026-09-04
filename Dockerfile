ARG BASE_IMAGE=kargo-base

####################################################################################################
# ui-builder
####################################################################################################
FROM --platform=$BUILDPLATFORM docker.io/library/node:24.20.0 AS ui-builder

ARG PNPM_VERSION=9.0.3
RUN npm install --global pnpm@${PNPM_VERSION}

WORKDIR /ui
COPY ["ui/package.json", "ui/pnpm-lock.yaml", "./"]

RUN pnpm install
COPY ["ui/", "."]

ARG VERSION
RUN NODE_ENV='production' VERSION=${VERSION} pnpm run build

####################################################################################################
# back-end-builder
####################################################################################################
# Always use the latest minor version of Go for anything we ship. A release
# branch lives until its EOL, and over that lifetime the Go minor version it
# started on may itself reach EOL. Building on the latest minor keeps released
# binaries off unsupported toolchains and picks up fixes for CVEs in the Go
# standard library.
FROM --platform=$BUILDPLATFORM golang:1.27.1-trixie AS back-end-builder

ARG TARGETOS
ARG TARGETARCH

ARG VERSION_PACKAGE=github.com/akuity/kargo/pkg/x/version

ARG CGO_ENABLED=0

WORKDIR /kargo
COPY ["api/go.mod", "api/go.sum", "api/"]
COPY ["go.mod", "go.sum", "./"]
RUN go mod download
COPY api/ api/
COPY pkg/ pkg/
COPY cmd/ cmd/
COPY --from=ui-builder /ui/build pkg/server/ui/

ARG VERSION
ARG GIT_COMMIT
ARG GIT_TREE_STATE

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
      -trimpath \
      -ldflags "-w -s" \
      -o bin/credential-helper \
      ./cmd/credential-helper

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
      -trimpath \
      -ldflags "-w -X ${VERSION_PACKAGE}.version=${VERSION} -X ${VERSION_PACKAGE}.buildDate=$(date -u +'%Y-%m-%dT%H:%M:%SZ') -X ${VERSION_PACKAGE}.gitCommit=${GIT_COMMIT} -X ${VERSION_PACKAGE}.gitTreeState=${GIT_TREE_STATE}" \
      -o bin/kargo \
      ./cmd/controlplane \
    && bin/kargo version

WORKDIR /kargo/bin

####################################################################################################
# helm-builder
####################################################################################################
# Helm is required by the kustomize-build promotion step's Helm plugin. We build
# it ourselves rather than shipping Helm's prebuilt release, because that release
# is compiled with whatever Go version upstream happened to use, and we inherit
# its stdlib CVEs. Building here means Helm always carries a current, patched Go
# stdlib.
#
# Building rather than downloading does not hide Helm from vulnerability
# scanners. Go embeds build metadata in the binary, and the module it records is
# the one containing the main package -- not the throwaway module used to drive
# the build. This binary therefore reports `helm.sh/helm/v3` at HELM_VERSION --
# the same module path and version that Helm's own release build reports.
# Scanners key on exactly that, so an advisory filed against Helm itself is
# matched here just as it would be against an upstream binary.
#
# The Helm version is intentionally ahead of the helm.sh/helm/v3 library in
# go.mod: the standalone binary carries no k8s dependency cascade, so we track
# the latest Helm 3 minor for CVE coverage.
#
# Source comes from the Go module proxy, so it is checksum-verified against
# sum.golang.org rather than trusted from a tarball download.
#
# As with the back-end-builder stage above, always use the latest minor version
# of Go for anything we ship.
FROM --platform=$BUILDPLATFORM golang:1.27.1-trixie AS helm-builder

ARG TARGETOS
ARG TARGETARCH

ARG HELM_VERSION=v3.21.4

WORKDIR /helm-build
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    go mod init helm-build && \
    go get helm.sh/helm/v3/cmd/helm@${HELM_VERSION}

# Helm's own Makefile derives the Kubernetes version it reports to charts from
# the k8s.io/client-go version it was built against, and injects it via ldflags.
# Without those, the binary falls back to the in-source default of v1.20.0, and
# any chart declaring a `kubeVersion` constraint newer than that fails to render.
# Mirror that derivation here rather than hardcoding a version, so it stays
# correct as HELM_VERSION moves.
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    K8S_VERSION="$(go list -f '{{.Version}}' -m k8s.io/client-go)" && \
    K8S_VERSION_MAJOR="$(( $(echo "${K8S_VERSION}" | cut -d. -f1 | tr -d v) + 1 ))" && \
    K8S_VERSION_MINOR="$(echo "${K8S_VERSION}" | cut -d. -f2)" && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
      -trimpath \
      -ldflags "-w -s \
        -X helm.sh/helm/v3/internal/version.version=${HELM_VERSION} \
        -X helm.sh/helm/v3/pkg/chartutil.k8sVersionMajor=${K8S_VERSION_MAJOR} \
        -X helm.sh/helm/v3/pkg/chartutil.k8sVersionMinor=${K8S_VERSION_MINOR} \
        -X helm.sh/helm/v3/pkg/lint/rules.k8sVersionMajor=${K8S_VERSION_MAJOR} \
        -X helm.sh/helm/v3/pkg/lint/rules.k8sVersionMinor=${K8S_VERSION_MINOR}" \
      -o /helm-build/helm \
      helm.sh/helm/v3/cmd/helm

####################################################################################################
# tools
####################################################################################################
# `tools` stage allows us to take the leverage of the parallel build.
# For example, this stage can be cached and re-used when we have to rebuild code base.
FROM curlimages/curl:8.21.0 AS tools

ARG TARGETOS
ARG TARGETARCH

WORKDIR /tools

RUN GRPC_HEALTH_PROBE_VERSION=v0.4.56 && \
    curl -fL -o /tools/grpc_health_probe https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-${TARGETOS}-${TARGETARCH} && \
    chmod +x /tools/grpc_health_probe

####################################################################################################
# back-end-dev
# - no UI
# - relies on go build that runs on host
# - supports development
# - not used for official image builds
####################################################################################################
FROM alpine:latest AS back-end-dev

RUN apk update && apk add ca-certificates git gpg gpg-agent openssh-client tini

# Match the published image: use the Helm binary we build ourselves (needed
# by the kustomize-build step's Helm plugin) rather than a distro package.
COPY --from=helm-builder /helm-build/helm /usr/local/bin/helm
COPY bin/credential-helper /usr/local/bin/credential-helper
COPY bin/controlplane/kargo /usr/local/bin/kargo

RUN adduser -D -H -u 1000 kargo
USER 1000:0

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/usr/local/bin/kargo"]

####################################################################################################
# ui-dev
# - includes UI dev dependencies
# - runs with vite
# - supports development
# - not used for official image builds
####################################################################################################
FROM --platform=$BUILDPLATFORM docker.io/library/node:24.20.0 AS ui-dev

ARG PNPM_VERSION=9.0.3
RUN npm install --global pnpm@${PNPM_VERSION}
WORKDIR /ui
COPY ["ui/package.json", "ui/pnpm-lock.yaml", "./"]

RUN pnpm install

COPY ["ui/", "."]

CMD ["pnpm", "dev"]

####################################################################################################
# final
# - the official image we publish
# - purposefully last so that it is the default target when building
####################################################################################################
FROM ${BASE_IMAGE}:latest-${TARGETARCH} AS final

COPY --from=back-end-builder /kargo/bin/ /usr/local/bin/
COPY --from=helm-builder /helm-build/helm /usr/local/bin/helm
COPY --from=tools /tools/grpc_health_probe /usr/local/bin/grpc_health_probe

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/usr/local/bin/kargo"]
