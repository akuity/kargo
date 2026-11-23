package projectconfigs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/client/apiserver"
	"github.com/akuity/kargo/pkg/conditions"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/database/projectconfigstore"
	engine "github.com/akuity/kargo/pkg/reconciler"
	"github.com/akuity/kargo/pkg/server/projectconfigs/wire"
	"github.com/akuity/kargo/pkg/webhook/external"
)

func TestNewReconciler(t *testing.T) {
	testCfg := ReconcilerConfig{}
	kc := fake.NewClientBuilder().Build()
	store := &fakeStore{}
	status := &fakeStatusWriter{}
	// we pass the same client for both the client and api-reader since the
	// fake client doesn't use the cache at all.
	r := newReconciler(kc, kc, store, status, testCfg)
	require.Equal(t, testCfg, r.cfg)
	require.NotNil(t, r.client)
	require.Same(t, store, r.store)
	require.Same(t, status, r.status)
}

func TestReconciler_syncWebhookReceivers(t *testing.T) {
	testScheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(testScheme))

	const testProjectName = "fake-project"

	testCases := []struct {
		name       string
		reconciler *reconciler
		projectCfg *kargoapi.ProjectConfig
		assertions func(*testing.T, kargoapi.ProjectConfigStatus, error)
	}{
		{
			name:       "project config does not define any webhook receivers",
			reconciler: &reconciler{},
			projectCfg: &kargoapi.ProjectConfig{},
			assertions: func(t *testing.T, status kargoapi.ProjectConfigStatus, err error) {
				require.NoError(t, err)
				require.Empty(t, status.WebhookReceivers)
				require.Len(t, status.Conditions, 1)
				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionTrue, readyCondition.Status)
				require.Equal(t, "Synced", readyCondition.Reason)
			},
		},
		{
			name:       "stale webhook receivers are cleared when spec is empty",
			reconciler: &reconciler{},
			projectCfg: &kargoapi.ProjectConfig{
				Status: kargoapi.ProjectConfigStatus{
					WebhookReceivers: []kargoapi.WebhookReceiverDetails{{
						Name: "stale-receiver",
						Path: "/webhook/github/abc123",
					}},
				},
			},
			assertions: func(t *testing.T, status kargoapi.ProjectConfigStatus, err error) {
				require.NoError(t, err)
				require.Empty(t, status.WebhookReceivers)
				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionTrue, readyCondition.Status)
				require.Equal(t, "Synced", readyCondition.Reason)
			},
		},
		{
			name: "error building receiver",
			reconciler: &reconciler{
				client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProjectName,
							Name:      "fake-token-secret",
						},
						Data: map[string][]byte{external.GithubSecretDataKey: []byte("fake-token")},
					},
				).Build(),
			},
			projectCfg: &kargoapi.ProjectConfig{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProjectName,
					Name:      testProjectName,
				},
				Spec: kargoapi.ProjectConfigSpec{
					WebhookReceivers: []kargoapi.WebhookReceiverConfig{
						{
							Name: "invalid-receiver",
							GitHub: &kargoapi.GitHubWebhookReceiverConfig{
								SecretRef: corev1.LocalObjectReference{
									Name: "non-existent-secret",
								},
							},
						},
						{
							Name: "valid-receiver",
							GitHub: &kargoapi.GitHubWebhookReceiverConfig{
								SecretRef: corev1.LocalObjectReference{
									Name: "fake-token-secret",
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.ProjectConfigStatus, err error) {
				// We should get an error because the first receiver's SecretRef could
				// not be resolved.
				require.ErrorContains(t, err, "not found")

				// But the second receiver should still have been processed.
				require.Len(t, status.WebhookReceivers, 1)
				require.Equal(t, "valid-receiver", status.WebhookReceivers[0].Name)
				require.NotEmpty(t, status.WebhookReceivers[0].Path)
				require.NotEmpty(t, status.WebhookReceivers[0].URL)

				// The conditions should reflect the error and that the ProjectConfig is
				// still syncing.
				require.Len(t, status.Conditions, 2)
				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionFalse, readyCondition.Status)
				require.Equal(t, "SyncWebhookReceiversFailed", readyCondition.Reason)
				reconcilingCondition := conditions.Get(&status, kargoapi.ConditionTypeReconciling)
				require.NotNil(t, reconcilingCondition)
				require.Equal(t, metav1.ConditionTrue, reconcilingCondition.Status)
				require.Equal(t, "Syncing WebhookReceivers", reconcilingCondition.Message)
			},
		},
		{
			name: "great success!",
			reconciler: &reconciler{
				client: fake.NewClientBuilder().WithScheme(testScheme).WithObjects(
					&corev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: testProjectName,
							Name:      "fake-token-secret",
						},
						Data: map[string][]byte{external.GithubSecretDataKey: []byte("fake-token")},
					},
				).Build(),
			},
			projectCfg: &kargoapi.ProjectConfig{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testProjectName,
					Name:      testProjectName,
				},
				Spec: kargoapi.ProjectConfigSpec{
					WebhookReceivers: []kargoapi.WebhookReceiverConfig{
						{
							Name: "valid-receiver",
							GitHub: &kargoapi.GitHubWebhookReceiverConfig{
								SecretRef: corev1.LocalObjectReference{
									Name: "fake-token-secret",
								},
							},
						},
					},
				},
			},
			assertions: func(t *testing.T, status kargoapi.ProjectConfigStatus, err error) {
				require.NoError(t, err)

				// But the second receiver should still have been processed.
				require.Len(t, status.WebhookReceivers, 1)
				require.Equal(t, "valid-receiver", status.WebhookReceivers[0].Name)
				require.NotEmpty(t, status.WebhookReceivers[0].Path)
				require.NotEmpty(t, status.WebhookReceivers[0].URL)

				// The conditions should reflect success.
				require.Len(t, status.Conditions, 1)
				readyCondition := conditions.Get(&status, kargoapi.ConditionTypeReady)
				require.NotNil(t, readyCondition)
				require.Equal(t, metav1.ConditionTrue, readyCondition.Status)
				require.Equal(t, "Synced", readyCondition.Reason)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// succinct way of setting the apiReader to the same fake client
			// since the fake client doesn't use the cache at all.
			testCase.reconciler.apiReader = testCase.reconciler.client
			status, err := testCase.reconciler.syncWebhookReceivers(
				t.Context(),
				testCase.projectCfg,
			)
			testCase.assertions(t, status, err)
		})
	}
}

