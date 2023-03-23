trigger_mode(TRIGGER_MODE_MANUAL)

load('ext://namespace', 'namespace_create')
namespace_create('kargo')
k8s_resource(
  new_name = 'namespace',
  objects = ['kargo:namespace'],
  labels = ['kargo']
)

docker_build(
  'ghcr.io/akuityio/kargo',
  '.',
  only = [
    'api/',
    'cmd/',
    'internal/',
    'go.mod',
    'go.sum'
  ],
  ignore = ['**/*_test.go']
)
k8s_resource(
  workload = 'kargo-api',
  new_name = 'api',
  port_forwards = [
    '30081:50051',
    '30082:8080'
  ],
  labels = ['kargo']
)
k8s_resource(
  workload = 'api',
  objects = [
    'kargo-api:clusterrole',
    'kargo-api:clusterrolebinding',
    'kargo-api:serviceaccount'
  ]
)
k8s_resource(
  workload = 'kargo-controller',
  new_name = 'controller',
  labels = ['kargo']
)
k8s_resource(
  workload = 'controller',
  objects = [
    'kargo:mutatingwebhookconfiguration',
    'kargo:validatingwebhookconfiguration',
    'kargo-controller:clusterrole',
    'kargo-controller:clusterrolebinding',
    'kargo-controller:serviceaccount',
    'kargo-webhook-server-cert:secret'
  ]
)
k8s_resource(
  new_name = 'crds',
  objects = [
    'environments.kargo.akuity.io:customresourcedefinition',
    'promotions.kargo.akuity.io:customresourcedefinition'
  ],
  labels = ['kargo']
)
k8s_resource(
  new_name = 'image-pull-secret',
  objects = ['kargo-image-pull-secret:secret'],
  labels = ['kargo']
)

k8s_yaml(
  helm(
    './charts/kargo',
    name = 'kargo',
    namespace = 'kargo',
    set = [
      'api.logLevel=DEBUG',
      'api-proxy.logLevel=DEBUG',
      'controller.logLevel=DEBUG'
    ]
  )
)
