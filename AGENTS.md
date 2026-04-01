# Kargo

Kargo is a Kubernetes-native continuous promotion platform for GitOps workflows.
Warehouses watch for new artifacts (container images, Git commits, Helm charts),
bundle them together as Freight, and promote that as a unit through a pipeline
of Stages.

Agent should reference `docs/docs/` if a more comprehensive understanding of the
domain is needed.

## Project Layout

| Path | Description |
|------|-------------|
| `api/v1alpha1/` | CRD types (Warehouse, Freight, Stage, Promotion, Project, etc.) and protobuf specs. Separate Go module |
| `cmd/controlplane/` | Back end entry point -- single binary with subcommands: `api`, `controller`, `management-controller`, `kubernetes-webhooks`, `external-webhooks`, `garbage-collector` |
| `cmd/cli/` | CLI entry point |
| `pkg/server/` | API server handlers. Two coexisting APIs: **ConnectRPC** (DEPRECATED, removal in v1.12.0; still used by UI -- avoid investing in fixes or enhancements) and **REST API** (the replacement; used by CLI; UI has not yet migrated) |
| `pkg/cli/` | CLI -- Cobra-based, subcommands in `pkg/cli/cmd/`, REST API client in `pkg/cli/client/` |
| `pkg/controller/` | Kubernetes controllers for Kargo resources |
| `pkg/promotion/` | Promotion engine and step runner |
| `pkg/promotion/runner/builtin/` | Built-in promotion steps (git, helm, kustomize, etc.) |
| `pkg/gitprovider/` | Git provider integrations (GitHub, GitLab, Gitea, BitBucket) |
| `pkg/image/` | Container image registry operations |
| `pkg/credentials/` | Secrets/credentials management |
| `pkg/webhook/` | Webhook handlers |
| `ui/` | React/TypeScript frontend (Vite + Ant Design + TanStack Query) |
| `charts/kargo/` | Helm chart -- primary installation method |

## Build & Development

### Prerequisites

Make, Go, Node.js, pnpm, and Docker are the primary prerequisites. Appropriate
versions of most other tools (golangci-lint, buf, controller-gen, swag, etc.)
are installed automatically in `hack/bin/` by Make targets.

### Common commands

```bash
make lint-go              # Lint Go code (golangci-lint)
make lint                 # Lint everything (Go, proto, charts, UI)
make format-go            # Auto-format Go code
make test-unit            # Run unit tests (with -race)
make build-cli            # Build CLI binary
make codegen              # Run all code generation
make hack-build-dev-tools # Build dev container with all tools
```

Containerized equivalents (no local tool installs needed):

```bash
make hack-lint-go
make hack-test-unit
make hack-codegen
```

Build the Kargo image:

```bash
make hack-build           # Build container image (kargo:dev)
```

There is seldom a need to do so directly.

### Local development with Tilt

```bash
make hack-kind-up          # Create local K8s cluster (or hack-k3d-up)
make hack-tilt-up          # Start local dev environment
```

`hack-kind-up` / `hack-k3d-up` are not needed if using OrbStack or Docker
Desktop's built-in Kubernetes clusters.

Tools like `tilt`, `ctlptl`, `kind`, `k3d`, and `helm` are auto-installed into
`hack/bin/` by these targets -- no manual installation needed. Tilt also handles
installing prerequisites (cert-manager, Argo CD, Argo Rollouts) idempotently.

