if 'ENABLE_NGROK_EXTENSION' in os.environ and os.environ['ENABLE_NGROK_EXTENSION'] == '1':
  v1alpha1.extension_repo(
    name = 'default',
    url = 'https://github.com/tilt-dev/tilt-extensions'
  )
  v1alpha1.extension(name = 'ngrok', repo_name = 'default', repo_path = 'ngrok')

trigger_mode(TRIGGER_MODE_MANUAL)

load('ext://namespace', 'namespace_create')
namespace_create('k8sta')
k8s_resource(
  new_name = 'namespace',
  objects = ['k8sta:namespace'],
  labels = ['k8sta']
)

docker_build(
  'ghcr.io/akuityio/k8sta-prototype',
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
  workload = 'k8sta-server',
  new_name = 'server',
  port_forwards = '30082:8080',
  labels = ['k8sta']
)
k8s_resource(
  workload = 'server',
  objects = [
    'k8sta-server:clusterrole',
    'k8sta-server:clusterrolebinding',
    'k8sta-server:serviceaccount'
  ]
)
k8s_resource(
  workload = 'k8sta-controller',
  new_name = 'controller',
  labels = ['k8sta']
)
k8s_resource(
  workload = 'controller',
  objects = [
    'k8sta-controller:clusterrole',
    'k8sta-controller:clusterrolebinding',
    'k8sta-controller:serviceaccount'
  ]
)
k8s_resource(
  new_name = 'crds',
  objects = [
    'environments.k8sta.akuity.io:customresourcedefinition'
  ],
  labels = ['k8sta']
)
k8s_resource(
  new_name = 'image-pull-secret',
  objects = ['k8sta-image-pull-secret:secret'],
  labels = ['k8sta']
)

k8s_yaml(
  helm(
    './charts/k8sta',
    name = 'k8sta',
    namespace = 'k8sta',
    set = [
      'controller.logLevel=DEBUG',
      'server.logLevel=DEBUG',
      'server.service.type=NodePort',
      'server.service.nodePort=30082',
      'server.tls.enabled=false'
    ]
  )
)
