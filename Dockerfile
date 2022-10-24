FROM --platform=$BUILDPLATFORM ghcr.io/akuityio/k8sta-tools:v0.3.0 as builder

ARG TARGETOS
ARG TARGETARCH

ARG VERSION_PACKAGE=github.com/akuityio/k8sta/internal/common/version
ARG VERSION
ARG CGO_ENABLED=0

WORKDIR /k8sta
COPY go.mod .
COPY go.sum .
# TODO: This is a temporary workaround because the bookkeeper module is private.
# We won't need this once bookkeeper is public. We can revoke the token before
# K8sTA is public.
RUN git config --global url."https://krancour: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@github.com".insteadOf https://github.com
RUN go mod download
COPY . .

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
      -ldflags "-w -X ${VERSION_PACKAGE}.version=${VERSION} -X ${VERSION_PACKAGE}.buildDate=$(date -u +'%Y-%m-%dT%H:%M:%SZ')" \
      -o bin/k8sta \
      ./cmd \
    && bin/k8sta version

WORKDIR /k8sta/bin
RUN ln -s k8sta k8sta-controller
RUN ln -s k8sta k8sta-server

FROM ghcr.io/akuityio/bookkeeper-prototype:v0.1.0-alpha.1-rc.2 as final

USER root

COPY --from=builder /k8sta/bin/ /usr/local/bin/

USER nonroot

CMD ["/usr/local/bin/k8sta"]
