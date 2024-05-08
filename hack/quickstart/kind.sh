#!/bin/sh

set -x

kind create cluster \
  --wait 120s \
  --config - <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: kargo-quickstart
nodes:
- extraPortMappings:
  - containerPort: 31443 # Argo CD dashboard
    hostPort: 31443
  - containerPort: 31444 # Kargo dashboard
    hostPort: 31444
  - containerPort: 30081 # test application instance
    hostPort: 30081
  - containerPort: 30082 # UAT application instance
    hostPort: 30082
  - containerPort: 30083 # prod application instance
    hostPort: 30083
  
EOF

helm install cert-manager cert-manager \
  --repo https://charts.jetstack.io \
  --version 1.11.5 \
  --namespace cert-manager \
  --create-namespace \
  --set installCRDs=true \
  --wait

helm install argocd argo-cd \
  --repo https://argoproj.github.io/argo-helm \
  --version 5.51.6 \
  --namespace argocd \
  --create-namespace \
  --set 'configs.secret.argocdServerAdminPassword=$2a$10$5vm8wXaSdbuff0m9l21JdevzXBzJFPCi8sy6OOnpZMAG.fOXL7jvO' \
  --set dex.enabled=false \
  --set notifications.enabled=false \
  --set server.service.type=NodePort \
  --set server.service.nodePortHttp=31443 \
  --set server.extensions.enabled=true \
  --set 'server.extensions.contents[0].name=argo-rollouts' \
  --set 'server.extensions.contents[0].url=https://github.com/argoproj-labs/rollout-extension/releases/download/v0.3.3/extension.tar' \
  --wait

helm install argo-rollouts argo-rollouts \
  --repo https://argoproj.github.io/argo-helm \
  --version 2.33.0 \
  --create-namespace \
  --namespace argo-rollouts \
  --wait

# Password is 'admin'
helm install kargo \
  oci://ghcr.io/akuity/kargo-charts/kargo \
  --version 0.6.0-rc.2 \
  --namespace kargo \
  --create-namespace \
  --set api.service.type=NodePort \
  --set api.service.nodePort=31444 \
  --set api.adminAccount.passwordHash='$2a$10$Zrhhie4vLz5ygtVSaif6o.qN36jgs6vjtMBdM6yrU1FOeiAAMMxOm' \
  --set api.adminAccount.tokenSigningKey=iwishtowashmyirishwristwatch \
  --wait
