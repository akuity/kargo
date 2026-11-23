package projectconfigs

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	structuralschema "k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/schema/cel"
	structuraldefaulting "k8s.io/apiextensions-apiserver/pkg/apiserver/schema/defaulting"
	structurallisttype "k8s.io/apiextensions-apiserver/pkg/apiserver/schema/listtype"
	structuralpruning "k8s.io/apiextensions-apiserver/pkg/apiserver/schema/pruning"
	apiextensionsvalidation "k8s.io/apiextensions-apiserver/pkg/apiserver/validation"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utiljson "k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/validation/field"
	celconfig "k8s.io/apiserver/pkg/apis/cel"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/yaml"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	projectconfigwebhook "github.com/akuity/kargo/pkg/webhook/kubernetes/projectconfig"
)

const (
	parityProject  = "kargo-demo"
	parityCRDPath  = "../../../charts/kargo/resources/crds/kargo.akuity.io_projectconfigs.yaml"
	parityFixtures = "testdata/parity"
)

// TestValidateParity runs every fixture under testdata/parity through the new
// validator and through a model of today's Kubernetes behavior: the actual CRD
// schema (pruning, defaulting, OpenAPI, list-type and CEL validation) plus the
// existing admission webhook. Both must agree with the fixture's "# want:"
// header.
//
// The model of today covers create only, as the name rule and the Project
// check are create-only. The namespace in the fixture's metadata is compared
// with the request namespace, as the Kubernetes API server does, and the
// webhook always sees a namespace labeled as a Project, so the Project check
// (which the new validator replaces with middleware) never fires. Dry-run is
// not modeled because it only skips that same Project check.
func TestValidateParity(t *testing.T) {
	today := newTodayModel(t)

	paths, err := filepath.Glob(filepath.Join(parityFixtures, "*.yaml"))
	require.NoError(t, err)
	require.NotEmpty(t, paths)

	for _, path := range paths {
		t.Run(strings.TrimSuffix(filepath.Base(path), ".yaml"), func(t *testing.T) {
			raw, err := os.ReadFile(path)
			require.NoError(t, err)
			want := readWant(t, raw)

			// Decode the fixture the way the API server will: a
			// case-preserving JSON map, then Decode, then Validate.
			jsonBytes, err := yaml.YAMLToJSON(raw)
			require.NoError(t, err)
			obj := map[string]any{}
			require.NoError(t, json.Unmarshal(jsonBytes, &obj))
			cfg, errs := Decode(obj)
			if len(errs) == 0 {
				errs = Validate(t.Context(), parityProject, cfg, true)
			}
			got := len(errs) == 0

			todayAccepts := today.crdSchemaAccepts(t, raw) &&
				today.webhookAccepts(t, raw)

			t.Logf(
				"want valid=%t, new validator valid=%t, today valid=%t",
				want, got, todayAccepts,
			)
			for _, e := range errs {
				t.Logf("new validator: %s", e)
			}
			require.Equal(t, want, got, "new validator disagrees with fixture")
			require.Equal(t, want, todayAccepts, "today's behavior disagrees with fixture")
		})
	}
}

// readWant parses the "# want: valid|invalid" header on a fixture's first
// line.
func readWant(t *testing.T, raw []byte) bool {
	t.Helper()
	line, _, _ := bufio.NewReader(bytes.NewReader(raw)).ReadLine()
	switch strings.TrimSpace(strings.TrimPrefix(string(line), "# want:")) {
	case "valid":
		return true
	case "invalid":
		return false
	}
	t.Fatalf("fixture must start with %q or %q", "# want: valid", "# want: invalid")
	return false
}

// todayModel evaluates a ProjectConfig the way Kubernetes does today: the CRD
// schema as the API server applies it, followed by the admission webhook.
type todayModel struct {
	structural   *structuralschema.Structural
	schema       apiextensionsvalidation.SchemaValidator
	celValidator *cel.Validator
	webhook      *admission.Webhook
}

func newTodayModel(t *testing.T) *todayModel {
	t.Helper()

	crdBytes, err := os.ReadFile(parityCRDPath)
	require.NoError(t, err)
	crd := &apiextensionsv1.CustomResourceDefinition{}
	require.NoError(t, yaml.Unmarshal(crdBytes, crd))
	require.Len(t, crd.Spec.Versions, 1)

	internal := &apiextensions.CustomResourceValidation{}
	require.NoError(
		t,
		apiextensionsv1.Convert_v1_CustomResourceValidation_To_apiextensions_CustomResourceValidation(
			crd.Spec.Versions[0].Schema,
			internal,
			nil,
		),
	)
	structural, err := structuralschema.NewStructural(internal.OpenAPIV3Schema)
	require.NoError(t, err)
	schemaValidator, _, err := apiextensionsvalidation.NewSchemaValidator(
		internal.OpenAPIV3Schema,
	)
	require.NoError(t, err)

	return &todayModel{
		structural:   structural,
		schema:       schemaValidator,
		celValidator: cel.NewValidator(structural, true, celconfig.PerCallLimit),
		webhook:      newTodayWebhook(t),
	}
}

