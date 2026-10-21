package promotions

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
)

func TestSync(t *testing.T) {
	t.Parallel()
	created := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	finished := created.Add(time.Minute)
	pending := testPromotion(created)
	pending.Status = kargoapi.PromotionStatus{}
	terminating := testPromotion(created)
	terminating.DeletionTimestamp = &metav1.Time{Time: created.Add(time.Hour)}
	noProject := testPromotion(created)
	noProject.Namespace = ""
	targeted := testPromotion(created)
	targeted.Spec.Target = "cluster-a"
	withCollection := testPromotion(created)
	withCollection.Status.FreightCollection = &kargoapi.FreightCollection{
		ID: "collection",
		Freight: map[string]kargoapi.FreightReference{
			"Warehouse/w2": {
				Name:   "abc123",
				Origin: kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "w2"},
				Images: []kargoapi.Image{{RepoURL: "example.com/app", Tag: "v1"}},
			},
			"Warehouse/w1": {
				Name:   "def456",
				Origin: kargoapi.FreightOrigin{Kind: kargoapi.FreightOriginKindWarehouse, Name: "w1"},
			},
		},
	}
	upserted := database.UpsertPromotionParams{
		ProjectName: "demo", Name: "dev.01abc.def456", StageName: "dev",
		FreightName: "def456", Phase: "Succeeded", CreatedAt: created, FinishedAt: &finished,
	}
	unfinished := upserted
	unfinished.Phase = ""
	unfinished.FinishedAt = nil
	targetName := "cluster-a"
	toTarget := upserted
	toTarget.TargetName = &targetName
	withFreight := upserted
	withFreight.OriginKinds = []string{"Warehouse", "Warehouse"}
	withFreight.OriginNames = []string{"w1", "w2"}
	withFreight.FreightNames = []string{"def456", "abc123"}
	noName := testPromotion(created)
	noName.Name = ""
	precise := testPromotion(created.Add(123456789 * time.Nanosecond))
	precise.Status.FinishedAt = &metav1.Time{Time: finished}
	truncated := upserted
	truncated.CreatedAt = created.Add(123456 * time.Microsecond)
	testCases := []struct {
		name      string
		obj       client.Object
		storeErr  error
		nameTaken bool
		wantErr   string
		want      *database.UpsertPromotionParams
	}{
		{name: "copies spec and status fields", obj: testPromotion(created), want: &upserted},
		{name: "a Promotion not yet reconciled has no phase or finish time", obj: pending, want: &unfinished},
		{name: "a Promotion to a Target names it", obj: targeted, want: &toTarget},
		{
			name: "a Promotion's collection is carried as names by origin, in origin order",
			obj:  withCollection, want: &withFreight,
		},
		{name: "preserves terminating objects", obj: terminating, want: &upserted},
		{name: "creation time is carried at PostgreSQL's precision", obj: precise, want: &truncated},
		{
			name: "storage failure propagates", obj: testPromotion(created),
			storeErr: errors.New("offline"), wantErr: "error syncing promotion", want: &upserted,
		},
		{
			name: "a name another Promotion was recorded under is an error", obj: testPromotion(created),
			nameTaken: true, wantErr: "has the name of an earlier Promotion", want: &upserted,
		},
		{name: "wrong object type", obj: &kargoapi.Stage{}, wantErr: "expected a Promotion"},
		{name: "missing name", obj: noName, wantErr: "incomplete identity"},
		{name: "missing namespace", obj: noProject, wantErr: "incomplete identity"},
		{name: "missing timestamp", obj: testPromotion(time.Time{}), wantErr: "incomplete identity"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			store := &fakeStore{writeErr: testCase.storeErr, nameTaken: testCase.nameTaken}
			s := NewSyncer(nil, store)
			require.IsType(t, &kargoapi.Promotion{}, s.NewObject())
			require.Len(t, s.Sources(nil), 1)
			for range 2 {
				err := s.Sync(context.Background(), testCase.obj)
				if testCase.wantErr != "" {
					require.ErrorContains(t, err, testCase.wantErr)
				} else {
					require.NoError(t, err)
				}
			}
			if testCase.want != nil {
				// Each sync offers a fresh id, minted from the creation time;
				// the store keeps a row's first.
				require.Len(t, store.upserts, 2)
				for i := range store.upserts {
					id, err := ulid.ParseStrict(store.upserts[i].ID)
					require.NoError(t, err)
					require.Equal(t, ulid.Timestamp(testCase.want.CreatedAt), id.Time())
					store.upserts[i].ID = ""
				}
				require.Equal(t, []database.UpsertPromotionParams{*testCase.want, *testCase.want}, store.upserts)
			} else {
				require.Empty(t, store.upserts)
			}
		})
	}
}

// TestDelete checks that a Promotion's row survives its deletion from
// Kubernetes: the store has no delete to call.
func TestDelete(t *testing.T) {
	t.Parallel()
	store := &fakeStore{writeErr: errors.New("offline")}
	s := NewSyncer(nil, store)
	err := s.Delete(context.Background(), client.ObjectKey{Namespace: "demo", Name: "dev.01abc.def456"})
	require.NoError(t, err)
	require.Empty(t, store.upserts)
	require.Zero(t, store.lists)
}

func testPromotion(created time.Time) *kargoapi.Promotion {
	return &kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "demo", Name: "dev.01abc.def456",
			CreationTimestamp: metav1.NewTime(created),
		},
		Spec: kargoapi.PromotionSpec{Stage: "dev", Freight: "def456"},
		Status: kargoapi.PromotionStatus{
			Phase:      kargoapi.PromotionPhaseSucceeded,
			FinishedAt: &metav1.Time{Time: created.Add(time.Minute)},
		},
	}
}

type fakeStore struct {
	rows      []database.PromotionRow
	upserts   []database.UpsertPromotionParams
	lists     [][]client.ObjectKey
	listErr   error
	writeErr  error
	nameTaken bool
}

func (s *fakeStore) UpsertPromotion(_ context.Context, params database.UpsertPromotionParams) (bool, error) {
	s.upserts = append(s.upserts, params)
	if s.writeErr != nil {
		return false, s.writeErr
	}
	return s.nameTaken, nil
}

// ListPromotionsByKey returns the rows whose keys are asked for, as the query
// does, and records what was asked.
func (s *fakeStore) ListPromotionsByKey(
	_ context.Context,
	params database.ListPromotionsByKeyParams,
) ([]database.PromotionRow, error) {
	keys := make([]client.ObjectKey, 0, len(params.Names))
	for i := range params.Names {
		keys = append(keys, client.ObjectKey{Namespace: params.ProjectNames[i], Name: params.Names[i]})
	}
	s.lists = append(s.lists, keys)
	var rows []database.PromotionRow
	for _, row := range s.rows {
		if slices.Contains(keys, rowKey(row)) {
			rows = append(rows, row)
		}
	}
	return rows, s.listErr
}