const (
	testProject    = "fake-project"
	testProjectUID = "fake-project-uid"
	testSecret     = "fake-token-secret"
	testBaseURL    = "https://webhooks.example.com"
	testToken      = "fake-refresh-token"
)

var testConfigID = uuid.MustParse("6f2d3c1e-9a3b-4f61-8d2a-0c4f6c9d1e01")

// fakeStore is a configStore holding at most one ProjectConfig.
type fakeStore struct {
	cfg *projectconfigstore.Config
	// next, if not empty, replaces cfg after each GetUnfiltered, one entry
	// per call, to model a ProjectConfig changing between reads. A nil entry
	// is a deleted ProjectConfig.
	next    []*projectconfigstore.Config
	getErr  error
	list    []projectconfigstore.Config
	listErr error
	deleted []deleteCall
	adopted []adoptCall
	// adoptErr, if set, is returned by AdoptForProject.
	adoptErr error
	// owners maps a webhook receiver path to the Project whose ProjectConfig
	// owns it. A path not in it is owned by none.
	owners    map[string]string
	lookupErr error
}

type deleteCall struct {
	project    string
	projectUID string
}

type adoptCall struct {
	project string
	newUID  string
}

func (s *fakeStore) GetUnfiltered(
	_ context.Context,
	project string,
) (projectconfigstore.Config, error) {
	if s.getErr != nil {
		return projectconfigstore.Config{}, s.getErr
	}
	cfg := s.cfg
	if len(s.next) > 0 {
		s.cfg, s.next = s.next[0], s.next[1:]
	}
	if cfg == nil || cfg.ProjectName != project {
		return projectconfigstore.Config{}, database.ErrNotFound
	}
	return *cfg, nil
}

func (s *fakeStore) List(context.Context) ([]projectconfigstore.Config, error) {
	return s.list, s.listErr
}

func (s *fakeStore) GetByWebhookReceiverPath(
	_ context.Context,
	path string,
) (projectconfigstore.Config, error) {
	if s.lookupErr != nil {
		return projectconfigstore.Config{}, s.lookupErr
	}
	owner, ok := s.owners[path]
	if !ok {
		return projectconfigstore.Config{}, database.ErrNotFound
	}
	return projectconfigstore.Config{
		ProjectConfigRow: database.ProjectConfigRow{ProjectName: owner},
	}, nil
}

func (s *fakeStore) DeleteForProject(
	_ context.Context,
	project string,
	projectUID string,
) error {
	s.deleted = append(s.deleted, deleteCall{
		project:    project,
		projectUID: projectUID,
	})
	return nil
}

func (s *fakeStore) AdoptForProject(
	_ context.Context,
	project string,
	newUID string,
) (bool, error) {
	s.adopted = append(s.adopted, adoptCall{project: project, newUID: newUID})
	if s.adoptErr != nil {
		return false, s.adoptErr
	}
	return true, nil
}

// fakeStatusWriter records the status requests it is given. The nth call
// returns skipped[n], if present, and err.
type fakeStatusWriter struct {
	reqs    []wire.StatusRequest
	skipped [][]string
	err     error
}

