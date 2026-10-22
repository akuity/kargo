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
	live := testPromotion(created)
	row := database.PromotionRow{
		ProjectName: "demo", Name: "dev.01abc.def456", StageName: "dev",
		FreightName: "def456", Phase: "Succeeded", CreatedAt: created, FinishedAt: &finished,
	}
	differing := func(mutate func(*database.PromotionRow)) database.PromotionRow {
		changed := row
		mutate(&changed)
		return changed
	}
	oldSyncTime := differing(func(r *database.PromotionRow) { r.SyncedAt = created.Add(-24 * time.Hour) })
	targetName := "cluster-a"
	targeted := testPromotion(created)
	targeted.Spec.Target = targetName
	targetedRow := differing(func(r *database.PromotionRow) { r.TargetName = &targetName })
	history := database.PromotionRow{
		ProjectName: "demo", Name: "dev.00old.abc123", StageName: "dev", Phase: "Succeeded", FinishedAt: &finished,
	}
	// A row without a finish time whose Promotion is gone: deleted before it
	// finished, and the mirror never saw it go.
	abandoned := database.PromotionRow{ProjectName: "demo", Name: "dev.00gone.abc123", StageName: "dev", Phase: "Running"}
	abandonedKey := reconcile.Request{NamespacedName: client.ObjectKey{Namespace: "demo", Name: abandoned.Name}}
	liveKey := client.ObjectKey{Namespace: "demo", Name: "dev.01abc.def456"}
	key := []reconcile.Request{{NamespacedName: liveKey}}
	asked := []client.ObjectKey{liveKey}
	testCases := []struct {
		name          string
		promotion     *kargoapi.Promotion
		rows          []database.PromotionRow
		listErr       error
		snapshotErr   error
		unfinishedErr error
		want          []reconcile.Request
		wantErr       string
		wantKeys      []client.ObjectKey
	}{
		{name: "empty database", promotion: live, want: key, wantKeys: asked},
		{
			name: "matching row is not queued", promotion: live,
			rows: []database.PromotionRow{row}, wantKeys: asked,
		},
		{
			name: "synced_at does not cause a diff", promotion: live,
			rows: []database.PromotionRow{oldSyncTime}, wantKeys: asked,
		},
		{
			// A row under another name is another Promotion's.
			name: "row under another name", promotion: live, want: key, wantKeys: asked,
			rows: []database.PromotionRow{differing(func(r *database.PromotionRow) { r.Name = "other" })},
		},
		{
			name: "different stage", promotion: live, want: key, wantKeys: asked,
			rows: []database.PromotionRow{differing(func(r *database.PromotionRow) { r.StageName = "wrong" })},
		},
		{
			name: "different freight", promotion: live, want: key, wantKeys: asked,
			rows: []database.PromotionRow{differing(func(r *database.PromotionRow) { r.FreightName = "wrong" })},
		},
		{
			name: "matching Target is not queued", promotion: targeted,
			rows: []database.PromotionRow{targetedRow}, wantKeys: asked,
		},
		{
			name: "different Target", promotion: targeted, want: key, wantKeys: asked,
			rows: []database.PromotionRow{differing(func(r *database.PromotionRow) { r.TargetName = new("wrong") })},
		},
		{
			name: "Target gained", promotion: targeted, rows: []database.PromotionRow{row},
			want: key, wantKeys: asked,
		},
		{
			name: "Target lost", promotion: live, rows: []database.PromotionRow{targetedRow},
			want: key, wantKeys: asked,
		},
		{
			name: "different phase", promotion: live, want: key, wantKeys: asked,
			rows: []database.PromotionRow{differing(func(r *database.PromotionRow) { r.Phase = "Running" })},
		},
		{
			// An earlier Promotion's row under the same name. Syncing the key
			// reports the conflict.
			name: "different creation timestamp", promotion: live, want: key, wantKeys: asked,
			rows: []database.PromotionRow{
				differing(func(r *database.PromotionRow) { r.CreatedAt = created.Add(-time.Hour) }),
			},
		},
		{
			name: "unfinished row for a finished Promotion", promotion: live, want: key, wantKeys: asked,
			rows: []database.PromotionRow{differing(func(r *database.PromotionRow) { r.FinishedAt = nil })},
		},
		// A finished row whose Promotion is gone is history, not an orphan:
		// it is neither asked for nor returned.
		{name: "deleted Promotions are kept", rows: []database.PromotionRow{history}, wantKeys: []client.ObjectKey{}},
		{
			name: "history beside a live Promotion", promotion: live,
			rows: []database.PromotionRow{history, row}, wantKeys: asked,
		},
		// An unfinished row whose Promotion is gone is queued, after the live
		// keys, so that Delete records the Promotion as finished.
		{
			name: "an unfinished row without a Promotion is queued",
			rows: []database.PromotionRow{history, abandoned}, want: []reconcile.Request{abandonedKey},
			wantKeys: []client.ObjectKey{},
		},
		{
			name: "an unfinished row without a Promotion is queued after live keys", promotion: live,
			rows: []database.PromotionRow{abandoned, history}, want: append(key, abandonedKey), wantKeys: asked,
		},
		{
			// The live Promotion's own unfinished row is not an orphan.
			name: "an unfinished row of a live Promotion is compared, not queued", promotion: live,
			rows: []database.PromotionRow{differing(func(r *database.PromotionRow) { r.FinishedAt = nil })},
			want: key, wantKeys: asked,
		},
		{
			name: "failed database snapshot", promotion: live, snapshotErr: errors.New("offline"),
			wantErr: "listing mirrored promotions", wantKeys: asked,
		},
		{
			name: "failed unfinished snapshot", promotion: live, unfinishedErr: errors.New("offline"),
			wantErr: "listing unfinished mirrored promotions", wantKeys: asked,
		},
		{
			name: "failed Kubernetes list", rows: []database.PromotionRow{row},
			listErr: errors.New("offline"), wantErr: "listing promotions",
		},
		{
			name: "invalid timestamp", promotion: testPromotion(time.Time{}),
			rows: []database.PromotionRow{row}, wantErr: "incomplete identity", wantKeys: asked,
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
			store := &fakeStore{
				rows: slices.Clone(testCase.rows), listErr: testCase.snapshotErr, unfinishedErr: testCase.unfinishedErr,
			}
			reader := &diffReader{Reader: builder.Build(), err: testCase.listErr}
			changes, err := NewSyncer(reader, store).Diff(context.Background())
			if testCase.wantErr != "" {
				require.ErrorContains(t, err, testCase.wantErr)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, testCase.want, changes)
			// The snapshot asks only for the rows of live Promotions.
			if testCase.wantKeys == nil {
				require.Empty(t, store.lists)
			} else {
				require.Equal(t, [][]client.ObjectKey{testCase.wantKeys}, store.lists)
			}
			require.Empty(t, store.upserts)
		})
	}
}

func TestMatchesTimestampPrecision(t *testing.T) {
	t.Parallel()
	created := time.Date(2026, 10, 15, 12, 0, 0, 123456789, time.UTC)
	promotion := testPromotion(created)
	promotion.Status.FinishedAt.Time = created.Add(time.Minute)
	// Test the matcher directly: the Kubernetes fake client's JSON round trip
	// itself removes subsecond precision from metav1.Time.
	finished := created.Add(time.Minute).Truncate(time.Microsecond)
	row := database.PromotionRow{
		ProjectName: "demo", Name: "dev.01abc.def456", StageName: "dev", FreightName: "def456",
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
