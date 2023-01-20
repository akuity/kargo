FROM --platform=$BUILDPLATFORM golang:1.19.3-bullseye as builder

ARG TARGETOS
ARG TARGETARCH

ARG VERSION_PACKAGE=github.com/akuityio/kargo/internal/common/version
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
RUN ln -s kargo kargo-controller
RUN ln -s kargo kargo-server

FROM ghcr.io/akuityio/bookkeeper-prototype:v0.1.0-alpha.2-rc.2 as final

USER root

COPY --from=builder /kargo/bin/ /usr/local/bin/

USER nonroot

CMD ["/usr/local/bin/kargo"]