func (w *fakeStatusWriter) UpdateProjectConfigStatus(
	_ context.Context,
	project string,
	req wire.StatusRequest,
) ([]string, error) {
	if project != testProject {
		return nil, fmt.Errorf("unexpected project %q", project)
	}
	n := len(w.reqs)
	w.reqs = append(w.reqs, req)
	if w.err != nil {
		return nil, w.err
	}
	if n < len(w.skipped) {
		return w.skipped[n], nil
	}
	return nil, nil
}

// applyStatus records a status request on a stored config the way the store
// does.
func applyStatus(
	t *testing.T,
	cfg *projectconfigstore.Config,
	req wire.StatusRequest,
) {
	t.Helper()
	conds, err := json.Marshal(req.Conditions)
	require.NoError(t, err)
	cfg.StatusConditions = conds
	gen := req.ObservedGeneration
	cfg.StatusObservedGeneration = &gen
	cfg.StatusLastHandledRefresh = nil
	if req.ObservedRefreshToken != "" {
		token := req.ObservedRefreshToken
		cfg.StatusLastHandledRefresh = &token
	}
	cfg.Receivers = nil
	for _, receiver := range req.WebhookReceivers {
		cfg.Receivers = append(cfg.Receivers, database.ProjectConfigWebhookReceiverRow{
			ProjectConfigID: cfg.ID,
			Name:            receiver.Name,
			Path:            receiver.Path,
			URL:             receiver.URL,
		})
	}
}

// newTestConfig returns a stored ProjectConfig with GitHub receivers of the
// given names, all reading the test Secret, and no status.
func newTestConfig(t *testing.T, receiverNames ...string) *projectconfigstore.Config {
	t.Helper()
	spec := kargoapi.ProjectConfigSpec{}
	for _, name := range receiverNames {
		spec.WebhookReceivers = append(spec.WebhookReceivers, kargoapi.WebhookReceiverConfig{
			Name: name,
			GitHub: &kargoapi.GitHubWebhookReceiverConfig{
				SecretRef: corev1.LocalObjectReference{Name: testSecret},
			},
		})
	}
	specJSON, err := json.Marshal(spec)
	require.NoError(t, err)
	token := testToken
	return &projectconfigstore.Config{
		ProjectConfigRow: database.ProjectConfigRow{
			ID:           testConfigID,
			ProjectName:  testProject,
			ProjectUID:   testProjectUID,
			Spec:         specJSON,
			Generation:   2,
			RefreshToken: &token,
		},
	}
}

// newTestKubeClient returns a client holding the test Secret and, unless
// projectUID is empty, the test Project with that UID.
func newTestKubeClient(t *testing.T, projectUID string) client.Client {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, kargoapi.AddToScheme(scheme))
	objs := []client.Object{
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Namespace: testProject, Name: testSecret},
			Data:       map[string][]byte{external.GithubSecretDataKey: []byte("fake-token")},
		},
	}
	if projectUID != "" {
		objs = append(objs, &kargoapi.Project{
			ObjectMeta: metav1.ObjectMeta{
				Name: testProject,
				UID:  types.UID(projectUID),
			},
		})
	}
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
}

func newTestReconciler(
	t *testing.T,
	projectUID string,
	store *fakeStore,
	status *fakeStatusWriter,
) *reconciler {
	t.Helper()
	kc := newTestKubeClient(t, projectUID)
	return newReconciler(
		kc,
		kc,
		store,
		status,
		ReconcilerConfig{ExternalWebhookServerBaseURL: testBaseURL},
	)
}

func readyCondition(t *testing.T, req wire.StatusRequest) metav1.Condition {
	t.Helper()
	cond := meta.FindStatusCondition(req.Conditions, kargoapi.ConditionTypeReady)
	require.NotNil(t, cond)
	return *cond
}

