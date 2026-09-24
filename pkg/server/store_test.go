package server

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
)

// fakePromotionStore is an in-memory promotionStore. Target rows keep the
// Project's name in ProjectID. Every mutation advances a clock that serves as
// the rows' update time, so that versions change the way they would in the
// database.
type fakePromotionStore struct {
	mu          sync.Mutex
	clock       int64
	targets     []database.Target
	targetsErr  error
	requests    []database.PromotionRequestSnapshot
	requestsErr error
	createErr   error
	created     []database.PromotionRequestCreate
	// calls counts store reads and writes, so tests can prove that a refused
	// request never touched the store.
	calls int
}

func (s *fakePromotionStore) tick() time.Time {
	s.clock++
	return time.UnixMicro(s.clock)
}

func (s *fakePromotionStore) addTarget(project, name string, labels map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	encoded, err := json.Marshal(labels)
	if err != nil {
		panic(err)
	}
	now := s.tick()
	s.targets = append(s.targets, database.Target{
		ID:        uuid.New(),
		ProjectID: project,
		Name:      name,
		Labels:    encoded,
		Params:    []byte(`{}`),
		CreatedAt: now,
		UpdatedAt: now,
	})
}

func (s *fakePromotionStore) setTargetLabels(name string, labels map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	encoded, err := json.Marshal(labels)
	if err != nil {
		panic(err)
	}
	for i := range s.targets {
		if s.targets[i].Name == name {
			s.targets[i].Labels = encoded
			s.targets[i].UpdatedAt = s.tick()
		}
	}
}

func (s *fakePromotionStore) removeTarget(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.targets = slices.DeleteFunc(s.targets, func(t database.Target) bool { return t.Name == name })
}

func (s *fakePromotionStore) addRequest(
	project, stage, name, freight string,
	phase kargoapi.PromotionRequestPhase,
	targets ...string,
) database.PromotionRequestSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.tick()
	snapshot := database.PromotionRequestSnapshot{
		PromotionRequest: database.PromotionRequest{
			ID:        uuid.New(),
			Seq:       int64(len(s.requests) + 1),
			Name:      name,
			Phase:     string(phase),
			CreatedAt: now,
			UpdatedAt: now,
		},
		ProjectName: project,
		Stage:       stage,
		Freight:     freight,
	}
	for i, target := range targets {
		snapshot.Targets = append(snapshot.Targets, database.PromotionRequestTargetRow{
			PromotionRequestID: snapshot.ID,
			Name:               target,
			Ordinal:            int64(i),
		})
	}
	s.requests = append(s.requests, snapshot)
	return snapshot
}

func (s *fakePromotionStore) setRequestPhase(name string, phase kargoapi.PromotionRequestPhase) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.requests {
		if s.requests[i].Name == name {
			s.requests[i].Phase = string(phase)
			s.requests[i].UpdatedAt = s.tick()
		}
	}
}

func (s *fakePromotionStore) removeRequest(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = slices.DeleteFunc(s.requests, func(r database.PromotionRequestSnapshot) bool {
		return r.Name == name
	})
}

func (s *fakePromotionStore) ListTargets(_ context.Context, project string) ([]database.Target, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.targetsErr != nil {
		return nil, s.targetsErr
	}
	var out []database.Target
	for _, target := range s.targets {
		if target.ProjectID == project {
			out = append(out, target)
		}
	}
	slices.SortFunc(out, func(a, b database.Target) int { return strings.Compare(a.Name, b.Name) })
	return out, nil
}

func (s *fakePromotionStore) GetTarget(
	_ context.Context,
	params database.GetTargetParams,
) (database.Target, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.targetsErr != nil {
		return database.Target{}, s.targetsErr
	}
	for _, target := range s.targets {
		if target.ProjectID == params.ProjectName && target.Name == params.Name {
			return target, nil
		}
	}
	return database.Target{}, fmt.Errorf("Target %q: %w", params.Name, database.ErrNotFound)
}

func (s *fakePromotionStore) ListPromotionRequests(
	_ context.Context,
	project string,
) ([]database.PromotionRequestSnapshot, error) {
	return s.listRequests(project, "")
}

func (s *fakePromotionStore) ListPromotionRequestsByStage(
	_ context.Context,
	params database.ListPromotionRequestsByStageParams,
) ([]database.PromotionRequestSnapshot, error) {
	return s.listRequests(params.ProjectName, params.Stage)
}

func (s *fakePromotionStore) listRequests(project, stage string) ([]database.PromotionRequestSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.requestsErr != nil {
		return nil, s.requestsErr
	}
	var out []database.PromotionRequestSnapshot
	for _, request := range s.requests {
		if request.ProjectName == project && (stage == "" || request.Stage == stage) {
			out = append(out, request)
		}
	}
	slices.SortFunc(out, func(a, b database.PromotionRequestSnapshot) int { return cmp.Compare(a.Seq, b.Seq) })
	return out, nil
}

