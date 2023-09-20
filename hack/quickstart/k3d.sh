#!/bin/sh

k3d cluster create kargo-quickstart \
  --no-lb \
  --k3s-arg '--disable=traefik@server:0' \
  -p '8443-8444:31443-31444@servers:0:direct' \
  -p '8081-8083:30081-30083@servers:0:direct' \
  --wait

helm install cert-manager cert-manager \
  --repo https://charts.jetstack.io \
  --version 1.11.5 \
  --namespace cert-manager \
  --create-namespace \
  --set installCRDs=true \
  --wait

helm install argocd argo-cd \
  --repo https://argoproj.github.io/argo-helm \
  --version 5.46.6 \
  --namespace argocd \
  --create-namespace \
  --set 'configs.secret.argocdServerAdminPassword=$2a$10$5vm8wXaSdbuff0m9l21JdevzXBzJFPCi8sy6OOnpZMAG.fOXL7jvO' \
  --set dex.enabled=false \
  --set notifications.enabled=false \
  --set server.service.type=NodePort \
  --set server.service.nodePortHttp=31443 \
  --wait

helm install kargo \
  oci://ghcr.io/akuity/kargo-charts/kargo \
  --namespace kargo \
  --create-namespace \
  --set api.service.type=NodePort \
  --set api.service.nodePort=31444 \
  --set 'api.adminAccount.password=admin' \
  --wait
