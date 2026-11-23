package projectconfigs

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// fakeLedger is an in-memory importLedger.
type fakeLedger struct {
	mu sync.Mutex
	// recorded is whether the import is recorded.
	recorded bool
	// recordWhileLocking records the import while lock blocks, as another
	// replica that held the lock first would have.
	recordWhileLocking bool
	// locks counts acquisitions of the lock and releases its releases.
	locks    int
	releases int
	// err, when set, fails every call.
	err error
}

func (l *fakeLedger) completed(context.Context) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.recorded, l.err
}

func (l *fakeLedger) lock(context.Context) (func(), error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.err != nil {
		return nil, l.err
	}
	l.locks++
	if l.recordWhileLocking {
		l.recorded = true
	}
	return func() {
		l.mu.Lock()
		defer l.mu.Unlock()
		l.releases++
	}, nil
}

func (l *fakeLedger) record(context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.err != nil {
		return l.err
	}
	l.recorded = true
	return nil
}

type importCall struct {
	cfg        *kargoapi.ProjectConfig
	projectUID string
}

// fakeImportStore is an in-memory configImporter.
type fakeImportStore struct {
	mu sync.Mutex
	// existing names the Projects that already have a ProjectConfig.
	existing map[string]bool
	// calls records every Import.
	calls []importCall
	// err, when set, fails every call.
	err error
}

func (s *fakeImportStore) Import(
	_ context.Context,
	cfg *kargoapi.ProjectConfig,
	projectUID string,
) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return false, s.err
	}
	s.calls = append(s.calls, importCall{cfg: cfg, projectUID: projectUID})
	if s.existing[cfg.Name] {
		return false, nil
	}
	if s.existing == nil {
		s.existing = map[string]bool{}
	}
	s.existing[cfg.Name] = true
	return true, nil
}

func (s *fakeImportStore) imported() map[string]importCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	calls := map[string]importCall{}
	for _, c := range s.calls {
		calls[c.cfg.Name] = c
	}
	return calls
}

// errMapper is a RESTMapper whose every mapping fails.
type errMapper struct {
	meta.RESTMapper
	err error
}

func (m errMapper) RESTMapping(schema.GroupKind, ...string) (*meta.RESTMapping, error) {
	return nil, m.err
}

// importerTestScheme deliberately leaves ProjectConfig out: the importer
// must read the resources without the typed kind registered.
func importerTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(
		kargoapi.GroupVersion,
		&kargoapi.Project{},
		&kargoapi.ProjectList{},
	)
	metav1.AddToGroupVersion(scheme, kargoapi.GroupVersion)
	require.NoError(t, admissionregistrationv1.AddToScheme(scheme))
	return scheme
}

func projectConfigMapper() meta.RESTMapper {
	mapper := meta.NewDefaultRESTMapper(nil)
	mapper.Add(projectConfigGVK, meta.RESTScopeNamespace)
	mapper.Add(kargoapi.GroupVersion.WithKind("Project"), meta.RESTScopeRoot)
	mapper.Add(
		admissionregistrationv1.SchemeGroupVersion.WithKind("ValidatingAdmissionPolicyBinding"),
		meta.RESTScopeRoot,
	)
	return mapper
}

func testProjectObj(name string) *kargoapi.Project {
	return &kargoapi.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			UID:  types.UID(name + "-uid"),
		},
	}
}

func testBinding() *admissionregistrationv1.ValidatingAdmissionPolicyBinding {
	return &admissionregistrationv1.ValidatingAdmissionPolicyBinding{
		ObjectMeta: metav1.ObjectMeta{Name: policyBindingName},
	}
}

// testProjectConfigCR returns a ProjectConfig resource for a Project, as
// unstructured, with a webhook receiver for each of the given paths.
func testProjectConfigCR(
	t *testing.T,
	project string,
	paths ...string,
) *unstructured.Unstructured {
	t.Helper()
	cfg := &kargoapi.ProjectConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      project,
			Namespace: project,
		},
		Spec: kargoapi.ProjectConfigSpec{
			PromotionPolicies: []kargoapi.PromotionPolicy{{
				StageSelector:        &kargoapi.PromotionPolicySelector{Name: "dev"},
				AutoPromotionEnabled: true,
			}},
		},
		Status: kargoapi.ProjectConfigStatus{
			Conditions: []metav1.Condition{{
				Type:               kargoapi.ConditionTypeReady,
				Status:             metav1.ConditionTrue,
				Reason:             "Synced",
				LastTransitionTime: metav1.NewTime(time.Unix(1, 0).UTC()),
			}},
		},
	}
	for i, path := range paths {
		cfg.Status.WebhookReceivers = append(
			cfg.Status.WebhookReceivers,
			kargoapi.WebhookReceiverDetails{
				Name: project + "-" + string(rune('a'+i)),
				Path: path,
				URL:  "https://webhooks.example.com" + path,
			},
		)
	}
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(cfg)
	require.NoError(t, err)
	u := &unstructured.Unstructured{Object: obj}
	u.SetGroupVersionKind(projectConfigGVK)
	return u
}

