package promotions

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
)

func TestDiff(t *testing.T) {
	t.Parallel()
	created := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	finished := created.Add(time.Minute)
	live := testPromotion("live", created)
	row := database.PromotionRow{
		ID: "live", ProjectName: "demo", Name: "dev.01abc.def456", StageName: "dev",
		FreightName: "def456", Phase: "Succeeded", CreatedAt: created, FinishedAt: &finished,
	}
	differing := func(mutate func(*database.PromotionRow)) database.PromotionRow {
		changed := row
		mutate(&changed)
		return changed
	}
	oldSyncTime := differing(func(r *database.PromotionRow) { r.SyncedAt = created.Add(-24 * time.Hour) })
	targetName := "cluster-a"
	targeted := testPromotion("live", created)
	targeted.Spec.Target = targetName
	targetedRow := differing(func(r *database.PromotionRow) { r.TargetName = &targetName })
	history := database.PromotionRow{ID: "gone", ProjectName: "demo", Name: "dev.00old.abc123", StageName: "dev"}
	key := []reconcile.Request{{NamespacedName: client.ObjectKey{Namespace: "demo", Name: "dev.01abc.def456"}}}
	testCases := []struct {
		name        string
		promotion   *kargoapi.Promotion
		rows        []database.PromotionRow
		listErr     error
		snapshotErr error
		want        []reconcile.Request
		wantErr     string
		wantIDs     []string
	}{
		{name: "empty database", promotion: live, want: key, wantIDs: []string{"live"}},
		{
			name: "matching row is not queued", promotion: live,
			rows: []database.PromotionRow{row}, wantIDs: []string{"live"},
		},
		{
			name: "synced_at does not cause a diff", promotion: live,
			rows: []database.PromotionRow{oldSyncTime}, wantIDs: []string{"live"},
		},
		{
			name: "different project", promotion: live, want: key, wantIDs: []string{"live"},
			rows: []database.PromotionRow{differing(func(r *database.PromotionRow) { r.ProjectName = "wrong" })},
		},
		{
			name: "different name", promotion: live, want: key, wantIDs: []string{"live"},
			rows: []database.PromotionRow{differing(func(r *database.PromotionRow) { r.Name = "wrong" })},
		},
		{
			name: "different stage", promotion: live, want: key, wantIDs: []string{"live"},
			rows: []database.PromotionRow{differing(func(r *database.PromotionRow) { r.StageName = "wrong" })},
		},
		{
			name: "different freight", promotion: live, want: key, wantIDs: []string{"live"},
			rows: []database.PromotionRow{differing(func(r *database.PromotionRow) { r.FreightName = "wrong" })},
		},
		{
			name: "matching Target is not queued", promotion: targeted,
			rows: []database.PromotionRow{targetedRow}, wantIDs: []string{"live"},
		},
		{
			name: "different Target", promotion: targeted, want: key, wantIDs: []string{"live"},
			rows: []database.PromotionRow{differing(func(r *database.PromotionRow) { r.TargetName = new("wrong") })},
		},
		{
			name: "Target gained", promotion: targeted, rows: []database.PromotionRow{row},
			want: key, wantIDs: []string{"live"},
		},
		{
			name: "Target lost", promotion: live, rows: []database.PromotionRow{targetedRow},
			want: key, wantIDs: []string{"live"},
		},
		{
			name: "different phase", promotion: live, want: key, wantIDs: []string{"live"},
			rows: []database.PromotionRow{differing(func(r *database.PromotionRow) { r.Phase = "Running" })},
		},
		{
			name: "different creation timestamp", promotion: live, want: key, wantIDs: []string{"live"},
			rows: []database.PromotionRow{
				differing(func(r *database.PromotionRow) { r.CreatedAt = created.Add(-time.Hour) }),
			},
		},
		{
			name: "unfinished row for a finished Promotion", promotion: live, want: key, wantIDs: []string{"live"},
			rows: []database.PromotionRow{differing(func(r *database.PromotionRow) { r.FinishedAt = nil })},
		},
		// A row whose Promotion is gone is history, not an orphan: it is
		// neither asked for nor returned.
		{name: "deleted Promotions are kept", rows: []database.PromotionRow{history}, wantIDs: []string{}},
		{
			name: "history beside a live Promotion", promotion: live,
			rows: []database.PromotionRow{history, row}, wantIDs: []string{"live"},
		},
		{
			name: "failed database snapshot", promotion: live, snapshotErr: errors.New("offline"),
			wantErr: "listing mirrored promotions", wantIDs: []string{"live"},
		},
		{
			name: "failed Kubernetes list", rows: []database.PromotionRow{row},
			listErr: errors.New("offline"), wantErr: "listing promotions",
		},
		{
			name: "invalid identity", promotion: testPromotion("", created),
			rows: []database.PromotionRow{row}, wantErr: "has no UID", wantIDs: []string{""},
		},
		{
			name: "invalid timestamp", promotion: testPromotion("live", time.Time{}),
			rows: []database.PromotionRow{row}, wantErr: "incomplete identity", wantIDs: []string{"live"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			scheme := runtime.NewScheme()
			require.NoError(t, kargoapi.AddToScheme(scheme))
			builder := fake.NewClientBuilder().WithScheme(scheme)
			if testCase.promotion != nil {
				builder.WithObjects(testCase.promotion.DeepCopy())
			}
			store := &fakeStore{rows: slices.Clone(testCase.rows), listErr: testCase.snapshotErr}
			reader := &diffReader{Reader: builder.Build(), err: testCase.listErr}
			changes, err := NewSyncer(reader, store).Diff(context.Background())
			if testCase.wantErr != "" {
				require.ErrorContains(t, err, testCase.wantErr)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, testCase.want, changes)
			// The snapshot asks only for the rows of live Promotions.
			if testCase.wantIDs == nil {
				require.Empty(t, store.lists)
			} else {
				require.Equal(t, [][]string{testCase.wantIDs}, store.lists)
			}
			require.Empty(t, store.upserts)
		})
	}
}

func TestMatchesTimestampPrecision(t *testing.T) {
	t.Parallel()
	created := time.Date(2026, 10, 15, 12, 0, 0, 123456789, time.UTC)
	promotion := testPromotion("live", created)
	promotion.Status.FinishedAt.Time = created.Add(time.Minute)
	// Test the matcher directly: the Kubernetes fake client's JSON round trip
	// itself removes subsecond precision from metav1.Time.
	finished := created.Add(time.Minute).Truncate(time.Microsecond)
	row := database.PromotionRow{
		ID: "live", ProjectName: "demo", Name: "dev.01abc.def456", StageName: "dev", FreightName: "def456",
		Phase: "Succeeded", CreatedAt: created.Truncate(time.Microsecond), FinishedAt: &finished,
	}
	match, err := matches(promotion, row)
	require.NoError(t, err)
	require.True(t, match)
	later := finished.Add(time.Microsecond)
	row.FinishedAt = &later
	match, err = matches(promotion, row)
	require.NoError(t, err)
	require.False(t, match)
	row.FinishedAt = &finished
	row.CreatedAt = row.CreatedAt.Add(time.Microsecond)
	match, err = matches(promotion, row)
	require.NoError(t, err)
	require.False(t, match)
}

type diffReader struct {
	client.Reader
	err error
}

func (r *diffReader) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	if r.err != nil {
		return r.err
	}
	return r.Reader.List(ctx, list, opts...)
}
