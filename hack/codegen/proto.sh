#!/usr/bin/env bash

set -euxo pipefail

readonly APIMACHINERY_PKGS=k8s.io/apimachinery/pkg/util/intstr,+k8s.io/apimachinery/pkg/api/resource,+k8s.io/apimachinery/pkg/runtime/schema,+k8s.io/apimachinery/pkg/runtime,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/api/core/v1,k8s.io/api/batch/v1

work_dir=$(dirname "${0}")
work_dir=$(readlink -f "${work_dir}/../..")

cd "${work_dir}"

function clean() {
  echo "Clean up intermediate resources..."
  rm -r "${work_dir}/pkg/api/v1alpha1"
}

function main() {
  echo "Vendor dependencies for protobuf code generation..."
  go mod tidy
  go mod vendor

  echo "Generate protobuf code from Kubebuilder structs..."
  GOPATH=${GOPATH} go-to-protobuf \
    --go-header-file=./hack/boilerplate.go.txt \
    --packages=github.com/akuity/kargo/api/v1alpha1 \
    --apimachinery-packages="${APIMACHINERY_PKGS}" \
    --proto-import "${work_dir}/vendor"

  echo "Copy generated code to the working directory..."
  cp -R "${GOPATH}/src/github.com/akuity/kargo/api" "${work_dir}"

  echo "Generate protobuf code (Go)..."
  buf generate api

  echo "Inject generated protobuf tag to Kubebuilder structs..."
  go run ./hack/codegen/prototag/main.go \
    -src-dir="${work_dir}/pkg/api/v1alpha1" \
    -dst-dir="${work_dir}/api/v1alpha1"

  echo "Generate protobuf code (UI)..."
  buf generate api \
    --include-imports \
    --template=buf.ui.gen.yaml
  pnpm --dir=ui install --dev
  pnpm run --dir=ui generate:proto-extensions
}

trap 'clean' EXIT

main