func TestReconciler_Reconcile(t *testing.T) {
	t.Parallel()
	key := projectconfigstore.Key{Project: testProject}
	testCases := []struct {
		name       string
		projectUID string
		store      func(*testing.T) *fakeStore
		status     *fakeStatusWriter
		assert     func(*testing.T, *fakeStore, *fakeStatusWriter, engine.Result, error)
	}{
		{
			name:       "ProjectConfig not found",
			projectUID: testProjectUID,
			store: func(*testing.T) *fakeStore {
				return &fakeStore{}
			},
			status: &fakeStatusWriter{},
			assert: func(t *testing.T, s *fakeStore, w *fakeStatusWriter, res engine.Result, err error) {
				require.NoError(t, err)
				require.Zero(t, res)
				require.Empty(t, s.deleted)
				require.Empty(t, w.reqs)
			},
		},
		{
			name:       "error reading ProjectConfig",
			projectUID: testProjectUID,
			store: func(*testing.T) *fakeStore {
				return &fakeStore{getErr: errors.New("boom")}
			},
			status: &fakeStatusWriter{},
			assert: func(t *testing.T, _ *fakeStore, w *fakeStatusWriter, _ engine.Result, err error) {
				require.ErrorContains(t, err, "boom")
				require.Empty(t, w.reqs)
			},
		},
		{
			name:       "leftover row: Project missing",
			projectUID: "",
			store: func(t *testing.T) *fakeStore {
				return &fakeStore{cfg: newTestConfig(t, "a")}
			},
			status: &fakeStatusWriter{},
			assert: func(t *testing.T, s *fakeStore, w *fakeStatusWriter, res engine.Result, err error) {
				require.NoError(t, err)
				require.Zero(t, res)
				require.Equal(
					t,
					[]deleteCall{{project: testProject, projectUID: testProjectUID}},
					s.deleted,
				)
				require.Empty(t, w.reqs)
			},
		},
		{
			name:       "Project has another UID: adopted, not deleted",
			projectUID: "another-uid",
			store: func(t *testing.T) *fakeStore {
				return &fakeStore{cfg: newTestConfig(t, "a")}
			},
			status: &fakeStatusWriter{},
			assert: func(t *testing.T, s *fakeStore, w *fakeStatusWriter, res engine.Result, err error) {
				require.NoError(t, err)
				// The next pass reconciles the row under its new Project.
				require.Equal(t, adoptRequeueDelay, res.RequeueAfter)
				require.Equal(
					t,
					[]adoptCall{{project: testProject, newUID: "another-uid"}},
					s.adopted,
				)
				require.Empty(t, s.deleted)
				require.Empty(t, w.reqs)
			},
		},
		{
			name:       "Project has another UID: adoption error",
			projectUID: "another-uid",
			store: func(t *testing.T) *fakeStore {
				return &fakeStore{cfg: newTestConfig(t, "a"), adoptErr: errors.New("boom")}
			},
			status: &fakeStatusWriter{},
			assert: func(t *testing.T, s *fakeStore, w *fakeStatusWriter, _ engine.Result, err error) {
				require.ErrorContains(t, err, "boom")
				require.Empty(t, s.deleted)
				require.Empty(t, w.reqs)
			},
		},
		{
			name:       "status written",
			projectUID: testProjectUID,
			store: func(t *testing.T) *fakeStore {
				return &fakeStore{cfg: newTestConfig(t, "a")}
			},
			status: &fakeStatusWriter{},
			assert: func(t *testing.T, s *fakeStore, w *fakeStatusWriter, res engine.Result, err error) {
				require.NoError(t, err)
				require.Zero(t, res)
				require.Empty(t, s.deleted)
				require.Len(t, w.reqs, 1)
				req := w.reqs[0]
				require.Equal(t, testConfigID.String(), req.ObservedUID)
				require.Equal(t, int64(2), req.ObservedGeneration)
				// The store records the observed token as the handled refresh.
				require.Equal(t, testToken, req.ObservedRefreshToken)
				ready := readyCondition(t, req)
				require.Equal(t, metav1.ConditionTrue, ready.Status)
				require.Equal(t, "Synced", ready.Reason)
				require.Len(t, req.WebhookReceivers, 1)
				require.Equal(t, "a", req.WebhookReceivers[0].Name)
				require.NotEmpty(t, req.WebhookReceivers[0].Path)
			},
		},
		{
			name:       "no refresh token",
			projectUID: testProjectUID,
			store: func(t *testing.T) *fakeStore {
				cfg := newTestConfig(t, "a")
				cfg.RefreshToken = nil
				return &fakeStore{cfg: cfg}
			},
			status: &fakeStatusWriter{},
			assert: func(t *testing.T, _ *fakeStore, w *fakeStatusWriter, _ engine.Result, err error) {
				require.NoError(t, err)
				require.Len(t, w.reqs, 1)
				require.Empty(t, w.reqs[0].ObservedRefreshToken)
			},
		},
		{
			name:       "error syncing receivers: status written, error returned",
			projectUID: testProjectUID,
			store: func(t *testing.T) *fakeStore {
				cfg := newTestConfig(t, "a")
				spec := kargoapi.ProjectConfigSpec{
					WebhookReceivers: []kargoapi.WebhookReceiverConfig{{
						Name: "a",
						GitHub: &kargoapi.GitHubWebhookReceiverConfig{
							SecretRef: corev1.LocalObjectReference{Name: "missing"},
						},
					}},
				}
				var err error
				cfg.Spec, err = json.Marshal(spec)
				require.NoError(t, err)
				return &fakeStore{cfg: cfg}
			},
			status: &fakeStatusWriter{},
			assert: func(t *testing.T, _ *fakeStore, w *fakeStatusWriter, _ engine.Result, err error) {
				require.ErrorContains(t, err, "not found")
				require.Len(t, w.reqs, 1)
				ready := readyCondition(t, w.reqs[0])
				require.Equal(t, metav1.ConditionFalse, ready.Status)
				require.Equal(t, "SyncWebhookReceiversFailed", ready.Reason)
			},
		},
		{
			name:       "skipped receiver: Ready False naming it, others written",
			projectUID: testProjectUID,
			store: func(t *testing.T) *fakeStore {
				return &fakeStore{cfg: newTestConfig(t, "a", "b")}
			},
			status: &fakeStatusWriter{skipped: [][]string{{"b"}}},
			assert: func(t *testing.T, _ *fakeStore, w *fakeStatusWriter, res engine.Result, err error) {
				require.NoError(t, err)
				require.Zero(t, res)
				require.Len(t, w.reqs, 2)
				require.Len(t, w.reqs[0].WebhookReceivers, 2)
				second := w.reqs[1]
				require.Equal(t, w.reqs[0].ObservedUID, second.ObservedUID)
				require.Equal(t, w.reqs[0].ObservedGeneration, second.ObservedGeneration)
				require.Equal(t, w.reqs[0].ObservedRefreshToken, second.ObservedRefreshToken)
				require.Len(t, second.WebhookReceivers, 1)
				require.Equal(t, "a", second.WebhookReceivers[0].Name)
				ready := readyCondition(t, second)
				require.Equal(t, metav1.ConditionFalse, ready.Status)
				require.Equal(
					t,
					`webhook receiver "b": path is already claimed by another ProjectConfig`,
					ready.Message,
				)
			},
		},
		{
			name:       "conflict: requeued without error",
			projectUID: testProjectUID,
			store: func(t *testing.T) *fakeStore {
				return &fakeStore{cfg: newTestConfig(t, "a")}
			},
			status: &fakeStatusWriter{
				err: fmt.Errorf("wrapped: %w", &apiserver.APIError{StatusCode: http.StatusConflict}),
			},
			assert: func(t *testing.T, _ *fakeStore, w *fakeStatusWriter, res engine.Result, err error) {
				require.NoError(t, err)
				require.Positive(t, res.RequeueAfter)
				require.Len(t, w.reqs, 1)
			},
		},
		{
			name:       "not found from the API, ProjectConfig gone: done",
			projectUID: testProjectUID,
			store: func(t *testing.T) *fakeStore {
				return &fakeStore{
					cfg:  newTestConfig(t, "a"),
					next: []*projectconfigstore.Config{nil},
				}
			},
			status: &fakeStatusWriter{
				err: fmt.Errorf("wrapped: %w", &apiserver.APIError{StatusCode: http.StatusNotFound}),
			},
			assert: func(t *testing.T, _ *fakeStore, w *fakeStatusWriter, res engine.Result, err error) {
				require.NoError(t, err)
				require.Zero(t, res)
				require.Len(t, w.reqs, 1)
			},
		},
		{
			name:       "not found from the API, ProjectConfig replaced: done",
			projectUID: testProjectUID,
			store: func(t *testing.T) *fakeStore {
				replaced := newTestConfig(t, "a")
				replaced.ID = uuid.MustParse("0b8e1f7a-3c2d-4e5f-9a1b-2c3d4e5f6a7b")
				return &fakeStore{
					cfg:  newTestConfig(t, "a"),
					next: []*projectconfigstore.Config{replaced},
				}
			},
			status: &fakeStatusWriter{
				err: fmt.Errorf("wrapped: %w", &apiserver.APIError{StatusCode: http.StatusNotFound}),
			},
			assert: func(t *testing.T, _ *fakeStore, w *fakeStatusWriter, res engine.Result, err error) {
				require.NoError(t, err)
				require.Zero(t, res)
				require.Len(t, w.reqs, 1)
			},
		},
		{
			name:       "not found from the API, ProjectConfig still there: requeued",
			projectUID: testProjectUID,
			store: func(t *testing.T) *fakeStore {
				return &fakeStore{cfg: newTestConfig(t, "a")}
			},
			status: &fakeStatusWriter{
				err: fmt.Errorf("wrapped: %w", &apiserver.APIError{StatusCode: http.StatusNotFound}),
			},
			assert: func(t *testing.T, _ *fakeStore, w *fakeStatusWriter, res engine.Result, err error) {
				require.NoError(t, err)
				require.Equal(t, notFoundRequeueDelay, res.RequeueAfter)
				require.Len(t, w.reqs, 1)
			},
		},
		{
			name:       "other error from the API",
			projectUID: testProjectUID,
			store: func(t *testing.T) *fakeStore {
				return &fakeStore{cfg: newTestConfig(t, "a")}
			},
			status: &fakeStatusWriter{
				err: &apiserver.APIError{StatusCode: http.StatusInternalServerError, Message: "boom"},
			},
			assert: func(t *testing.T, _ *fakeStore, _ *fakeStatusWriter, _ engine.Result, err error) {
				require.ErrorContains(t, err, "boom")
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			store := testCase.store(t)
			r := newTestReconciler(t, testCase.projectUID, store, testCase.status)
			res, err := r.Reconcile(t.Context(), key)
			testCase.assert(t, store, testCase.status, res, err)
		})
	}
}

