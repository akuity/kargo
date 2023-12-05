module github.com/akuity/kargo

go 1.20

require (
	connectrpc.com/connect v1.11.0
	connectrpc.com/grpchealth v1.2.0
	github.com/AlecAivazis/survey/v2 v2.3.7
	github.com/Masterminds/semver v1.5.0
	github.com/adrg/xdg v0.4.0
	// These important changes are merged, but not yet released:
	// https://github.com/argoproj-labs/argocd-image-updater/pull/456
	// https://github.com/argoproj-labs/argocd-image-updater/pull/559
	github.com/argoproj-labs/argocd-image-updater v0.0.0-20230921174511-420bd97b57d2
	github.com/argoproj/argo-cd/v2 v2.8.6
	github.com/argoproj/gitops-engine v0.7.1-0.20230607163028-425d65e07695 // indirect
	github.com/bacongobbler/browser v1.1.0
	github.com/coreos/go-oidc/v3 v3.6.0
	github.com/evanphx/json-patch/v5 v5.6.0
	github.com/gobwas/glob v0.2.3
	github.com/golang-jwt/jwt/v5 v5.0.0
	github.com/google/uuid v1.3.1
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/oklog/ulid/v2 v2.1.0
	github.com/pkg/errors v0.9.1
	github.com/samber/mo v1.8.0
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/cobra v1.7.0
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.8.4
	golang.org/x/crypto v0.14.0
	golang.org/x/exp v0.0.0-20230807204917-050eac23e9de
	golang.org/x/net v0.17.0
	golang.org/x/oauth2 v0.13.0
	google.golang.org/grpc v1.58.3
	google.golang.org/protobuf v1.31.0
	gopkg.in/yaml.v3 v3.0.1
	k8s.io/api v1.22.4
	k8s.io/apimachinery v1.22.4
	k8s.io/cli-runtime v0.24.2
	k8s.io/client-go v11.0.1-0.20190816222228-6d55c1b1f1ca+incompatible
	k8s.io/kubectl v0.24.2
	k8s.io/utils v0.0.0-20230220204549-a5ecb0141aa5
	oras.land/oras-go v1.2.4
	sigs.k8s.io/controller-runtime v0.13.1
	sigs.k8s.io/kustomize/api v0.11.5
	sigs.k8s.io/kustomize/kyaml v0.13.7
	sigs.k8s.io/yaml v1.3.0
)

require (
	github.com/rs/cors v1.8.0
	github.com/technosophos/moniker v0.0.0-20210218184952-3ea787d3943b
	github.com/withfig/autocomplete-tools/integrations/cobra v1.2.1
)

