trigger_mode(TRIGGER_MODE_MANUAL)

load('ext://namespace', 'namespace_create')
namespace_create('kargo')
k8s_resource(
  new_name = 'namespace',
  objects = ['kargo:namespace'],
  labels = ['kargo']
)

docker_build(
  'ghcr.io/akuity/kargo',
  '.',
  only = [
    'api/',
    'cmd/',
    'internal/',
    'pkg/',
    'go.mod',
    'go.sum'
  ],
  ignore = ['**/*_test.go'],
  build_args = {
    'GIT_COMMIT': local('git rev-parse HEAD'),
    'GIT_TREE_STATE': local('if [ -z "`git status --porcelain`" ]; then echo "clean" ; else echo "dirty"; fi')
  }
)

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
    'kargo-api:serviceaccount'
  ]
)

k8s_resource(
  workload = 'kargo-controller',
  new_name = 'controller',
  labels = ['kargo'],
  objects = [
    'kargo-controller:clusterrole',
    'kargo-controller:clusterrolebinding',
    'kargo-controller:role',
    'kargo-controller:rolebinding',
    'kargo-controller:serviceaccount',
    'kargo-controller-argocd:clusterrole',
    'kargo-controller-argocd:clusterrolebinding'
  ]
)

k8s_resource(
  workload = 'kargo-dex-server',
  new_name = 'dex-server',
  labels = ['kargo'],
  objects = [
    'kargo-dex-server:serviceaccount',
    'kargo-dex-server:configmap',
    'kargo-dex-server:certificate'
  ]
)

k8s_resource(
  workload = 'kargo-garbage-collector',
  new_name = 'garbage-collector',
  labels = ['kargo'],
  objects = [
    'kargo-garbage-collector:clusterrole',
    'kargo-garbage-collector:clusterrolebinding',
    'kargo-garbage-collector:serviceaccount'
  ]
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
    'kargo-webhooks-server:serviceaccount',
    'kargo-webhooks-server-ns-controller:clusterrole',
    'kargo-webhooks-server-ns-controller:clusterrolebinding'
  ]
)

k8s_resource(
  new_name = 'crds',
  objects = [
    'stages.kargo.akuity.io:customresourcedefinition',
    'promotionpolicies.kargo.akuity.io:customresourcedefinition',
    'promotions.kargo.akuity.io:customresourcedefinition'
  ],
  labels = ['kargo']
)

k8s_yaml(
  helm(
    './charts/kargo',
    name = 'kargo',
    namespace = 'kargo',
    set = [
      'api.logLevel=DEBUG',
      'api.address=http://localhost:30081',
      'api.oidc.enabled=true',
      'api.oidc.dex.enabled=true',
      'api-proxy.logLevel=DEBUG',
      'controller.logLevel=DEBUG'
    ]
  )
)