// TestReconciler_Reconcile_unchangedStatus checks that a reconcile which
// computes the status already stored does not write it, and that a write
// keeps the transition time of a condition whose status did not change.
func TestReconciler_Reconcile_unchangedStatus(t *testing.T) {
	t.Parallel()
	key := projectconfigstore.Key{Project: testProject}
	// Receivers out of name order: the store serves them sorted by name.
	store := &fakeStore{cfg: newTestConfig(t, "b", "a")}
	w := &fakeStatusWriter{}
	r := newTestReconciler(t, testProjectUID, store, w)

	_, err := r.Reconcile(t.Context(), key)
	require.NoError(t, err)
	require.Len(t, w.reqs, 1)

	// Store what was written, with transition times an hour old, as a later
	// resync would find them.
	written := w.reqs[0]
	past := metav1.NewTime(time.Now().Add(-time.Hour).UTC().Truncate(time.Second))
	for i := range written.Conditions {
		written.Conditions[i].LastTransitionTime = past
	}
	applyStatus(t, store.cfg, written)
	slices.Reverse(store.cfg.Receivers)

	_, err = r.Reconcile(t.Context(), key)
	require.NoError(t, err)
	require.Len(t, w.reqs, 1, "an unchanged status must not be written")

	// A new generation is written, and keeps the transition time of the
	// unchanged Ready condition.
	store.cfg.Generation++
	_, err = r.Reconcile(t.Context(), key)
	require.NoError(t, err)
	require.Len(t, w.reqs, 2)
	ready := readyCondition(t, w.reqs[1])
	require.True(t, past.Equal(&ready.LastTransitionTime))
	require.Equal(t, store.cfg.Generation, ready.ObservedGeneration)
}