func receiverPaths(cfg *kargoapi.ProjectConfig) []string {
	var paths []string
	for _, r := range cfg.Status.WebhookReceivers {
		paths = append(paths, r.Path)
	}
	return paths
}

func TestImporterRun(t *testing.T) {
	type fixture struct {
		ledger *fakeLedger
		store  *fakeImportStore
		lists  int
	}
	testCases := []struct {
		name string
		// objects are what the cluster holds. Unless noBinding is set, the
		// policy binding is added to them.
		objects   []client.Object
		noBinding bool
		ledger    *fakeLedger
		store     *fakeImportStore
		mapper    meta.RESTMapper
		funcs     interceptor.Funcs
		timeout   time.Duration
		assert    func(*testing.T, fixture, error)
	}{
		{
			name:   "already recorded",
			ledger: &fakeLedger{recorded: true},
			assert: func(t *testing.T, f fixture, err error) {
				require.NoError(t, err)
				require.Zero(t, f.ledger.locks)
				require.Zero(t, f.lists)
			},
		},
		{
			name:   "error reading the record",
			ledger: &fakeLedger{err: errors.New("connection refused")},
			assert: func(t *testing.T, f fixture, err error) {
				require.ErrorContains(t, err, "connection refused")
				require.False(t, f.ledger.recorded)
			},
		},
		{
			name:   "recorded by another replica while waiting for the lock",
			ledger: &fakeLedger{recordWhileLocking: true},
			objects: []client.Object{
				testProjectObj("p1"),
				testProjectConfigCR(t, "p1"),
			},
			assert: func(t *testing.T, f fixture, err error) {
				require.NoError(t, err)
				require.Equal(t, 1, f.ledger.releases)
				require.Zero(t, f.lists)
				require.Empty(t, f.store.calls)
			},
		},
		{
			name:   "CRD missing",
			mapper: meta.NewDefaultRESTMapper(nil),
			assert: func(t *testing.T, f fixture, err error) {
				require.NoError(t, err)
				require.True(t, f.ledger.recorded)
				require.Zero(t, f.lists)
				require.Equal(t, 1, f.ledger.releases)
			},
		},
		{
			name:   "discovery error",
			mapper: errMapper{err: errors.New("the server is currently unable to handle the request")},
			assert: func(t *testing.T, f fixture, err error) {
				require.ErrorContains(t, err, "unable to handle the request")
				require.False(t, f.ledger.recorded)
				require.Zero(t, f.lists)
				require.Equal(t, 1, f.ledger.releases)
			},
		},
		{
			name: "list forbidden",
			funcs: interceptor.Funcs{
				List: func(
					context.Context,
					client.WithWatch,
					client.ObjectList,
					...client.ListOption,
				) error {
					return apierrors.NewForbidden(
						schema.GroupResource{Group: kargoapi.GroupVersion.Group, Resource: "projectconfigs"},
						"",
						errors.New("no"),
					)
				},
			},
			assert: func(t *testing.T, f fixture, err error) {
				require.True(t, apierrors.IsForbidden(err))
				require.False(t, f.ledger.recorded)
			},
		},
		{
			name:      "waits for the policy binding",
			noBinding: true,
			objects: []client.Object{
				testProjectObj("p1"),
				testProjectConfigCR(t, "p1"),
			},
			funcs: func() interceptor.Funcs {
				var mu sync.Mutex
				gets := 0
				return interceptor.Funcs{
					Get: func(
						ctx context.Context,
						c client.WithWatch,
						key client.ObjectKey,
						obj client.Object,
						opts ...client.GetOption,
					) error {
						if _, ok := obj.(*admissionregistrationv1.ValidatingAdmissionPolicyBinding); ok {
							mu.Lock()
							gets++
							first := gets == 1
							mu.Unlock()
							if first {
								// The binding appears after the first poll.
								if err := c.Get(ctx, key, obj, opts...); !apierrors.IsNotFound(err) {
									return errors.New("binding exists before the first poll")
								}
								if err := c.Create(ctx, testBinding()); err != nil {
									return err
								}
								return apierrors.NewNotFound(
									admissionregistrationv1.Resource("validatingadmissionpolicybindings"),
									key.Name,
								)
							}
						}
						return c.Get(ctx, key, obj, opts...)
					},
				}
			}(),
			assert: func(t *testing.T, f fixture, err error) {
				require.NoError(t, err)
				require.Contains(t, f.store.imported(), "p1")
				require.True(t, f.ledger.recorded)
			},
		},
		{
			name:      "policy binding never appears",
			noBinding: true,
			timeout:   50 * time.Millisecond,
			assert: func(t *testing.T, f fixture, err error) {
				require.ErrorIs(t, err, context.DeadlineExceeded)
				require.Zero(t, f.lists)
				require.False(t, f.ledger.recorded)
			},
		},
		{
			name: "error reading the policy binding",
			funcs: interceptor.Funcs{
				Get: func(
					ctx context.Context,
					c client.WithWatch,
					key client.ObjectKey,
					obj client.Object,
					opts ...client.GetOption,
				) error {
					if _, ok := obj.(*admissionregistrationv1.ValidatingAdmissionPolicyBinding); ok {
						return apierrors.NewForbidden(
							admissionregistrationv1.Resource("validatingadmissionpolicybindings"),
							key.Name,
							errors.New("no"),
						)
					}
					return c.Get(ctx, key, obj, opts...)
				},
			},
			assert: func(t *testing.T, f fixture, err error) {
				require.True(t, apierrors.IsForbidden(err))
				require.Zero(t, f.lists)
				require.False(t, f.ledger.recorded)
			},
		},
		{
			name: "imports each ProjectConfig for its Project",
			objects: []client.Object{
				testProjectObj("p1"),
				testProjectObj("p2"),
				testProjectConfigCR(t, "p1", "/p1"),
				testProjectConfigCR(t, "p2", "/p2a", "/p2b"),
			},
			assert: func(t *testing.T, f fixture, err error) {
				require.NoError(t, err)
				imported := f.store.imported()
				require.Len(t, imported, 2)
				p1 := imported["p1"]
				require.Equal(t, "p1-uid", p1.projectUID)
				require.Equal(t, "p1", p1.cfg.Namespace)
				require.Equal(t, "dev", p1.cfg.Spec.PromotionPolicies[0].StageSelector.Name)
				require.Equal(t, []string{"/p1"}, receiverPaths(p1.cfg))
				require.Len(t, p1.cfg.Status.Conditions, 1)
				require.Equal(t, "Synced", p1.cfg.Status.Conditions[0].Reason)
				require.Equal(t, []string{"/p2a", "/p2b"}, receiverPaths(imported["p2"].cfg))
				require.True(t, f.ledger.recorded)
				require.Equal(t, 1, f.ledger.releases)
			},
		},
		{
			name: "skips a ProjectConfig being deleted",
			objects: []client.Object{
				testProjectObj("p1"),
				testProjectObj("p2"),
				testProjectConfigCR(t, "p1"),
				func() client.Object {
					cr := testProjectConfigCR(t, "p2")
					now := metav1.Now()
					cr.SetDeletionTimestamp(&now)
					cr.SetFinalizers([]string{"kargo.akuity.io/test"})
					return cr
				}(),
			},
			assert: func(t *testing.T, f fixture, err error) {
				require.NoError(t, err)
				imported := f.store.imported()
				require.Contains(t, imported, "p1")
				require.NotContains(t, imported, "p2")
				require.True(t, f.ledger.recorded)
			},
		},
		{
			name: "skips a ProjectConfig without a Project",
			objects: []client.Object{
				testProjectObj("p1"),
				testProjectConfigCR(t, "p1"),
				testProjectConfigCR(t, "orphan"),
			},
			assert: func(t *testing.T, f fixture, err error) {
				require.NoError(t, err)
				imported := f.store.imported()
				require.Contains(t, imported, "p1")
				require.NotContains(t, imported, "orphan")
				require.True(t, f.ledger.recorded)
			},
		},
		{
			name: "error reading a Project",
			objects: []client.Object{
				testProjectObj("p1"),
				testProjectConfigCR(t, "p1"),
			},
			funcs: interceptor.Funcs{
				Get: func(
					ctx context.Context,
					c client.WithWatch,
					key client.ObjectKey,
					obj client.Object,
					opts ...client.GetOption,
				) error {
					if _, ok := obj.(*kargoapi.Project); ok {
						return errors.New("etcdserver: request timed out")
					}
					return c.Get(ctx, key, obj, opts...)
				},
			},
			assert: func(t *testing.T, f fixture, err error) {
				require.ErrorContains(t, err, "request timed out")
				require.Empty(t, f.store.calls)
				require.False(t, f.ledger.recorded)
			},
		},
		{
			name: "a path claimed by two ProjectConfigs is imported for neither",
			objects: []client.Object{
				testProjectObj("p1"),
				testProjectObj("p2"),
				testProjectConfigCR(t, "p1", "/shared", "/p1"),
				testProjectConfigCR(t, "p2", "/p2", "/shared"),
			},
			assert: func(t *testing.T, f fixture, err error) {
				require.NoError(t, err)
				imported := f.store.imported()
				require.Equal(t, []string{"/p1"}, receiverPaths(imported["p1"].cfg))
				require.Equal(t, []string{"/p2"}, receiverPaths(imported["p2"].cfg))
				require.True(t, f.ledger.recorded)
			},
		},
		{
			name: "a path repeated within one ProjectConfig is kept",
			objects: []client.Object{
				testProjectObj("p1"),
				testProjectConfigCR(t, "p1", "/p1", "/p1"),
			},
			assert: func(t *testing.T, f fixture, err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"/p1", "/p1"}, receiverPaths(f.store.imported()["p1"].cfg))
			},
		},
		{
			name:  "an existing ProjectConfig is not overwritten",
			store: &fakeImportStore{existing: map[string]bool{"p1": true}},
			objects: []client.Object{
				testProjectObj("p1"),
				testProjectObj("p2"),
				testProjectConfigCR(t, "p1"),
				testProjectConfigCR(t, "p2"),
			},
			assert: func(t *testing.T, f fixture, err error) {
				// The store, which never overwrites, decides; the importer
				// carries on with the rest and records the import.
				require.NoError(t, err)
				require.Len(t, f.store.calls, 2)
				require.True(t, f.ledger.recorded)
			},
		},
		{
			name:  "error importing",
			store: &fakeImportStore{err: errors.New("deadlock detected")},
			objects: []client.Object{
				testProjectObj("p1"),
				testProjectConfigCR(t, "p1"),
			},
			assert: func(t *testing.T, f fixture, err error) {
				require.ErrorContains(t, err, "deadlock detected")
				require.ErrorContains(t, err, `"p1"`)
				require.False(t, f.ledger.recorded)
				require.Equal(t, 1, f.ledger.releases)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			f := fixture{ledger: testCase.ledger, store: testCase.store}
			if f.ledger == nil {
				f.ledger = &fakeLedger{}
			}
			if f.store == nil {
				f.store = &fakeImportStore{}
			}
			mapper := testCase.mapper
			if mapper == nil {
				mapper = projectConfigMapper()
			}
			objects := testCase.objects
			if !testCase.noBinding {
				objects = append(objects, testBinding())
			}
			var mu sync.Mutex
			funcs := testCase.funcs
			list := funcs.List
			funcs.List = func(
				ctx context.Context,
				c client.WithWatch,
				l client.ObjectList,
				opts ...client.ListOption,
			) error {
				mu.Lock()
				f.lists++
				mu.Unlock()
				if list != nil {
					return list(ctx, c, l, opts...)
				}
				return c.List(ctx, l, opts...)
			}
			kube := fake.NewClientBuilder().
				WithScheme(importerTestScheme(t)).
				WithRESTMapper(projectConfigMapper()).
				WithObjects(objects...).
				WithInterceptorFuncs(funcs).
				Build()
			importer := newImporter(f.ledger, f.store, kube, mapper)
			importer.pollInterval = time.Millisecond
			ctx := context.Background()
			if testCase.timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, testCase.timeout)
				defer cancel()
			}
			err := importer.Run(ctx)
			mu.Lock()
			defer mu.Unlock()
			testCase.assert(t, f, err)
		})
	}
}