// toUnstructured decodes a fixture into the generic form the API server
// works with, then applies the API server's request-namespace handling.
func (m *todayModel) toUnstructured(t *testing.T, raw []byte) (map[string]any, bool) {
	t.Helper()
	jsonBytes, err := yaml.YAMLToJSON(raw)
	require.NoError(t, err)
	obj := map[string]any{}
	require.NoError(t, utiljson.Unmarshal(jsonBytes, &obj))

	meta, _ := obj["metadata"].(map[string]any)
	if meta == nil {
		meta = map[string]any{}
		obj["metadata"] = meta
	}
	// The API server rejects a body whose namespace differs from the request
	// namespace, and fills it in when absent.
	if ns, _ := meta["namespace"].(string); ns != "" && ns != parityProject {
		return nil, false
	}
	meta["namespace"] = parityProject

	structuralpruning.Prune(obj, m.structural, true)
	structuraldefaulting.Default(obj, m.structural)
	return obj, true
}

// crdSchemaAccepts mirrors customResourceStrategy.Validate for a create.
func (m *todayModel) crdSchemaAccepts(t *testing.T, raw []byte) bool {
	t.Helper()
	obj, ok := m.toUnstructured(t, raw)
	if !ok {
		t.Log("today: namespace does not match the request namespace")
		return false
	}
	errs := apiextensionsvalidation.ValidateCustomResource(nil, obj, m.schema)
	errs = append(
		errs,
		structurallisttype.ValidateListSetsAndMaps(nil, m.structural, obj)...,
	)
	celErrs, _ := m.celValidator.Validate(
		t.Context(),
		nil,
		m.structural,
		obj,
		nil,
		celconfig.RuntimeCELCostBudget,
	)
	errs = append(errs, celErrs...)
	logErrs(t, "today (CRD schema)", errs)
	return len(errs) == 0
}

// webhookAccepts sends the pruned, defaulted object to the existing admission
// webhook as a create.
func (m *todayModel) webhookAccepts(t *testing.T, raw []byte) bool {
	t.Helper()
	obj, ok := m.toUnstructured(t, raw)
	if !ok {
		return false
	}
	objBytes, err := json.Marshal(obj)
	require.NoError(t, err)
	dryRun := false
	resp := m.webhook.Handle(
		context.Background(),
		admission.Request{
			AdmissionRequest: admissionv1.AdmissionRequest{
				UID:       "parity",
				Operation: admissionv1.Create,
				Namespace: parityProject,
				DryRun:    &dryRun,
				Object:    runtime.RawExtension{Raw: objBytes},
			},
		},
	)
	if !resp.Allowed {
		t.Logf("today (webhook): %s", resp.Result.Message)
	}
	return resp.Allowed
}

func logErrs(t *testing.T, prefix string, errs field.ErrorList) {
	t.Helper()
	for _, e := range errs {
		t.Logf("%s: %s", prefix, e)
	}
}

// newTodayWebhook builds the existing ProjectConfig admission webhook through
// its exported setup function, capturing the handler it registers.
func newTodayWebhook(t *testing.T) *admission.Webhook {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, kargoapi.AddToScheme(scheme))
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: parityProject,
				Labels: map[string]string{
					kargoapi.LabelKeyProject: kargoapi.LabelValueTrue,
				},
			},
		}).
		Build()
	srv := &capturingWebhookServer{}
	require.NoError(
		t,
		projectconfigwebhook.SetupWebhookWithManager(&fakeManager{
			scheme: scheme,
			client: c,
			server: srv,
		}),
	)
	require.Len(t, srv.hooks, 1)
	for _, h := range srv.hooks {
		wh, ok := h.(*admission.Webhook)
		require.True(t, ok)
		return wh
	}
	return nil
}

// fakeManager provides the few Manager methods the webhook builder uses. Any
// other method panics on the nil embedded interface.
type fakeManager struct {
	ctrl.Manager
	scheme *runtime.Scheme
	client client.Client
	server webhook.Server
}

func (f *fakeManager) GetScheme() *runtime.Scheme       { return f.scheme }
func (f *fakeManager) GetClient() client.Client         { return f.client }
func (f *fakeManager) GetConfig() *rest.Config          { return &rest.Config{} }
func (f *fakeManager) GetWebhookServer() webhook.Server { return f.server }

// capturingWebhookServer records registered handlers instead of serving them.
type capturingWebhookServer struct {
	webhook.Server
	hooks map[string]http.Handler
}

func (s *capturingWebhookServer) Register(path string, hook http.Handler) {
	if s.hooks == nil {
		s.hooks = map[string]http.Handler{}
	}
	s.hooks[path] = hook
}

func (s *capturingWebhookServer) WebhookMux() *http.ServeMux { return nil }
