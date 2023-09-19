set -e

# npm install -g @bitnami/readme-generator-for-helm
readme-generator --values "${PWD}/charts/kargo/values.yaml" --readme "${PWD}/charts/kargo/README.md" --config "${PWD}/hack/helm-docs/readme-generator-config.json"