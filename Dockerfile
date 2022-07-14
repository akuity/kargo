FROM --platform=$BUILDPLATFORM brigadecore/go-tools:v0.8.0 as builder

ARG VERSION
ARG COMMIT
ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=0

WORKDIR /k8sta
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
COPY main.go .

RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
  -o bin/k8sta \
  -ldflags "-w -X github.com/akuityio/k8sta/internal/common/version.version=$VERSION -X github.com/akuityio/k8sta/internal/common/version.commit=$COMMIT" \
  .

WORKDIR /k8sta/bin
RUN ln -s k8sta k8sta-controller
RUN ln -s k8sta k8sta-promoter
RUN ln -s k8sta k8sta-server

FROM alpine:3.15.4 as final

RUN apk update \
    && apk add git openssh-client \
    && addgroup -S -g 65532 nonroot \
    && adduser -S -D -u 65532 -g nonroot -G nonroot nonroot

COPY --chown=nonroot:nonroot cmd/promoter/ssh_config /home/nonroot/.ssh/config
COPY --from=builder /k8sta/bin/ /usr/local/bin/

CMD ["/usr/local/bin/k8sta"]
