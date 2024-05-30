#!/usr/bin/env bash

set -euxo pipefail

readonly API_PKGS=(
  "github.com/akuity/kargo/api/v1alpha1"
  "github.com/akuity/kargo/api/rbac/v1alpha1"
  "github.com/akuity/kargo/internal/controller/rollouts/api/v1alpha1"
)

readonly APIMACHINERY_PKGS=(
  "-k8s.io/apimachinery/pkg/util/intstr"
  "-k8s.io/apimachinery/pkg/api/resource"
  "-k8s.io/apimachinery/pkg/runtime/schema"
  "-k8s.io/apimachinery/pkg/runtime"
  "-k8s.io/apimachinery/pkg/apis/meta/v1"
  "-k8s.io/api/core/v1"
  "-k8s.io/api/batch/v1"
  "-k8s.io/api/rbac/v1"
)

work_dir=$(dirname "${0}")
work_dir=$(readlink -f "${work_dir}/../..")

build_dir=$(mktemp -d)

function clean() {
  echo "Clean up intermediate resources..."
  rm -r "${build_dir}" || true
  rm -r "${work_dir}/pkg/api/v1alpha1" || true
  rm -r "${work_dir}/pkg/api/rbac" || true
  rm -r "${work_dir}/vendor" || true
}

function main() {
  echo "Change working directory to build_dir..."
  cd "${build_dir}"

  echo "Prepare build directory..."
  # Initialize dummy module inside build directory
  go mod init github.com/akuity
  go work init

  # Copy source files to build directory
  local build_src_dir
  build_src_dir="${build_dir}/src/github.com/akuity/kargo"
  mkdir -p "${build_src_dir}"
  rsync -av \
    --prune-empty-dirs \
    --include='*/' \
    --include='*.go' \
    --include='go.mod' \
    --include='go.sum' \
    --exclude='*' \
    "${work_dir}/" \
    "${build_src_dir}"
  go work use ./src/github.com/akuity/kargo

  echo "Vendor dependencies for protobuf code generation..."
  go work sync
  go work vendor

  echo "Generate protobuf code from Kubebuilder structs..."
  go-to-protobuf \
    --go-header-file="${work_dir}/hack/boilerplate.go.txt" \
    --packages="$(IFS=, ; echo "${API_PKGS[*]}")" \
    --apimachinery-packages="$(IFS=, ; echo "${APIMACHINERY_PKGS[*]}")" \
    --proto-import="${build_dir}/vendor" \
    --output-dir="${build_dir}/src"

  echo "Copy generated code to the working directory..."
  cp -R "${build_src_dir}/api" "${work_dir}"
  cp -R "${build_src_dir}/internal" "${work_dir}"

  echo "Change working directory to work_dir..."
  cd "${work_dir}"

  echo "Vendor dependencies for protobuf code generation..."
  go mod vendor

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

  clean
}

trap 'clean' EXIT

main