// TestReconciler_Reconcile_newRefreshToken checks that a refresh requested
// since the stored status was written is handled and recorded.
func TestReconciler_Reconcile_newRefreshToken(t *testing.T) {
	t.Parallel()
	key := projectconfigstore.Key{Project: testProject}
	store := &fakeStore{cfg: newTestConfig(t, "a")}
	w := &fakeStatusWriter{}
	r := newTestReconciler(t, testProjectUID, store, w)

	_, err := r.Reconcile(t.Context(), key)
	require.NoError(t, err)
	require.Len(t, w.reqs, 1)
	applyStatus(t, store.cfg, w.reqs[0])

	token := "newer-token"
	store.cfg.RefreshToken = &token
	_, err = r.Reconcile(t.Context(), key)
	require.NoError(t, err)
	require.Len(t, w.reqs, 2)
	require.Equal(t, token, w.reqs[1].ObservedRefreshToken)
}

// TestReconciler_Reconcile_conditionTransition checks that a condition whose
// status changes gets a new transition time.
func TestReconciler_Reconcile_conditionTransition(t *testing.T) {
	t.Parallel()
	key := projectconfigstore.Key{Project: testProject}
	store := &fakeStore{cfg: newTestConfig(t, "a")}
	past := metav1.NewTime(time.Now().Add(-time.Hour).UTC().Truncate(time.Second))
	applyStatus(t, store.cfg, wire.StatusRequest{
		ObservedGeneration: 1,
		Conditions: []metav1.Condition{{
			Type:               kargoapi.ConditionTypeReady,
			Status:             metav1.ConditionFalse,
			Reason:             "SyncWebhookReceiversFailed",
			LastTransitionTime: past,
			ObservedGeneration: 1,
		}},
	})
	w := &fakeStatusWriter{}
	r := newTestReconciler(t, testProjectUID, store, w)

	_, err := r.Reconcile(t.Context(), key)
	require.NoError(t, err)
	require.Len(t, w.reqs, 1)
	ready := readyCondition(t, w.reqs[0])
	require.Equal(t, metav1.ConditionTrue, ready.Status)
	require.True(t, ready.LastTransitionTime.After(past.Time))
}

