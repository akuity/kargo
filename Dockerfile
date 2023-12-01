####################################################################################################
# back-end-builder
####################################################################################################
FROM --platform=$BUILDPLATFORM golang:1.21.4-bookworm as back-end-builder

ARG TARGETOS
ARG TARGETARCH

ARG VERSION_PACKAGE=github.com/akuity/kargo/internal/version

ARG CGO_ENABLED=0

WORKDIR /kargo
COPY ["go.mod", "go.sum", "./"]
RUN go mod download
COPY api/ api/
COPY cmd/ cmd/
COPY internal/ internal/
COPY pkg/ pkg/

ARG VERSION
ARG GIT_COMMIT
ARG GIT_TREE_STATE

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
      -ldflags "-w -X ${VERSION_PACKAGE}.version=${VERSION} -X ${VERSION_PACKAGE}.buildDate=$(date -u +'%Y-%m-%dT%H:%M:%SZ') -X ${VERSION_PACKAGE}.gitCommit=${GIT_COMMIT} -X ${VERSION_PACKAGE}.gitTreeState=${GIT_TREE_STATE}" \
      -o bin/kargo \
      ./cmd/controlplane \
    && bin/kargo version

WORKDIR /kargo/bin

####################################################################################################
# ui-builder
####################################################################################################
FROM --platform=$BUILDPLATFORM docker.io/library/node:20.9.0 AS ui-builder

RUN npm install --global pnpm
WORKDIR /ui
COPY ["ui/package.json", "ui/pnpm-lock.yaml", "./"]

RUN pnpm install

COPY ["ui/", "."]

RUN NODE_ENV='production' pnpm run build

####################################################################################################
# tools
####################################################################################################
# `tools` stage allows us to take the leverage of the parallel build.
# For example, this stage can be cached and re-used when we have to rebuild code base.
FROM curlimages/curl:8.4.0 as tools

ARG TARGETOS
ARG TARGETARCH

WORKDIR /tools

RUN GRPC_HEALTH_PROBE_VERSION=v0.4.15 && \
    curl -fL -o /tools/grpc_health_probe https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-${TARGETOS}-${TARGETARCH} && \
    chmod +x /tools/grpc_health_probe

####################################################################################################
# back-end-dev
# - no UI
# - relies on go build that runs on host
# - supports development
# - not used for official image builds
####################################################################################################
FROM ghcr.io/akuity/kargo-render:v0.1.0-rc.33 as back-end-dev

USER root

COPY --from=tools /tools/ /usr/local/bin/
COPY bin/controlplane/kargo /usr/local/bin/kargo

USER 1000:0

CMD ["/usr/local/bin/kargo"]

####################################################################################################
# ui-dev
# - includes UI dev dependencies
# - runs with vite
# - supports development
# - not used for official image builds
####################################################################################################
FROM --platform=$BUILDPLATFORM docker.io/library/node:20.9.0 AS ui-dev

RUN npm install --global pnpm
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
FROM ghcr.io/akuity/kargo-render:v0.1.0-rc.33 as final

USER root

COPY --from=back-end-builder /kargo/bin/ /usr/local/bin/
COPY --from=tools /tools/ /usr/local/bin/
COPY --from=ui-builder /ui/build /ui/build

USER 1000:0

CMD ["/usr/local/bin/kargo"]
