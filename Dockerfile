FROM --platform=$BUILDPLATFORM brigadecore/go-tools:v0.8.0 as builder

ARG TARGETOS
ARG TARGETARCH

ARG HELM_VERSION=v3.9.4
RUN curl -L -o /tmp/helm.tar.gz \
      https://get.helm.sh/helm-${HELM_VERSION}-linux-${TARGETARCH}.tar.gz \
    && tar xvfz /tmp/helm.tar.gz -C /usr/local/bin --strip-components 1

ARG KUSTOMIZE_VERSION=v4.5.5
RUN curl -L -o /tmp/kustomize.tar.gz \
      https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2F${KUSTOMIZE_VERSION}/kustomize_${KUSTOMIZE_VERSION}_linux_${TARGETARCH}.tar.gz \
    && tar xvfz /tmp/kustomize.tar.gz -C /usr/local/bin

ARG YTT_VERSION=v0.41.1
RUN curl -L -o /usr/local/bin/ytt \
      https://github.com/vmware-tanzu/carvel-ytt/releases/download/${YTT_VERSION}/ytt-linux-${TARGETARCH} \
      && chmod 755 /usr/local/bin/ytt

ARG VERSION
ARG COMMIT
ARG CGO_ENABLED=0

WORKDIR /k8sta
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY api/ api/
COPY cmd/ cmd/
COPY internal/ internal/
COPY config.go .
COPY main.go .

RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
      -o bin/k8sta \
      -ldflags "-w -X github.com/akuityio/k8sta/internal/common/version.version=${VERSION} -X github.com/akuityio/k8sta/internal/common/version.commit=${COMMIT}" \
      .

WORKDIR /k8sta/bin
RUN ln -s k8sta k8sta-controller
RUN ln -s k8sta k8sta-server

FROM alpine:3.15.4 as final

RUN apk update \
    && apk add git openssh-client \
    && addgroup -S -g 65532 nonroot \
    && adduser -S -D -H -u 65532 -g nonroot -G nonroot nonroot

COPY --from=builder /usr/local/bin/helm /usr/local/bin/
COPY --from=builder /usr/local/bin/kustomize /usr/local/bin/
COPY --from=builder /usr/local/bin/ytt /usr/local/bin/
COPY --from=builder /k8sta/bin/ /usr/local/bin/

USER nonroot

CMD ["/usr/local/bin/k8sta"]
