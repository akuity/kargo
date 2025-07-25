ARG BASE_IMAGE=kargo-base

####################################################################################################
# ui-builder
####################################################################################################
FROM --platform=$BUILDPLATFORM docker.io/library/node:24.3.0 AS ui-builder

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
FROM --platform=$BUILDPLATFORM golang:1.24.5-bookworm AS back-end-builder

ARG TARGETOS
ARG TARGETARCH

ARG VERSION_PACKAGE=github.com/akuity/kargo/pkg/x/version

ARG CGO_ENABLED=0

WORKDIR /kargo
COPY ["api/go.mod", "api/go.sum", "api/"]
COPY ["pkg/go.mod", "pkg/go.sum", "pkg/"]
COPY ["go.mod", "go.sum", "./"]
RUN go mod download
COPY api/ api/
COPY pkg/ pkg/
COPY cmd/ cmd/
COPY internal/ internal/
COPY --from=ui-builder /ui/build internal/server/ui/

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
# tools
####################################################################################################
# `tools` stage allows us to take the leverage of the parallel build.
# For example, this stage can be cached and re-used when we have to rebuild code base.
FROM curlimages/curl:8.15.0 AS tools

ARG TARGETOS
ARG TARGETARCH

WORKDIR /tools

RUN GRPC_HEALTH_PROBE_VERSION=v0.4.39 && \
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
FROM --platform=$BUILDPLATFORM docker.io/library/node:24.3.0 AS ui-dev

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
COPY --from=tools /tools/ /usr/local/bin/

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/usr/local/bin/kargo"]