- Tilt compiles back end on source changes
- **Manual trigger mode** used for re-deploying to the cluster -- trigger
  re-deployment from the Tilt UI (http://localhost:10350) or `hack/bin/tilt
  trigger <component>`
- API: localhost:30081, UI: localhost:30082, External webhooks: localhost:30083
- Argo CD: localhost:30080 (admin/admin)
- Kargo admin password: `admin`
- `make hack-tilt-down` to undeploy Kargo (preserves prerequisites)
- `make hack-kind-down` / `make hack-k3d-down` to destroy the cluster entirely

### Code generation

Run `make codegen` (or `make hack-codegen`) after modifying:

- API types in `api/v1alpha1/`
- Protobuf definitions
- Swagger annotations in `pkg/server/`
- JSON schemas for promotion step configs

This generates: protobuf bindings, CRDs, deepcopy methods, OpenAPI specs,
TypeScript API clients, and Go client code.

## Code Conventions

### Principles

- Clear over clever
- Simple over complex -- don't over-engineer or prematurely optimize
- Break large problems into small, well-defined pieces
- Structure code for testability
- Always include unit tests for new and modified code
- Never disable or skip a failing test -- fix the underlying problem

### Go style

- **Line length**: soft limit 80, hard limit 120. Don't sacrifice readability
  to hit 80 -- a few characters over is fine. Use `nolint: lll` when exceeding
  120 is truly unavoidable
- **Errors**: stdlib `errors` package only; never `github.com/pkg/errors`
- **Error handling**: always handle errors (except fmt.Print variants)
- **Naked returns**: forbidden
- **Exports**: unexported by default. Find a reason to export, not a reason
  not to
- **Package-level variables**: minimize; prefer passing dependencies explicitly,
  i.e. dependency injection
- **Variable shadowing**: forbidden (enforced by govet)
- **Import order** (enforced by gci): stdlib, third-party,
  `github.com/akuity`, dot, blank
- **var-naming**: linter rule disabled due to protobuf naming conflicts, but
  still follow Go conventions (`ID`, `URL`, `HTTP`) except where
  protobuf-generated code makes this impossible
- **Constants over literals**: extract repeated literals into named constants.
  In tests, inline literals are fine when extracting them hurts readability
- **Generated files**: `*_types.go` and `groupversion_info.go` are excluded
  from linting

### YAML style

Avoid the extra indent for list items:

```yaml
# Good
items:
- name: foo
  value: bar

# Avoid
items:
  - name: foo
    value: bar
```

### Readability

Write for a small viewport -- assume reviewers may read on a phone. These
guidelines can be bent when strict adherence hurts more than it helps.

**Single-field structs on one line:**

```go
// Good
foo := MyStruct{Name: "bar"}
items := []Item{{Name: "only-one"}}

// Avoid
foo := MyStruct{
    Name: "bar",
}
```

**One argument per line when breaking across lines.** Applies to definitions
and invocations:

```go
// Good
func NewServer(
    addr string,
    handler http.Handler,
    logger *slog.Logger,
) *Server {

// Avoid
func NewServer(addr string,
    handler http.Handler, logger *slog.Logger,
) *Server {
```

Related arguments may share a line (e.g. key/value pairs in structured logging):

```go
logger.Info(
    "promotion completed",
    "stage", stage.Name,
    "freight", freight.Name,
)
```

**Closing delimiters align with the opening statement:**

```go
// Good
results, err := client.Query(
    ctx,
    query,
    args,
)

// Avoid
results, err := client.Query(
    ctx,
    query,
    args)
```

**Group delimiters for single-element composites; separate for multiple:**

```go
// Single element -- grouped
items := []Item{{
    Name:  "only-one",
    Value: value,
}}

// Multiple elements -- separate
items := []Item{
    {
        Name:  "first",
        Value: val1,
    },
    {
        Name:  "second",
        Value: val2,
    },
}
```

### Testing

- **Framework**: testify (prefer `require` over `assert`)
- **Pattern**: table-driven tests with `t.Run()` subtests. Each case typically
  includes a `name` for identification and an `assert func(*testing.T, ...)`
  field for flexible outcome verification. White-box testing is fine -- order
  cases to exercise logical paths through the code under test from top to bottom
- **Parallelism**: `t.Parallel()` where possible
- **Mocking**: manual fake implementations via interfaces; no mock frameworks.
  Kubernetes tests use controller-runtime's `fake.NewClientBuilder().Build()`
- **Test location**: same package as the code under test

Example -- table-driven test with per-case assertions:

```go
func TestGetAuthorizedClient(t *testing.T) {
    testInternalClient := fake.NewClientBuilder().Build()
    testCases := []struct {
        name     string
        userInfo *user.Info
        assert   func(*testing.T, libClient.Client, error)
    }{
        {
            name: "no context-bound user.Info",
            assert: func(t *testing.T, _ libClient.Client, err error) {
                require.Error(t, err)
                require.Equal(t, "not allowed", err.Error())
            },
        },
        {
            name: "admin user",
            userInfo: &user.Info{IsAdmin: true},
            assert: func(t *testing.T, c libClient.Client, err error) {
                require.NoError(t, err)
                require.Same(t, testInternalClient, c)
            },
        },
    }
    for _, testCase := range testCases {
        t.Run(testCase.name, func(t *testing.T) {
            ctx := context.Background()
            if testCase.userInfo != nil {
                ctx = user.ContextWithInfo(ctx, *testCase.userInfo)
            }
            client, err := getAuthorizedClient(nil)(
                ctx, testInternalClient, "",
                schema.GroupVersionResource{}, "",
                libClient.ObjectKey{},
            )
            testCase.assert(t, client, err)
        })
    }
}
```

### Common Patterns

#### Constructors

Use an exported `New*` function returning an exported interface. Keep the
implementing struct unexported so callers depend on behavior, not
implementation:

```go
// credentials/database.go -- exported interface
type Database interface {
    Get(ctx context.Context, namespace string, credType Type, repo string) (*Credentials, error)
}

// credentials/kubernetes/database.go -- unexported implementation
type database struct {
    controlPlaneClient client.Client
    // ...
}

// credentials/kubernetes/database.go -- constructor returns interface
func NewDatabase(
    controlPlaneClient client.Client,
    localClusterClient client.Client,
    credentialProvidersRegistry credentials.ProviderRegistry,
    cfg DatabaseConfig,
) credentials.Database {
    return &database{
        controlPlaneClient:          controlPlaneClient,
        localClusterClient:          localClusterClient,
        credentialProvidersRegistry: credentialProvidersRegistry,
        cfg:                         cfg,
    }
}
```

#### Component registries

Self-registering component registries backed by `pkg/component`.
Implementations register in `init()`; the right one is selected at runtime by
name or predicate.

**Two flavors:**

- **Name-based** (`component.NameBasedRegistry`) -- O(1) lookup by string key
  (e.g. promotion steps)
- **Predicate-based** (`component.PredicateBasedRegistry`) -- sequential
  predicate evaluation until first match (e.g. webhook receivers, credential
  providers)

**Name-based example** -- promotion step runners:

```go
// pkg/promotion/runner/builtin/file_copier.go
func init() {
    promotion.DefaultStepRunnerRegistry.MustRegister(
        promotion.StepRunnerRegistration{
            Name:  stepKindCopy,
            Value: newFileCopier,
        },
    )
}
```

**Predicate-based example** -- webhook receivers:

```go
// pkg/webhook/external/github.go
func init() {
    defaultWebhookReceiverRegistry.MustRegister(
        webhookReceiverRegistration{
            Predicate: func(
                _ context.Context,
                cfg kargoapi.WebhookReceiverConfig,
            ) (bool, error) {
                return cfg.GitHub != nil, nil
            },
            Value: newGitHubWebhookReceiver,
        },
    )
}
```

When adding a new implementation: define it in its own file, self-register in
`init()`, and ensure the package is imported (usually via blank import at the
wiring location).

#### Environment-based configuration

Components define a companion config struct with `envconfig` tags and an
exported `*ConfigFromEnv()` function:

```go
// pkg/garbage/collector.go
type CollectorConfig struct {
    NumWorkers          int           `envconfig:"NUM_WORKERS" default:"3"`
    MaxRetainedFreight  int           `envconfig:"MAX_RETAINED_FREIGHT" default:"20"`
    MinFreightDeletionAge time.Duration `envconfig:"MIN_FREIGHT_DELETION_AGE" default:"336h"`
}

func CollectorConfigFromEnv() CollectorConfig {
    cfg := CollectorConfig{}
    envconfig.MustProcess("", &cfg)
    return cfg
}
```

#### Context propagation

Typed values stored in `context.Context` using unexported key types to prevent
collisions:

```go
// pkg/server/user/user.go
type userInfoKey struct{}

func ContextWithInfo(ctx context.Context, u Info) context.Context {
    return context.WithValue(ctx, userInfoKey{}, u)
}

func InfoFromContext(ctx context.Context) (Info, bool) {
    val := ctx.Value(userInfoKey{})
    if val == nil {
        return Info{}, false
    }
    u, ok := val.(Info)
    return u, ok
}
```

#### Error wrapping

Always wrap with `fmt.Errorf` and `%w`. Messages are lowercase (except when
starting with an exported type name), have no trailing punctuation, and the
wrapped error is always last. Two common phrasing styles:

```go
// "error <gerund>" -- most common
return fmt.Errorf("error listing projects: %w", err)
return fmt.Errorf("error getting runner for step kind %q: %w", req.Step.Kind, err)

// "failed to <infinitive>" -- also common
return fmt.Errorf("failed to create temporary directory: %w", err)
```

Either style is fine; be consistent within a file.

#### Logging

The `pkg/logging` package wraps `zap` with a simplified API. Loggers are stored
in context and enriched with key/value pairs as they flow through call chains:

```go
// Retrieve from context (falls back to global logger if absent)
logger := logging.LoggerFromContext(ctx)

// Enrich with request-scoped values, store back in context
logger = logger.WithValues("namespace", ns, "name", name)
ctx = logging.ContextWithLogger(ctx, logger)

// Log at various levels
logger.Trace("discovered commit", "tag", tag.Tag)   // very verbose
logger.Debug("routing webhook request")              // debugging detail
logger.Info("promotion completed", "stage", s.Name)  // normal operations
logger.Error(err, "error refreshing object")         // error takes err first
```

Levels (configured via `LOG_LEVEL` env var): `trace`, `debug`, `info`, `error`,
`discard`. Prefer `Debug` for operational detail, `Info` for state transitions,
`Error` for actionable failures. Use `Trace` sparingly for high-volume
discovery loops.

#### REST API structure

The REST API (`pkg/server/`) uses Gin. Routes are defined in
`rest_router.go` under `/v1beta1`. Gin middleware handles authentication,
error formatting, and request body limits.

Project-scoped routes live under `/v1beta1/projects/:project`. Middleware on
this group confirms the project exists before any handler runs, so individual
endpoints do not need to check this.

#### Authorization model

Kargo resources are Kubernetes-native, so authorization is largely implicit.
The API server uses an **authorizing client** (`pkg/server/kubernetes/client.go`)
that wraps the controller-runtime client and performs a `SubjectAccessReview`
before every operation. If the user lacks permission, the request fails before
any data is read or written.

**Custom verbs ("dolphin verbs"):** Kargo defines a `"promote"` verb for Stages.
Because this is not a standard Kubernetes CRUD verb, the authorizing client
cannot check it implicitly. Endpoints that require promote permission
(promote-to-stage, promote-downstream, approve-freight) must test for permission
using the wrapper's `Authorize()` method explicitly.

**Internal client bypass:** In rare cases the API server uses its own
(non-authorizing) internal client to act on behalf of a user. **Any code that
bypasses the authorizing client must be treated as security-sensitive:**
document the justification, test thoroughly, and ensure no path allows
unauthorized data access or mutation.

#### Pluggable methods (legacy -- do not introduce)

Many existing components use function-typed fields on structs to make behaviors
swappable for testing. The constructor wires each field to a real method, and
tests override specific fields:

```go
// pkg/garbage/collector.go
type collector struct {
    cfg              CollectorConfig
    cleanProjectFn   func(ctx context.Context, project string) error
    listProjectsFn   func(context.Context, client.ObjectList, ...client.ListOption) error
    deleteFreightFn  func(context.Context, client.Object, ...client.DeleteOption) error
    // ...
}

func NewCollector(kubeClient client.Client, cfg CollectorConfig) Collector {
    c := &collector{cfg: cfg}
    c.cleanProjectFn = c.cleanProject
    c.listProjectsFn = kubeClient.List
    c.deleteFreightFn = kubeClient.Delete
    return c
}
```

**This pattern is being phased out.** It is fine to continue using it (even
adding new `Fn` fields) where it already exists, but new components should
prefer interfaces and fake implementations. In most cases, controller-runtime's
`fake.NewClientBuilder().Build()` is sufficient for mocking Kubernetes
interactions.

## Documentation

The docs site lives in `docs/` and uses Docusaurus 3. Content is in
`docs/docs/` under numbered directories that control sidebar ordering. The doc
tree is organized by audience:

- **Quickstart** -- for anyone evaluating Kargo
- **Operator guide** -- for platform engineers installing and configuring Kargo
- **User guide** -- for end users promoting freight through stages
- **Contributor guide** -- for developers working on Kargo itself
- **Release notes** -- per-version changelogs

### Conventions

- **File naming**: `NN-kebab-case-name.md` -- numeric prefix controls sidebar
  order
- **Frontmatter**: use `sidebar_label` and `description` fields
- **Directory metadata**: `_category_.json` for folder labels, collapsibility,
  and generated index pages
- **Admonitions**: Docusaurus syntax (`:::note`, `:::info`, `:::caution`,
  `:::warning`). Favor `note` for things the reader should pay attention to.
  Favor `info` for supplemental information safe to skip. Use `warning` or
  `caution` sparingly -- reserve for common mistakes or insecure configurations
- **Tabs**: `<Tabs>` / `<TabItem>` for instructions that vary by OS, interface
  (dashboard vs CLI vs API), or other mutually exclusive options
- **UI elements**: use the `<hlt>` tag (e.g. `<hlt>Save</hlt>`) only for text
  that actually appears on screen -- not for abstract descriptions
- **"Freight" is a mass noun** (like "luggage"). Never pluralize as "freights"
  or use "a freight." Say "`Freight` resource" for the Kubernetes object or
  "piece of freight" for the abstract concept. Generated code may sometimes
  force incorrect pluralization -- accept it there rather than fighting the
  generator
- **Avoid "deploy"/"deployment"** when describing what Kargo does. Deploying is
  the job of a GitOps agent like Argo CD. Kargo *promotes*. Use "promote,"
  "promotion," or "progress"
- **Media assets**: keep close to the pages that reference them (same directory
  or sibling `img/` directory)
- **Refer to `docs/STYLE_GUIDE.md`** for phraseology, capitalization, and
  additional formatting conventions

### Building and previewing

```bash
make serve-docs          # Dev server on localhost:3000 (or $DOCS_PORT)
make hack-serve-docs     # Containerized version (no local Node.js needed)
```

Under the hood these run `pnpm install`, build a custom gtag plugin, then
start `docusaurus start`. To build the static site without serving:

```bash
cd docs && pnpm install && pnpm build-gtag-plugin && docusaurus build
```

## Workflow

### Definition of done

A task is complete when:

1. Code changes are implemented
2. Unit tests cover the changes
3. Linting passes (`make lint-go`, or `make lint` if non-Go files changed)
4. The project builds successfully
5. Documentation is updated if user-facing behavior changed

### Commits

All commits must include a DCO sign-off:

```plaintext
Signed-off-by: Legal Name <email@example.com>
```

Use `git commit -s` to add this automatically.

### Problem-solving

- **Read before guessing.** Understand existing code, tests, and error messages
  before proposing changes. Find similar features in the codebase and mirror
  their patterns. Grep for usage, read neighboring files, check tests
- **Stay focused.** Do what was asked -- no more, no less. If a tangential
  improvement seems valuable, mention it but don't act without approval.
  Exception: in docs already being modified for an in-scope reason, fixing
  obvious typos or markdown lint issues is welcome
- **Ask when ambiguous.** If requirements are unclear or there are multiple
  reasonable approaches, ask rather than guess. When choosing between valid
  approaches, prioritize: testability, readability, consistency with existing
  patterns, simplicity, reversibility
- **Three strikes, then ask.** If an approach fails three times, stop and
  explain what you've tried instead of continuing to iterate
- **Fix root causes.** Investigate why something fails rather than papering over
  symptoms. Don't disable linters, skip tests, or add `//nolint` without
  understanding the underlying issue
- **Minimize blast radius.** Prefer small, focused changes. If a fix touches
  many files, consider whether a simpler approach exists
