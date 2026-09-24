package stages

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
)

// fakePromotionRequestStore is an in-memory promotionRequestStore.
type fakePromotionRequestStore struct {
	targets    []database.Target
	targetsErr error
	requests   []database.PromotionRequestSnapshot
	listErr    error
	getErr     error
	existsErr  error
	createErr  error
	created    []database.PromotionRequestCreate
}

func (s *fakePromotionRequestStore) ListTargets(context.Context, string) ([]database.Target, error) {
	return s.targets, s.targetsErr
}

func (s *fakePromotionRequestStore) ListPromotionRequestsByStage(
	_ context.Context,
	params database.ListPromotionRequestsByStageParams,
) ([]database.PromotionRequestSnapshot, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	var out []database.PromotionRequestSnapshot
	for _, request := range s.requests {
		if request.ProjectName == params.ProjectName && request.Stage == params.Stage {
			out = append(out, request)
		}
	}
	slices.SortFunc(out, func(a, b database.PromotionRequestSnapshot) int {
		return cmp.Compare(a.Seq, b.Seq)
	})
	return out, nil
}

func (s *fakePromotionRequestStore) GetPromotionRequest(
	_ context.Context,
	params database.GetPromotionRequestParams,
) (database.PromotionRequestSnapshot, error) {
	if s.getErr != nil {
		return database.PromotionRequestSnapshot{}, s.getErr
	}
	for _, request := range s.requests {
		if request.ProjectName == params.ProjectName && request.Name == params.Name {
			return request, nil
		}
	}
	return database.PromotionRequestSnapshot{}, fmt.Errorf("PromotionRequest %q: %w", params.Name, database.ErrNotFound)
}

func (s *fakePromotionRequestStore) PromotionRequestExists(
	_ context.Context,
	params database.PromotionRequestExistsParams,
) (bool, error) {
	if s.existsErr != nil {
		return false, s.existsErr
	}
	for _, request := range s.requests {
		if request.ProjectName == params.ProjectName &&
			request.Stage == params.Stage &&
			request.Freight == params.Freight {
			return true, nil
		}
	}
	return false, nil
}

func (s *fakePromotionRequestStore) CreatePromotionRequest(
	_ context.Context,
	create database.PromotionRequestCreate,
) (database.PromotionRequestSnapshot, error) {
	if s.createErr != nil {
		return database.PromotionRequestSnapshot{}, s.createErr
	}
	s.created = append(s.created, create)
	snapshot := database.PromotionRequestSnapshot{
		PromotionRequest: database.PromotionRequest{
			ID:        uuid.New(),
			Seq:       int64(len(s.requests) + 1),
			Name:      create.Name,
			CreatedBy: create.CreatedBy,
			Phase:     string(kargoapi.PromotionRequestPhasePending),
		},
		ProjectName: create.ProjectName,
		Stage:       create.Stage,
		Freight:     create.Freight,
	}
	for i, name := range create.Targets {
		snapshot.Targets = append(snapshot.Targets, database.PromotionRequestTargetRow{
			PromotionRequestID: snapshot.ID,
			Name:               name,
			Ordinal:            int64(i),
		})
	}
	s.requests = append(s.requests, snapshot)
	return snapshot, nil
}

// testTargetRow builds a Target row with the given labels.
func testTargetRow(name string, labels map[string]string) database.Target {
	encoded, err := json.Marshal(labels)
	if err != nil {
		panic(err)
	}
	return database.Target{
		ID:     uuid.New(),
		Name:   name,
		Labels: encoded,
		Params: []byte(`{}`),
	}
}

// testPromotionRequest builds a snapshot of a PromotionRequest of test-stage in
// fake-project. Only seq expresses creation order; names are deliberately
// meaningless.
func testPromotionRequest(
	seq int64,
	name string,
	freight string,
	phase kargoapi.PromotionRequestPhase,
	finishedAt *metav1.Time,
) database.PromotionRequestSnapshot {
	snapshot := database.PromotionRequestSnapshot{
		PromotionRequest: database.PromotionRequest{
			ID:    uuid.New(),
			Seq:   seq,
			Name:  name,
			Phase: string(phase),
		},
		ProjectName: "fake-project",
		Stage:       "test-stage",
		Freight:     freight,
	}
	if finishedAt != nil {
		snapshot.FinishedAt = pgtype.Timestamptz{Time: finishedAt.Time, Valid: true}
	}
	return snapshot
}