func TestSpecChanged(t *testing.T) {
	t.Parallel()
	token := func(s string) *string { return &s }
	gen := int64(1)
	testCases := []struct {
		name     string
		old      database.ProjectConfigRow
		updated  database.ProjectConfigRow
		expected bool
	}{
		{
			name:     "generation changed",
			old:      database.ProjectConfigRow{Generation: 1},
			updated:  database.ProjectConfigRow{Generation: 2},
			expected: true,
		},
		{
			name:     "refresh requested",
			old:      database.ProjectConfigRow{Generation: 1},
			updated:  database.ProjectConfigRow{Generation: 1, RefreshToken: token("a")},
			expected: true,
		},
		{
			name:     "refresh token replaced",
			old:      database.ProjectConfigRow{Generation: 1, RefreshToken: token("a")},
			updated:  database.ProjectConfigRow{Generation: 1, RefreshToken: token("b")},
			expected: true,
		},
		{
			name: "status only",
			old:  database.ProjectConfigRow{Generation: 1, RefreshToken: token("a")},
			updated: database.ProjectConfigRow{
				Generation:               1,
				RefreshToken:             token("a"),
				StatusConditions:         json.RawMessage(`[{"type":"Ready"}]`),
				StatusObservedGeneration: &gen,
				StatusLastHandledRefresh: token("a"),
			},
			expected: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, testCase.expected, specChanged(testCase.old, testCase.updated))
		})
	}
}

func TestListKeys(t *testing.T) {
	t.Parallel()
	t.Run("lists every config", func(t *testing.T) {
		t.Parallel()
		store := &fakeStore{list: []projectconfigstore.Config{
			{ProjectConfigRow: database.ProjectConfigRow{ProjectName: "p1"}},
			{ProjectConfigRow: database.ProjectConfigRow{ProjectName: "p2"}},
		}}
		keys, err := listKeys(store)(t.Context())
		require.NoError(t, err)
		require.Equal(t, []projectconfigstore.Key{{Project: "p1"}, {Project: "p2"}}, keys)
	})
	t.Run("error", func(t *testing.T) {
		t.Parallel()
		store := &fakeStore{listErr: errors.New("boom")}
		_, err := listKeys(store)(t.Context())
		require.ErrorContains(t, err, "boom")
	})
}

// TestNewController checks that the controller assembles with its event
// source and its resync.
func TestNewController(t *testing.T) {
	t.Parallel()
	store := &fakeStore{}
	r := newTestReconciler(t, testProjectUID, store, &fakeStatusWriter{})
	c, err := newController(nil, store, r)
	require.NoError(t, err)
	require.Len(t, c.Sources, 2)
	require.Equal(t, r.cfg.MaxConcurrentReconciles, c.Workers)
}

// TestWithSkippedReceivers checks that reporting skipped receivers changes
// only Ready and the receivers, keeping every other condition.
func TestWithSkippedReceivers(t *testing.T) {
	t.Parallel()
	req := wire.StatusRequest{
		ObservedGeneration: 3,
		Conditions: []metav1.Condition{
			{
				Type:   kargoapi.ConditionTypeReconciling,
				Status: metav1.ConditionTrue,
				Reason: "SyncingWebhooks",
			},
			{
				Type:   kargoapi.ConditionTypeReady,
				Status: metav1.ConditionTrue,
				Reason: "Synced",
			},
		},
		WebhookReceivers: []kargoapi.WebhookReceiverDetails{
			{Name: "a"},
			{Name: "b"},
			{Name: "c"},
		},
	}
	got := withSkippedReceivers(req, []string{"a", "c"})
	require.Equal(t, []kargoapi.WebhookReceiverDetails{{Name: "b"}}, got.WebhookReceivers)
	require.Len(t, got.Conditions, 2)
	require.NotNil(t, meta.FindStatusCondition(got.Conditions, kargoapi.ConditionTypeReconciling))
	ready := readyCondition(t, got)
	require.Equal(t, metav1.ConditionFalse, ready.Status)
	require.Equal(t, int64(3), ready.ObservedGeneration)
	require.Equal(
		t,
		`webhook receiver "a": path is already claimed by another ProjectConfig; `+
			`webhook receiver "c": path is already claimed by another ProjectConfig`,
		ready.Message,
	)
	// The caller's request is left as it was.
	require.Len(t, req.WebhookReceivers, 3)
	require.Equal(t, metav1.ConditionTrue, readyCondition(t, req).Status)
}

// lostPathConfig returns a store whose ProjectConfig, with receivers a and
// b, has lost b's path to another Project and whose stored status already
// says so, as the fallback second write records it. It returns b's path.
func lostPathConfig(t *testing.T) (*fakeStore, string) {
	t.Helper()
	store := &fakeStore{cfg: newTestConfig(t, "a", "b")}
	w := &fakeStatusWriter{skipped: [][]string{{"b"}}}
	r := newTestReconciler(t, testProjectUID, store, w)
	_, err := r.Reconcile(t.Context(), projectconfigstore.Key{Project: testProject})
	require.NoError(t, err)
	require.Len(t, w.reqs, 2)
	var bPath string
	for _, receiver := range w.reqs[0].WebhookReceivers {
		if receiver.Name == "b" {
			bPath = receiver.Path
		}
	}
	require.NotEmpty(t, bPath)
	applyStatus(t, store.cfg, w.reqs[1])
	store.owners = map[string]string{bPath: "other-project"}
	return store, bPath
}

