#!/usr/bin/env sh

set -o errexit
set -o nounset

# Check for required tools
for tool in helm jq kubectl; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "Error: $tool is not installed" >&2
    exit 1
  fi
done

# Parse options
release=""
namespace=""
auto_detect=0

while getopts ":r:n:a" opt; do
  case $opt in
    r)
      release="$OPTARG"
      ;;
    n)
      namespace="$OPTARG"
      ;;
    a)
      auto_detect=1
      ;;
    \?)
      echo "Error: Invalid option -$OPTARG" >&2
      exit 1
      ;;
  esac
done
shift $((OPTIND - 1))

# Auto detect release and namespace if requested
if [ "$auto_detect" -eq 1 ]; then
  release_info=$(helm list -o json --all-namespaces | jq '.[] | select(.chart | contains("kargo-"))')
  if [ -z "$release_info" ]; then
    echo "Error: No release with kargo chart found" >&2
    exit 1
  fi

  release=$(echo "$release_info" | jq -r '.name')
  namespace=$(echo "$release_info" | jq -r '.namespace')
fi

# Validate required options
if [ -z "$release" ] || [ -z "$namespace" ]; then
  echo "Error: -r (release) and -n (namespace), or -a (auto-detect) must be provided" >&2
  exit 1
fi

# Adopt CustomResourceDefinitions
echo "Starting adoption of Kargo CustomResourceDefinitions as release $release in namespace $namespace"

for name in $(kubectl get crds -o name | grep kargo.akuity.io); do
  echo "Adopting $name"
  kubectl annotate --overwrite "$name" meta.helm.sh/release-name="$release"
  kubectl annotate --overwrite "$name" meta.helm.sh/release-namespace="$namespace"
  kubectl label --overwrite "$name" app.kubernetes.io/managed-by=Helm
done

echo "Adoption complete!"
