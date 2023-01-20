if 'ENABLE_NGROK_EXTENSION' in os.environ and os.environ['ENABLE_NGROK_EXTENSION'] == '1':
  v1alpha1.extension_repo(
    name = 'default',
    url = 'https://github.com/tilt-dev/tilt-extensions'
  )
  v1alpha1.extension(name = 'ngrok', repo_name = 'default', repo_path = 'ngrok')

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
  workload = 'kargo-server',
  new_name = 'server',
  port_forwards = '30082:8080',
  labels = ['kargo']
)
k8s_resource(
  workload = 'server',
  objects = [
    'kargo-server:clusterrole',
    'kargo-server:clusterrolebinding',
    'kargo-server:serviceaccount'
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
      'controller.logLevel=DEBUG',
      'server.logLevel=DEBUG',
      'server.service.type=NodePort',
      'server.service.nodePort=30082',
      'server.tls.enabled=false'
    ]
  )
)
