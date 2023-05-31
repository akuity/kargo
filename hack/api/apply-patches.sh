#!/usr/bin/env bash

set -euxo pipefail

# shellcheck disable=SC2128
PROJECT_ROOT=$(cd "$(dirname "${BASH_SOURCE}")"/../..; pwd)

for patch in "${PROJECT_ROOT}/hack/api/patches/"*.patch;
do
  git apply --ignore-whitespace "${patch}"
done