func (s *fakePromotionStore) GetPromotionRequest(
	_ context.Context,
	params database.GetPromotionRequestParams,
) (database.PromotionRequestSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.requestsErr != nil {
		return database.PromotionRequestSnapshot{}, s.requestsErr
	}
	for _, request := range s.requests {
		if request.ProjectName == params.ProjectName && request.Name == params.Name {
			return request, nil
		}
	}
	return database.PromotionRequestSnapshot{}, fmt.Errorf(
		"PromotionRequest %q: %w", params.Name, database.ErrNotFound,
	)
}

func (s *fakePromotionStore) CreatePromotionRequest(
	_ context.Context,
	create database.PromotionRequestCreate,
) (database.PromotionRequestSnapshot, error) {
	s.mu.Lock()
	s.calls++
	if s.createErr != nil {
		s.mu.Unlock()
		return database.PromotionRequestSnapshot{}, s.createErr
	}
	s.created = append(s.created, create)
	s.mu.Unlock()
	snapshot := s.addRequest(
		create.ProjectName, create.Stage, create.Name, create.Freight,
		kargoapi.PromotionRequestPhasePending, create.Targets...,
	)
	snapshot.CreatedBy = create.CreatedBy
	s.mu.Lock()
	s.requests[len(s.requests)-1].CreatedBy = create.CreatedBy
	s.mu.Unlock()
	return snapshot, nil
}

// withStore returns a serverSetup that installs the store and a fast poll.
func withStore(store *fakePromotionStore) func(*testing.T, *server) {
	return func(_ *testing.T, s *server) {
		s.store = store
		s.storePollInterval = 5 * time.Millisecond
	}
}

// authorizationCall records what a handler asked to authorize.
type authorizationCall struct {
	verb string
	gvr  schema.GroupVersionResource
	key  client.ObjectKey
}

// recordAuthorizations replaces the server's authorization with one that
// records every call and grants it.
func recordAuthorizations(calls *[]authorizationCall) func(*testing.T, *server) {
	return func(_ *testing.T, s *server) {
		s.authorizeFn = func(
			_ context.Context,
			verb string,
			gvr schema.GroupVersionResource,
			_ string,
			key client.ObjectKey,
		) error {
			*calls = append(*calls, authorizationCall{verb: verb, gvr: gvr, key: key})
			return nil
		}
	}
}

// forbidEverything replaces the server's authorization with one that refuses
// every call.
func forbidEverything(_ *testing.T, s *server) {
	s.authorizeFn = func(
		_ context.Context,
		_ string,
		gvr schema.GroupVersionResource,
		_ string,
		key client.ObjectKey,
	) error {
		return apierrors.NewForbidden(gvr.GroupResource(), key.Name, errors.New("not authorized"))
	}
}

// serverSetups combines several serverSetup functions.
func serverSetups(setups ...func(*testing.T, *server)) func(*testing.T, *server) {
	return func(t *testing.T, s *server) {
		for _, setup := range setups {
			setup(t, s)
		}
	}
}

// sseEvents decodes every SSE data event in the body as "<TYPE> <name>", in
// the order received.
func sseEvents[T any](t *testing.T, body string, name func(T) string) []string {
	t.Helper()
	var got []string
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		// The data of an error event is not a watch event.
		if i > 0 && strings.HasPrefix(lines[i-1], "event: ") {
			continue
		}
		e := WatchEvent[T]{}
		require.NoError(t, json.Unmarshal([]byte(line[len("data: "):]), &e))
		got = append(got, e.Type+" "+name(e.Object))
	}
	return got
}

func TestResourceVersions(t *testing.T) {
	t.Parallel()
	require.Equal(t, "", maxResourceVersion())
	require.Equal(t, "", maxResourceVersion("", "nope"))
	require.Equal(t, "30", maxResourceVersion("10", "30", "", "20"))

	_, ok := parseResourceVersion("")
	require.False(t, ok)
	_, ok = parseResourceVersion("abc")
	require.False(t, ok)
	v, ok := parseResourceVersion("42")
	require.True(t, ok)
	require.Equal(t, int64(42), v)
}

func TestAuthorizeStoreRead(t *testing.T) {
	t.Parallel()
	s := &server{}
	require.ErrorContains(
		t,
		s.authorizeStoreRead(context.Background(), "get", "targets", "demo", "x"),
		"authorize function is not configured",
	)
	var calls []authorizationCall
	recordAuthorizations(&calls)(t, s)
	require.NoError(t, s.authorizeStoreRead(context.Background(), "get", "targets", "demo", "x"))
	require.Equal(t, []authorizationCall{{
		verb: "get",
		gvr:  kargoapi.GroupVersion.WithResource("targets"),
		key:  client.ObjectKey{Namespace: "demo", Name: "x"},
	}}, calls)
}
