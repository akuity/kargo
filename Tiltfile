trigger_mode(TRIGGER_MODE_MANUAL)
allow_k8s_contexts('orbstack')

load('ext://namespace', 'namespace_create')

local_resource(
  'back-end-compile',
  'CGO_ENABLED=0 GOOS=linux GOARCH=$(go env GOARCH) go build -o bin/controlplane/kargo ./cmd/controlplane',
  deps=[
    'api/',
    'cmd/controlplane/',
    'internal/',
    'pkg/',
    'go.mod',
    'go.sum'
  ],
  labels = ['native-processes'],
  trigger_mode = TRIGGER_MODE_AUTO
)
local_resource(
  'credential-helper-compile',
  'CGO_ENABLED=0 GOOS=linux GOARCH=$(go env GOARCH) go build -o bin/credential-helper ./cmd/credential-helper',
  deps=['cmd/credential-helper/'],
  labels = ['native-processes'],
  trigger_mode = TRIGGER_MODE_AUTO
)
docker_build(
  'ghcr.io/akuity/kargo',
  '.',
  only = [
    'bin/controlplane/kargo',
    'bin/credential-helper'
  ],
  target = 'back-end-dev', # Just the back end, built natively, copied to the image
)

docker_build(
  'kargo-ui',
  '.',
  only = ['ui/'],
  target = 'ui-dev', # Just the font end, served by vite, live updated
  live_update = [sync('ui', '/ui')]
)

namespace_create('kargo')
k8s_resource(
  new_name = 'namespace',
  objects = ['kargo:namespace'],
  labels = ['kargo']
)

k8s_yaml(
  helm(
    './charts/kargo',
    name = 'kargo',
    namespace = 'kargo',
    values = 'hack/tilt/values.dev.yaml'
  )
)
# Normally the API server serves up the front end, but we want live updates
# of the UI, so we're breaking it out into its own separate deployment here.
k8s_yaml('hack/tilt/ui.yaml')

k8s_resource(
  new_name = 'common',
  labels = ['kargo'],
  objects = [
    'kargo-admin:clusterrole',
    'kargo-admin:clusterrolebinding',
    'kargo-admin:serviceaccount',
    'kargo-viewer:clusterrole',
    'kargo-viewer:serviceaccount',
    'kargo-viewer:clusterrolebinding',
    'kargo-selfsigned-cert-issuer:issuer'
  ]
)

k8s_resource(
  workload = 'kargo-api',
  new_name = 'api',
  port_forwards = [
    '30081:8080'
  ],
  labels = ['kargo'],
  objects = [
    'kargo-api:clusterrole',
    'kargo-api:clusterrolebinding',
    'kargo-api:configmap',
    'kargo-api:secret',
    'kargo-api:serviceaccount',
    'kargo-api-rollouts:clusterrole',
    'kargo-api-rollouts:clusterrolebinding',
    'kargo-project-admin:clusterrole'
  ],
  resource_deps=['back-end-compile','dex-server']
)

k8s_resource(
  workload = 'kargo-controller',
  new_name = 'controller',
  labels = ['kargo'],
  objects = [
    'kargo-controller:clusterrole',
    'kargo-controller:clusterrolebinding',
    'kargo-controller:configmap',
    'kargo-controller:serviceaccount',
    'kargo-controller-argocd:clusterrole',
    'kargo-controller-argocd:clusterrolebinding',
    'kargo-controller-read-secrets:clusterrole',
    'kargo-controller-rollouts:clusterrole',
    'kargo-controller-rollouts:clusterrolebinding'
  ],
  resource_deps=['back-end-compile', 'credential-helper-compile', ]
)

k8s_resource(
  workload = 'kargo-dex-server',
  new_name = 'dex-server',
  labels = ['kargo'],
  objects = [
    'kargo-dex-server:certificate',
    'kargo-dex-server:secret',
    'kargo-dex-server:serviceaccount'
  ]
)

k8s_resource(
  workload = 'kargo-external-webhooks-server',
  new_name = 'external-webhooks-server',
  port_forwards = [
    '30083:8080'
  ],
  labels = ['kargo'],
  objects = [
    'kargo-external-webhooks-server:clusterrole',
    'kargo-external-webhooks-server:clusterrolebinding',
    'kargo-external-webhooks-server:configmap',
    'kargo-external-webhooks-server:serviceaccount'
  ],
)

k8s_resource(
  workload = 'kargo-garbage-collector',
  new_name = 'garbage-collector',
  labels = ['kargo'],
  objects = [
    'kargo-garbage-collector:clusterrole',
    'kargo-garbage-collector:clusterrolebinding',
    'kargo-garbage-collector:configmap',
    'kargo-garbage-collector:serviceaccount'
  ],
  resource_deps=['back-end-compile']
)

k8s_resource(
  workload = 'kargo-management-controller',
  new_name = 'management-controller',
  labels = ['kargo'],
  objects = [
    'kargo-management-controller:clusterrole',
    'kargo-management-controller:clusterrolebinding',
    'kargo-management-controller:configmap',
    'kargo-management-controller:serviceaccount'
  ],
  resource_deps=['back-end-compile']
)

k8s_resource(
  workload = 'kargo-ui',
  new_name = 'ui',
  port_forwards = [
    '30082:3333'
  ],
  labels = ['kargo'],
  trigger_mode = TRIGGER_MODE_AUTO
)

k8s_resource(
  workload = 'kargo-webhooks-server',
  new_name = 'kubernetes-webhooks-server',
  labels = ['kargo'],
  objects = [
    'kargo:mutatingwebhookconfiguration',
    'kargo:validatingwebhookconfiguration',
    'kargo-webhooks-server:certificate',
    'kargo-webhooks-server:clusterrole',
    'kargo-webhooks-server:clusterrolebinding',
    'kargo-webhooks-server:configmap',
    'kargo-webhooks-server-generic-gc:clusterrole',
    'kargo-webhooks-server-generic-gc:clusterrolebinding',
    'kargo-webhooks-server:serviceaccount',
    'kargo-webhooks-server-ns-controller:clusterrole',
    'kargo-webhooks-server-ns-controller:clusterrolebinding'
  ],
  resource_deps=['back-end-compile']
)

k8s_resource(
  new_name = 'crds',
  objects = [
    'clusterpromotiontasks.kargo.akuity.io:customresourcedefinition',
    'freights.kargo.akuity.io:customresourcedefinition',
    'projectconfigs.kargo.akuity.io:customresourcedefinition',
    'projects.kargo.akuity.io:customresourcedefinition',
    'promotions.kargo.akuity.io:customresourcedefinition',
    'promotiontasks.kargo.akuity.io:customresourcedefinition',
    'stages.kargo.akuity.io:customresourcedefinition',
    'warehouses.kargo.akuity.io:customresourcedefinition'
  ],
  labels = ['kargo']
)
