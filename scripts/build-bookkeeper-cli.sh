#!/usr/bin/env bash

set -euo pipefail

VERSION=${VERSION:-}

version_package="github.com/akuityio/k8sta/internal/common/version"

for os in $OSES; do
  for arch in $ARCHS; do 
    echo "building $os-$arch"
    GOOS=$os GOARCH=$arch CGO_ENABLED=0 \
      go build \
      -o ./bin/bookkeeper-$os-$arch \
      -ldflags "-w -X ${version_package}.version=${VERSION} -X ${version_package}.buildDate=$(date -u +'%Y-%m-%dT%H:%M:%SZ')" \
      ./cmd/bookkeeper-cli
  done
  if [ $os = 'windows' ]; then
    mv ./bin/bookkeeper-$os-$arch ./bin/bookkeeper-$os-$arch.exe
  fi
done
