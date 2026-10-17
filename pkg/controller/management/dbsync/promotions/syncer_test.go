package promotions

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
)

func TestSync(t *testing.T) {
	t.Parallel()
	created := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	finished := created.Add(time.Minute)
	pending := testPromotion("opaque-id", created)
	pending.Status = kargoapi.PromotionStatus{}
	terminating := testPromotion("opaque-id", created)
	terminating.DeletionTimestamp = &metav1.Time{Time: created.Add(time.Hour)}
	noProject := testPromotion("opaque-id", created)
	noProject.Namespace = ""
	targeted := testPromotion("opaque-id", created)
	targeted.Spec.Target = "cluster-a"
	upserted := database.UpsertPromotionParams{
		ID: "opaque-id", ProjectName: "demo", Name: "dev.01abc.def456", StageName: "dev",
		FreightName: "def456", Phase: "Succeeded", CreatedAt: created, FinishedAt: &finished,
	}
	unfinished := upserted
	unfinished.Phase = ""
	unfinished.FinishedAt = nil
	targetName := "cluster-a"
	toTarget := upserted
	toTarget.TargetName = &targetName
	testCases := []struct {
		name     string
		obj      client.Object
		storeErr error
		wantErr  string
		want     *database.UpsertPromotionParams
	}{
		{name: "copies spec and status fields", obj: testPromotion("opaque-id", created), want: &upserted},
		{name: "a Promotion not yet reconciled has no phase or finish time", obj: pending, want: &unfinished},
		{name: "a Promotion to a Target names it", obj: targeted, want: &toTarget},
		{name: "preserves terminating objects", obj: terminating, want: &upserted},
		{
			name: "storage failure propagates", obj: testPromotion("opaque-id", created),
			storeErr: errors.New("offline"), wantErr: "error syncing promotion", want: &upserted,
		},
		{name: "wrong object type", obj: &kargoapi.Stage{}, wantErr: "expected a Promotion"},
		{name: "missing UID", obj: testPromotion("", created), wantErr: "incomplete identity"},
		{name: "missing namespace", obj: noProject, wantErr: "incomplete identity"},
		{name: "missing timestamp", obj: testPromotion("opaque-id", time.Time{}), wantErr: "incomplete identity"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			store := &fakeStore{writeErr: testCase.storeErr}
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

func testPromotion(uid string, created time.Time) *kargoapi.Promotion {
	return &kargoapi.Promotion{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "demo", Name: "dev.01abc.def456", UID: types.UID(uid),
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
	rows     []database.PromotionRow
	upserts  []database.UpsertPromotionParams
	lists    [][]string
	listErr  error
	writeErr error
}

func (s *fakeStore) UpsertPromotion(_ context.Context, params database.UpsertPromotionParams) error {
	s.upserts = append(s.upserts, params)
	return s.writeErr
}

// ListPromotionsByID returns the rows whose IDs are asked for, as the query
// does, and records what was asked.
func (s *fakeStore) ListPromotionsByID(_ context.Context, ids []string) ([]database.PromotionRow, error) {
	s.lists = append(s.lists, slices.Clone(ids))
	var rows []database.PromotionRow
	for _, row := range s.rows {
		if slices.Contains(ids, row.ID) {
			rows = append(rows, row)
		}
	}
	return rows, s.listErr
}
