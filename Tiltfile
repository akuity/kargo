trigger_mode(TRIGGER_MODE_MANUAL)

load('ext://namespace', 'namespace_create')
namespace_create('kargo')
k8s_resource(
  new_name = 'namespace',
  objects = ['kargo:namespace'],
  labels = ['kargo']
)

docker_build(
  'ghcr.io/akuityio/kargo-prototype',
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
  workload = 'kargo-controller',
  new_name = 'controller',
  labels = ['kargo']
)
k8s_resource(
  workload = 'controller',
  objects = [
    'kargo-controller:clusterrole',
    'kargo-controller:clusterrolebinding',
    'kargo-controller:serviceaccount'
  ]
)
k8s_resource(
  new_name = 'crds',
  objects = [
    'environments.kargo.akuity.io:customresourcedefinition'
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
      'controller.logLevel=DEBUG'
    ]
  )
)
