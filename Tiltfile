trigger_mode(TRIGGER_MODE_MANUAL)
allow_k8s_contexts('orbstack')

load('ext://namespace', 'namespace_create')

local_resource(
  'back-end-compile',
  'CGO_ENABLED=0 GOOS=linux GOARCH=$(go env GOARCH) go build -o bin/controlplane/kargo ./cmd/controlplane',
  deps=[
    'api/',
    'cmd/',
    'internal/',
    'pkg/',
    'go.mod',
    'go.sum'
  ],
  labels = ['native-processes'],
  trigger_mode = TRIGGER_MODE_AUTO
)
docker_build(
  'ghcr.io/akuity/kargo',
  '.',
  only = ['bin/controlplane/kargo'],
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
    'kargo-developer:clusterrole',
    'kargo-promoter:clusterrole',
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
    'kargo-api:serviceaccount'
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
    'kargo-controller:role',
    'kargo-controller:rolebinding',
    'kargo-controller:serviceaccount',
    'kargo-controller-argocd:clusterrole',
    'kargo-controller-argocd:clusterrolebinding'
  ],
  resource_deps=['back-end-compile']
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
  new_name = 'webhooks-server',
  labels = ['kargo'],
  objects = [
    'kargo:mutatingwebhookconfiguration',
    'kargo:validatingwebhookconfiguration',
    'kargo-webhooks-server:certificate',
    'kargo-webhooks-server:clusterrole',
    'kargo-webhooks-server:clusterrolebinding',
    'kargo-webhooks-server:configmap',
    'kargo-webhooks-server:serviceaccount',
    'kargo-webhooks-server-ns-controller:clusterrole',
    'kargo-webhooks-server-ns-controller:clusterrolebinding'
  ],
  resource_deps=['back-end-compile']
)

k8s_resource(
  new_name = 'crds',
  objects = [
    'freights.kargo.akuity.io:customresourcedefinition',
    'stages.kargo.akuity.io:customresourcedefinition',
    'promotionpolicies.kargo.akuity.io:customresourcedefinition',
    'promotions.kargo.akuity.io:customresourcedefinition',
    'warehouses.kargo.akuity.io:customresourcedefinition'
  ],
  labels = ['kargo']
)
