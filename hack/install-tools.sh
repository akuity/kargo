#!/usr/bin/env bash

#######################################
# This script installs Go programs which defined in hack/tools.go.
#######################################

set -euo pipefail

#######################################
# Get imports in a given Go source file
# Arguments:
#   A path to the Go source code.
# Outputs:
#  Writes space-separated imports to stdout.
#######################################
function get_imports() {
  local src="$1"

  # Listing imports in the given source file
  local imports
  imports=$(go list -e -f '{{.Imports}}' "${src}")

  # Trim brackets from the output
  imports="${imports/[/}"
  imports="${imports/]/}"

  echo "${imports}"
}

#######################################
# Get module version which specified in go.mod of the given package.
# Arguments:
#   A name of the package.
# Outputs:
#   Writes version of the given package to stdout.
#######################################
function get_module_version() {
  local package="$1"

  # Output of `go list -f '{{.Module}}' "${package}"` is formatted in "<package> <version>",
  # so treat output as an array and print the second element - version.
  IFS=" " read -r -a module <<< "$(go list -f '{{.Module}}' "${package}")"
  echo "${module[1]}"
}

function main() {
  local workdir
  workdir=$(dirname "$0")

  local programs
  programs=$(get_imports "${workdir}/tools.go")

  for program in ${programs}; do
    version=$(get_module_version "${program}")

    echo "Installing ${program}@${version}..."
    go install "${program}@${version}"
  done
}

main "$@"