// TestReconciler_Reconcile_lostPathSteadyState checks that a ProjectConfig
// whose stored status already reports a path another Project owns is not
// written again while that Project still owns it.
func TestReconciler_Reconcile_lostPathSteadyState(t *testing.T) {
	t.Parallel()
	store, _ := lostPathConfig(t)
	w := &fakeStatusWriter{}
	r := newTestReconciler(t, testProjectUID, store, w)
	res, err := r.Reconcile(t.Context(), projectconfigstore.Key{Project: testProject})
	require.NoError(t, err)
	require.Zero(t, res)
	require.Empty(t, w.reqs, "a status reporting the lost path must not be rewritten")
}

// TestReconciler_Reconcile_lostPathPreSkipped checks that a path another
// Project owns is left out of the first write, with Ready False naming it.
func TestReconciler_Reconcile_lostPathPreSkipped(t *testing.T) {
	t.Parallel()
	store, bPath := lostPathConfig(t)
	// Forget the stored status, so a write is due.
	fresh := newTestConfig(t, "a", "b")
	store.cfg = fresh
	store.owners = map[string]string{bPath: "other-project"}
	w := &fakeStatusWriter{}
	r := newTestReconciler(t, testProjectUID, store, w)
	_, err := r.Reconcile(t.Context(), projectconfigstore.Key{Project: testProject})
	require.NoError(t, err)
	require.Len(t, w.reqs, 1)
	require.Len(t, w.reqs[0].WebhookReceivers, 1)
	require.Equal(t, "a", w.reqs[0].WebhookReceivers[0].Name)
	ready := readyCondition(t, w.reqs[0])
	require.Equal(t, metav1.ConditionFalse, ready.Status)
	require.Equal(
		t,
		`webhook receiver "b": path is already claimed by another ProjectConfig`,
		ready.Message,
	)
}

// TestReconciler_Reconcile_lostPathOwnedByItself checks that a path the
// ProjectConfig itself owns is not treated as lost.
func TestReconciler_Reconcile_lostPathOwnedByItself(t *testing.T) {
	t.Parallel()
	store, bPath := lostPathConfig(t)
	store.cfg = newTestConfig(t, "a", "b")
	store.owners = map[string]string{bPath: testProject}
	w := &fakeStatusWriter{}
	r := newTestReconciler(t, testProjectUID, store, w)
	_, err := r.Reconcile(t.Context(), projectconfigstore.Key{Project: testProject})
	require.NoError(t, err)
	require.Len(t, w.reqs, 1)
	require.Len(t, w.reqs[0].WebhookReceivers, 2)
	require.Equal(t, metav1.ConditionTrue, readyCondition(t, w.reqs[0]).Status)
}

// TestReconciler_Reconcile_lostPathRecovered checks that once the other
// Project releases the path, the next reconcile writes the receiver and
// Ready True.
func TestReconciler_Reconcile_lostPathRecovered(t *testing.T) {
	t.Parallel()
	store, _ := lostPathConfig(t)
	store.owners = nil
	w := &fakeStatusWriter{}
	r := newTestReconciler(t, testProjectUID, store, w)
	_, err := r.Reconcile(t.Context(), projectconfigstore.Key{Project: testProject})
	require.NoError(t, err)
	require.Len(t, w.reqs, 1)
	names := []string{}
	for _, receiver := range w.reqs[0].WebhookReceivers {
		names = append(names, receiver.Name)
	}
	require.ElementsMatch(t, []string{"a", "b"}, names)
	ready := readyCondition(t, w.reqs[0])
	require.Equal(t, metav1.ConditionTrue, ready.Status)
	require.Equal(t, "Synced", ready.Reason)
}

// TestReconciler_Reconcile_pathLookupError checks that a failure to look up
// a path's owner fails the reconcile rather than guessing.
func TestReconciler_Reconcile_pathLookupError(t *testing.T) {
	t.Parallel()
	store := &fakeStore{
		cfg:       newTestConfig(t, "a"),
		lookupErr: errors.New("lookup boom"),
	}
	w := &fakeStatusWriter{}
	r := newTestReconciler(t, testProjectUID, store, w)
	_, err := r.Reconcile(t.Context(), projectconfigstore.Key{Project: testProject})
	require.ErrorContains(t, err, "lookup boom")
	require.Empty(t, w.reqs)
}
