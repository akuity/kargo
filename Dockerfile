FROM --platform=$BUILDPLATFORM golang:1.20.2-bullseye as builder

ARG TARGETOS
ARG TARGETARCH

ARG VERSION_PACKAGE=github.com/akuity/kargo/internal/version
ARG VERSION
ARG CGO_ENABLED=0

WORKDIR /kargo
COPY go.mod .
COPY go.sum .
# TODO: This is a temporary workaround because the bookkeeper module is private.
# We won't need this once bookkeeper is public. We can revoke the token before
# Kargo is public.
RUN git config --global url."https://krancour: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@github.com".insteadOf https://github.com
RUN go mod download
COPY . .

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
      -ldflags "-w -X ${VERSION_PACKAGE}.version=${VERSION} -X ${VERSION_PACKAGE}.buildDate=$(date -u +'%Y-%m-%dT%H:%M:%SZ')" \
      -o bin/kargo \
      ./cmd \
    && bin/kargo version

WORKDIR /kargo/bin


# `tools` stage allows us to take the leverage of the parallel build.
# For example, this stage can be cached and re-used when we have to rebuild code base.
FROM curlimages/curl:7.88.1 as tools

ARG TARGETOS
ARG TARGETARCH

WORKDIR /tools

RUN GRPC_HEALTH_PROBE_VERSION=v0.4.15 && \
    curl -fL -o /tools/grpc_health_probe https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-${TARGETOS}-${TARGETARCH} && \
    chmod +x /tools/grpc_health_probe


FROM ghcr.io/akuityio/bookkeeper-prototype:v0.1.0-alpha.2-rc.2 as final

USER root

COPY --from=builder /kargo/bin/ /usr/local/bin/
COPY --from=tools /tools/ /usr/local/bin/

USER nonroot

CMD ["/usr/local/bin/kargo"]