require (
	cloud.google.com/go/compute v1.21.0 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	dario.cat/mergo v1.0.0 // indirect
	github.com/AdaLogics/go-fuzz-headers v0.0.0-20230811130428-ced1acdcaa24 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.27 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.20 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/MakeNowJust/heredoc v0.0.0-20170808103936-bb23615498cd // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.2.1 // indirect
	github.com/Masterminds/sprig/v3 v3.2.3 // indirect
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/Microsoft/hcsshim v0.11.0 // indirect
	github.com/ProtonMail/go-crypto v0.0.0-20230717121422-5aa5874ade95 // indirect
	github.com/acomagu/bufpipe v1.0.4 // indirect
	github.com/argoproj/pkg v0.13.7-0.20230627120311-a4dd357b057e // indirect
	github.com/benbjohnson/clock v1.3.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bmatcuk/doublestar/v4 v4.6.0 // indirect
	github.com/bombsimon/logrusr/v2 v2.0.1 // indirect
	github.com/bradleyfalzon/ghinstallation/v2 v2.5.0 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/chai2010/gettext-go v0.0.0-20170215093142-bf70f2a70fb1 // indirect
	github.com/cloudflare/circl v1.3.3 // indirect
	github.com/containerd/containerd v1.7.6 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/distribution/distribution/v3 v3.0.0-20230722181636-7b502560cad4 // indirect
	github.com/docker/cli v24.0.6+incompatible // indirect
	github.com/docker/distribution v2.8.2+incompatible // indirect
	github.com/docker/docker v24.0.7+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.7.0 // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/docker/libtrust v0.0.0-20160708172513-aabc10ec26b7 // indirect
	github.com/emicklei/go-restful/v3 v3.10.1 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/exponent-io/jsonpath v0.0.0-20151013193312-d6023ce2651d // indirect
	github.com/fatih/camelcase v1.0.0 // indirect
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/fvbommel/sortorder v1.0.1 // indirect
	github.com/go-errors/errors v1.4.2 // indirect
	github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 // indirect
	github.com/go-git/go-billy/v5 v5.4.1 // indirect
	github.com/go-git/go-git/v5 v5.8.1 // indirect
	github.com/go-jose/go-jose/v3 v3.0.1 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.20.0 // indirect
	github.com/go-openapi/swag v0.22.3 // indirect
	github.com/go-redis/cache/v9 v9.0.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/btree v1.1.2 // indirect
	github.com/google/gnostic v0.6.9 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/go-github/v53 v53.0.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/huandu/xstrings v1.3.3 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jonboulle/clockwork v0.2.2 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/kevinburke/ssh_config v1.2.0 // indirect
	github.com/klauspost/compress v1.16.5 // indirect
	github.com/liggitt/tabwriter v0.0.0-20181228230101-89fcab3d43de // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/mattn/go-isatty v0.0.19 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/mitchellh/copystructure v1.0.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.0 // indirect
	github.com/moby/locker v1.0.1 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/moby/term v0.0.0-20221205130635-1aeaba878587 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/monochromegane/go-gitignore v0.0.0-20200626010858-205db1a8cc00 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0-rc5 // indirect
	github.com/patrickmn/go-cache v2.1.0+incompatible // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pjbgf/sha1cd v0.3.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_golang v1.16.0 // indirect
	github.com/prometheus/client_model v0.3.0 // indirect
	github.com/prometheus/common v0.42.0 // indirect
	github.com/prometheus/procfs v0.10.1 // indirect
	github.com/redis/go-redis/v9 v9.0.5 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	github.com/russross/blackfriday v1.6.0 // indirect
	github.com/sergi/go-diff v1.1.0 // indirect
	github.com/shopspring/decimal v1.2.0 // indirect
	github.com/skeema/knownhosts v1.2.0 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	github.com/vmihailenco/go-tinylfu v0.2.2 // indirect
	github.com/vmihailenco/msgpack/v5 v5.3.4 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	github.com/xlab/treeprint v1.1.0 // indirect
	go.opentelemetry.io/otel v1.16.0 // indirect
	go.opentelemetry.io/otel/metric v1.16.0 // indirect
	go.opentelemetry.io/otel/trace v1.16.0 // indirect
	go.starlark.net v0.0.0-20220328144851-d1966c6b9fcd // indirect
	go.uber.org/goleak v1.2.0 // indirect
	go.uber.org/ratelimit v0.3.0 // indirect
	golang.org/x/mod v0.11.0 // indirect
	golang.org/x/sync v0.3.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/term v0.13.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	golang.org/x/tools v0.7.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.2.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230711160842-782d3b101e98 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/apiextensions-apiserver v0.25.0 // indirect
	k8s.io/apiserver v0.26.2 // indirect
	k8s.io/component-base v0.26.2 // indirect
	k8s.io/component-helpers v0.24.2 // indirect
	k8s.io/klog/v2 v2.90.1 // indirect
	k8s.io/kube-aggregator v0.24.2 // indirect
	k8s.io/kube-openapi v0.0.0-20220803162953-67bda5d908f1 // indirect
	k8s.io/kubernetes v1.24.15 // indirect
	oras.land/oras-go/v2 v2.2.0 // indirect
	sigs.k8s.io/json v0.0.0-20220713155537-f223a00ba0e2 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.3 // indirect
)

replace (
	// Avoid CVE-2022-28948
	gopkg.in/yaml.v3 => gopkg.in/yaml.v3 v3.0.1

	// https://github.com/kubernetes/kubernetes/issues/79384#issuecomment-505627280
	k8s.io/api => k8s.io/api v0.24.2
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.24.2
	k8s.io/apimachinery => k8s.io/apimachinery v0.24.2
	k8s.io/apiserver => k8s.io/apiserver v0.24.2
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.24.2
	k8s.io/client-go => k8s.io/client-go v0.24.2
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.24.2
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.24.2
	k8s.io/code-generator => k8s.io/code-generator v0.24.2
	k8s.io/component-base => k8s.io/component-base v0.24.2
	k8s.io/component-helpers => k8s.io/component-helpers v0.24.2
	k8s.io/controller-manager => k8s.io/controller-manager v0.24.2
	k8s.io/cri-api => k8s.io/cri-api v0.24.2
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.24.2
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.24.2
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.24.2
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.24.2
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.24.2
	k8s.io/kubectl => k8s.io/kubectl v0.24.2
	k8s.io/kubelet => k8s.io/kubelet v0.24.2
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.24.2
	k8s.io/metrics => k8s.io/metrics v0.24.2
	k8s.io/mount-utils => k8s.io/mount-utils v0.24.2
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.24.2
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.24.2
)
