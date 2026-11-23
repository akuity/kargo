//go:build integration && db

package projectconfigstore

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/database/databasetest"
	"github.com/akuity/kargo/pkg/reconciler/nats"
)

type recordedEvent struct {
	subject string
	event   nats.Event[Key, database.ProjectConfigRow]
}

// recorder is a nats.Publisher that decodes and keeps what is published. A
// hook, if set, runs inside Publish before the event is kept.
type recorder struct {
	mu     sync.Mutex
	events []recordedEvent
	hook   func(recordedEvent)
}

func (r *recorder) Publish(subject string, data []byte) error {
	var e nats.Event[Key, database.ProjectConfigRow]
	if err := json.Unmarshal(data, &e); err != nil {
		return err
	}
	rec := recordedEvent{subject: subject, event: e}
	if r.hook != nil {
		r.hook(rec)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, rec)
	return nil
}

func (r *recorder) take() []recordedEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	events := r.events
	r.events = nil
	return events
}

func TestStoreIntegration(t *testing.T) {
	t.Parallel()
	pool, _ := databasetest.IsolatedDatabase(t)
	queries := database.New(pool)
	ctx := context.Background()

	newStore := func() (Store, *recorder) {
		rec := &recorder{}
		return New(pool, rec), rec
	}
	newCR := func(project string) *kargoapi.ProjectConfig {
		return &kargoapi.ProjectConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:        project,
				Namespace:   project,
				Labels:      map[string]string{"team": "a"},
				Annotations: map[string]string{"note": "x"},
			},
			Spec: kargoapi.ProjectConfigSpec{
				PromotionPolicies: []kargoapi.PromotionPolicy{{
					Stage:                "dev",
					AutoPromotionEnabled: true,
				}},
				WebhookReceivers: []kargoapi.WebhookReceiverConfig{{
					Name:   "gh",
					GitHub: &kargoapi.GitHubWebhookReceiverConfig{SecretRef: corev1.LocalObjectReference{Name: "s"}},
				}},
			},
		}
	}
	mirror := func(t *testing.T, project, uid string) {
		t.Helper()
		require.NoError(t, queries.UpsertProject(ctx, database.UpsertProjectParams{
			ID:        uid,
			Name:      project,
			CreatedAt: time.Now(),
		}))
	}
	ready := []metav1.Condition{{
		Type:               kargoapi.ConditionTypeReady,
		Status:             metav1.ConditionTrue,
		Reason:             "Synced",
		LastTransitionTime: metav1.NewTime(time.Now().Truncate(time.Second)),
	}}
	receiver := func(name, path string) kargoapi.WebhookReceiverDetails {
		return kargoapi.WebhookReceiverDetails{Name: name, Path: path, URL: "https://x" + path}
	}
	requireKinds := func(t *testing.T, events []recordedEvent, kinds ...nats.EventKind) {
		t.Helper()
		got := make([]nats.EventKind, len(events))
		for i, e := range events {
			got[i] = e.event.Kind
		}
		require.Equal(t, kinds, got)
	}

	t.Run("create then get", func(t *testing.T) {
		t.Parallel()
		store, rec := newStore()
		cr := newCR("create-get")
		cr.Annotations[kargoapi.AnnotationKeyRefresh] = "client-token"
		created, err := store.Create(ctx, cr, "uid-1")
		require.NoError(t, err)
		require.Equal(t, "create-get", created.ProjectName)
		require.Equal(t, "uid-1", created.ProjectUID)
		require.Equal(t, int64(1), created.Generation)
		require.Equal(t, map[string]string{"note": "x"}, created.Annotations)
		require.Nil(t, created.RefreshToken)

		got, err := store.Get(ctx, "create-get", "uid-1")
		require.NoError(t, err)
		require.Equal(t, created.ID, got.ID)
		require.Equal(t, cr.Spec, ToAPI(got).Spec)

		_, err = store.Get(ctx, "create-get", "other-uid")
		require.ErrorIs(t, err, database.ErrNotFound)
		_, err = store.Get(ctx, "no-such-project", "uid-1")
		require.ErrorIs(t, err, database.ErrNotFound)

		events := rec.take()
		requireKinds(t, events, nats.Created)
		require.Equal(t, "kargo.internal.projectconfigs.created", events[0].subject)
		require.Equal(t, Key{Project: "create-get"}, events[0].event.Key)
		require.Equal(t, created.ID, events[0].event.New.ID)
	})

	t.Run("create twice", func(t *testing.T) {
		t.Parallel()
		store, rec := newStore()
		_, err := store.Create(ctx, newCR("create-twice"), "uid-1")
		require.NoError(t, err)
		_, err = store.Create(ctx, newCR("create-twice"), "uid-1")
		require.ErrorIs(t, err, database.ErrAlreadyExists)
		requireKinds(t, rec.take(), nats.Created)
	})

	t.Run("create over a leftover replaces it", func(t *testing.T) {
		t.Parallel()
		store, rec := newStore()
		old, err := store.Create(ctx, newCR("leftover"), "old-uid")
		require.NoError(t, err)
		skipped, err := store.UpdateStatus(ctx, "leftover", Status{
			ObservedID:         old.ID,
			ObservedGeneration: 1,
			WebhookReceivers:   []kargoapi.WebhookReceiverDetails{receiver("gh", "/leftover")},
		})
		require.NoError(t, err)
		require.Empty(t, skipped)
		rec.take()

		created, err := store.Create(ctx, newCR("leftover"), "new-uid")
		require.NoError(t, err)
		require.NotEqual(t, old.ID, created.ID)
		require.Equal(t, "new-uid", created.ProjectUID)
		require.Empty(t, created.Receivers)

		got, err := store.GetUnfiltered(ctx, "leftover")
		require.NoError(t, err)
		require.Equal(t, created.ID, got.ID)
		// The old row's receivers went with it, so the path is free.
		_, err = store.GetByWebhookReceiverPath(ctx, "/leftover")
		require.ErrorIs(t, err, database.ErrNotFound)

		events := rec.take()
		requireKinds(t, events, nats.Deleted, nats.Created)
		require.Equal(t, old.ID, events[0].event.Old.ID)
		require.Equal(t, created.ID, events[1].event.New.ID)
	})

	t.Run("update", func(t *testing.T) {
		t.Parallel()
		store, rec := newStore()
		created, err := store.Create(ctx, newCR("update"), "uid-1")
		require.NoError(t, err)
		staleRV := ToAPI(created).ResourceVersion
		rec.take()

		// The same spec rewritten leaves the generation alone.
		cr := newCR("update")
		cr.Labels = map[string]string{"team": "b"}
		cr.Annotations[kargoapi.AnnotationKeyRefresh] = "client-token"
		cr.ResourceVersion = staleRV
		cr.UID = types.UID(created.ID.String())
		updated, err := store.Update(ctx, cr, "uid-1")
		require.NoError(t, err)
		require.Equal(t, int64(1), updated.Generation)
		require.Equal(t, map[string]string{"team": "b"}, updated.Labels)
		require.Equal(t, map[string]string{"note": "x"}, updated.Annotations)
		require.True(t, updated.UpdatedAt.After(created.UpdatedAt))

		events := rec.take()
		requireKinds(t, events, nats.Updated)
		require.Equal(t, map[string]string{"team": "a"}, events[0].event.Old.Labels)
		require.Equal(t, map[string]string{"team": "b"}, events[0].event.New.Labels)

		// A changed spec bumps it.
		cr = newCR("update")
		cr.Spec.PromotionPolicies[0].AutoPromotionEnabled = false
		updated, err = store.Update(ctx, cr, "uid-1")
		require.NoError(t, err)
		require.Equal(t, int64(2), updated.Generation)
		rec.take()

		cr = newCR("update")
		cr.ResourceVersion = staleRV
		_, err = store.Update(ctx, cr, "uid-1")
		require.ErrorIs(t, err, database.ErrConflict)

		cr = newCR("update")
		cr.ResourceVersion = "not-a-version"
		_, err = store.Update(ctx, cr, "uid-1")
		require.ErrorIs(t, err, database.ErrInvalid)

		cr = newCR("update")
		cr.UID = types.UID(uuid.NewString())
		_, err = store.Update(ctx, cr, "uid-1")
		require.ErrorIs(t, err, database.ErrConflict)

		_, err = store.Update(ctx, newCR("update"), "other-uid")
		require.ErrorIs(t, err, database.ErrNotFound)
		_, err = store.Update(ctx, newCR("no-such-project"), "uid-1")
		require.ErrorIs(t, err, database.ErrNotFound)
		require.Empty(t, rec.take())
	})

	t.Run("update status", func(t *testing.T) {
		t.Parallel()
		store, rec := newStore()
		other, err := store.Create(ctx, newCR("status-other"), "uid-o")
		require.NoError(t, err)
		_, err = store.UpdateStatus(ctx, "status-other", Status{
			ObservedID:         other.ID,
			ObservedGeneration: 1,
			WebhookReceivers:   []kargoapi.WebhookReceiverDetails{receiver("taken", "/status-shared")},
		})
		require.NoError(t, err)
		created, err := store.Create(ctx, newCR("status"), "uid-1")
		require.NoError(t, err)
		rec.take()

		skipped, err := store.UpdateStatus(ctx, "status", Status{
			ObservedID:         created.ID,
			ObservedGeneration: 1,
			Conditions:         ready,
			WebhookReceivers: []kargoapi.WebhookReceiverDetails{
				receiver("collides", "/status-shared"),
				receiver("mine", "/status-mine"),
			},
		})
		require.NoError(t, err)
		require.Equal(t, []string{"collides"}, skipped)

		got, err := store.Get(ctx, "status", "uid-1")
		require.NoError(t, err)
		status := ToAPI(got).Status
		require.Equal(t, int64(1), status.ObservedGeneration)
		require.Len(t, status.Conditions, 1)
		require.Equal(t, "Synced", status.Conditions[0].Reason)
		require.Equal(t, []kargoapi.WebhookReceiverDetails{receiver("mine", "/status-mine")}, status.WebhookReceivers)
		events := rec.take()
		requireKinds(t, events, nats.Updated)
		require.Nil(t, events[0].event.Old.StatusObservedGeneration)
		require.Equal(t, int64(1), *events[0].event.New.StatusObservedGeneration)

		// A later status replaces the receivers.
		_, err = store.UpdateStatus(ctx, "status", Status{
			ObservedID:         created.ID,
			ObservedGeneration: 1,
			WebhookReceivers:   []kargoapi.WebhookReceiverDetails{receiver("next", "/status-next")},
		})
		require.NoError(t, err)
		got, err = store.Get(ctx, "status", "uid-1")
		require.NoError(t, err)
		require.Equal(
			t,
			[]database.ProjectConfigWebhookReceiverRow{{
				ProjectConfigID: created.ID,
				Name:            "next",
				Path:            "/status-next",
				URL:             "https://x/status-next",
			}},
			got.Receivers,
		)
		rec.take()

		// A newer generation than observed.
		cr := newCR("status")
		cr.Spec.PromotionPolicies = nil
		_, err = store.Update(ctx, cr, "uid-1")
		require.NoError(t, err)
		_, err = store.UpdateStatus(ctx, "status", Status{ObservedID: created.ID, ObservedGeneration: 1})
		require.ErrorIs(t, err, database.ErrConflict)

		// A newer refresh token than observed.
		token, err := store.RequestRefresh(ctx, "status", "uid-1")
		require.NoError(t, err)
		_, err = store.UpdateStatus(ctx, "status", Status{ObservedID: created.ID, ObservedGeneration: 2})
		require.ErrorIs(t, err, database.ErrConflict)
		_, err = store.UpdateStatus(ctx, "status", Status{
			ObservedID:           created.ID,
			ObservedGeneration:   2,
			ObservedRefreshToken: "older",
		})
		require.ErrorIs(t, err, database.ErrConflict)
		rec.take()

		_, err = store.UpdateStatus(ctx, "status", Status{
			ObservedID:           created.ID,
			ObservedGeneration:   2,
			ObservedRefreshToken: token,
		})
		require.NoError(t, err)
		got, err = store.Get(ctx, "status", "uid-1")
		require.NoError(t, err)
		require.Equal(t, token, ToAPI(got).Status.LastHandledRefresh)
		// The last status had no receivers, so none remain.
		require.Equal(t, []kargoapi.WebhookReceiverDetails(nil), ToAPI(got).Status.WebhookReceivers)
		requireKinds(t, rec.take(), nats.Updated)

		_, err = store.UpdateStatus(ctx, "no-such-project", Status{ObservedID: created.ID, ObservedGeneration: 1})
		require.ErrorIs(t, err, database.ErrNotFound)
	})

	t.Run("rejected status write leaves receivers alone", func(t *testing.T) {
		t.Parallel()
		store, _ := newStore()
		created, err := store.Create(ctx, newCR("status-rejected"), "uid-1")
		require.NoError(t, err)
		_, err = store.UpdateStatus(ctx, "status-rejected", Status{
			ObservedID:         created.ID,
			ObservedGeneration: 1,
			WebhookReceivers:   []kargoapi.WebhookReceiverDetails{receiver("a", "/status-rejected-a")},
		})
		require.NoError(t, err)
		_, err = store.UpdateStatus(ctx, "status-rejected", Status{
			ObservedID:         created.ID,
			ObservedGeneration: 0,
			WebhookReceivers:   []kargoapi.WebhookReceiverDetails{receiver("b", "/status-rejected-b")},
		})
		require.ErrorIs(t, err, database.ErrConflict)
		got, err := store.Get(ctx, "status-rejected", "uid-1")
		require.NoError(t, err)
		require.Equal(
			t,
			[]kargoapi.WebhookReceiverDetails{receiver("a", "/status-rejected-a")},
			ToAPI(got).Status.WebhookReceivers,
		)
	})

	t.Run("duplicate receiver names in a status", func(t *testing.T) {
		t.Parallel()
		store, _ := newStore()
		created, err := store.Create(ctx, newCR("status-dup"), "uid-1")
		require.NoError(t, err)
		skipped, err := store.UpdateStatus(ctx, "status-dup", Status{
			ObservedID:         created.ID,
			ObservedGeneration: 1,
			Conditions:         ready,
			WebhookReceivers: []kargoapi.WebhookReceiverDetails{
				receiver("dup", "/status-dup-1"),
				receiver("dup", "/status-dup-2"),
				receiver("other", "/status-dup-3"),
			},
		})
		require.NoError(t, err)
		require.Equal(t, []string{"dup"}, skipped)
		got, err := store.Get(ctx, "status-dup", "uid-1")
		require.NoError(t, err)
		status := ToAPI(got).Status
		require.Equal(t, int64(1), status.ObservedGeneration)
		require.Len(t, status.Conditions, 1)
		require.Equal(
			t,
			[]kargoapi.WebhookReceiverDetails{
				receiver("dup", "/status-dup-1"),
				receiver("other", "/status-dup-3"),
			},
			status.WebhookReceivers,
		)
	})

	t.Run("status of a replaced row", func(t *testing.T) {
		t.Parallel()
		store, rec := newStore()
		_, err := store.Create(ctx, newCR("status-replaced"), "uid-a")
		require.NoError(t, err)
		// The reconciler reads row A.
		a, err := store.GetUnfiltered(ctx, "status-replaced")
		require.NoError(t, err)
		// Row A is replaced by row B, at generation 1 with no refresh token,
		// just as row A was.
		b, err := store.Create(ctx, newCR("status-replaced"), "uid-b")
		require.NoError(t, err)
		require.Equal(t, a.Generation, b.Generation)
		require.Nil(t, b.RefreshToken)
		rec.take()

		_, err = store.UpdateStatus(ctx, "status-replaced", Status{
			ObservedID:         a.ID,
			ObservedGeneration: a.Generation,
			Conditions:         ready,
			WebhookReceivers:   []kargoapi.WebhookReceiverDetails{receiver("gh", "/status-replaced")},
		})
		require.ErrorIs(t, err, database.ErrConflict)
		got, err := store.Get(ctx, "status-replaced", "uid-b")
		require.NoError(t, err)
		require.Nil(t, got.StatusObservedGeneration)
		require.Nil(t, got.StatusConditions)
		require.Empty(t, got.Receivers)
		require.Equal(t, b.UpdatedAt, got.UpdatedAt)
		require.Empty(t, rec.take())
	})

	t.Run("request refresh", func(t *testing.T) {
		t.Parallel()
		store, rec := newStore()
		created, err := store.Create(ctx, newCR("refresh"), "uid-1")
		require.NoError(t, err)
		rec.take()

		token, err := store.RequestRefresh(ctx, "refresh", "uid-1")
		require.NoError(t, err)
		require.NotEmpty(t, token)
		got, err := store.Get(ctx, "refresh", "uid-1")
		require.NoError(t, err)
		require.Equal(t, created.Generation, got.Generation)
		require.NotNil(t, got.RefreshToken)
		require.Equal(t, token, *got.RefreshToken)
		require.Equal(t, token, ToAPI(got).Annotations[kargoapi.AnnotationKeyRefresh])

		events := rec.take()
		requireKinds(t, events, nats.Updated)
		require.Nil(t, events[0].event.Old.RefreshToken)
		require.Equal(t, token, *events[0].event.New.RefreshToken)

		next, err := store.RequestRefresh(ctx, "refresh", "uid-1")
		require.NoError(t, err)
		require.NotEqual(t, token, next)

		_, err = store.RequestRefresh(ctx, "refresh", "other-uid")
		require.ErrorIs(t, err, database.ErrNotFound)
		_, err = store.RequestRefresh(ctx, "no-such-project", "uid-1")
		require.ErrorIs(t, err, database.ErrNotFound)
	})

	t.Run("leftover filtering by the mirror", func(t *testing.T) {
		t.Parallel()
		store, _ := newStore()
		for _, project := range []string{"mirror-none", "mirror-same", "mirror-other"} {
			created, err := store.Create(ctx, newCR(project), "uid-"+project)
			require.NoError(t, err)
			_, err = store.UpdateStatus(ctx, project, Status{
				ObservedID:         created.ID,
				ObservedGeneration: 1,
				WebhookReceivers:   []kargoapi.WebhookReceiverDetails{receiver("gh", "/"+project)},
			})
			require.NoError(t, err)
		}
		mirror(t, "mirror-same", "uid-mirror-same")
		mirror(t, "mirror-other", "a-recreated-uid")

		got, err := store.GetByWebhookReceiverPath(ctx, "/mirror-none")
		require.NoError(t, err)
		require.Equal(t, "mirror-none", got.ProjectName)
		require.Len(t, got.Receivers, 1)
		got, err = store.GetByWebhookReceiverPath(ctx, "/mirror-same")
		require.NoError(t, err)
		require.Equal(t, "mirror-same", got.ProjectName)
		_, err = store.GetByWebhookReceiverPath(ctx, "/mirror-other")
		require.ErrorIs(t, err, database.ErrNotFound)
		_, err = store.GetByWebhookReceiverPath(ctx, "/no-such-path")
		require.ErrorIs(t, err, database.ErrNotFound)

		configs, err := store.List(ctx)
		require.NoError(t, err)
		listed := map[string]Config{}
		for _, cfg := range configs {
			listed[cfg.ProjectName] = cfg
		}
		require.Contains(t, listed, "mirror-none")
		require.Contains(t, listed, "mirror-same")
		require.NotContains(t, listed, "mirror-other")
		require.Len(t, listed["mirror-same"].Receivers, 1)
		require.Equal(t, "/mirror-same", listed["mirror-same"].Receivers[0].Path)

		// The leftover is still there for those who ask for it by name.
		leftover, err := store.GetUnfiltered(ctx, "mirror-other")
		require.NoError(t, err)
		require.Equal(t, "uid-mirror-other", leftover.ProjectUID)
		require.Len(t, leftover.Receivers, 1)
		_, err = store.GetUnfiltered(ctx, "no-such-project")
		require.ErrorIs(t, err, database.ErrNotFound)
	})

	t.Run("delete", func(t *testing.T) {
		t.Parallel()
		store, rec := newStore()
		created, err := store.Create(ctx, newCR("delete"), "uid-1")
		require.NoError(t, err)
		rec.take()

		require.ErrorIs(t, store.Delete(ctx, "delete", "other-uid"), database.ErrNotFound)
		require.Empty(t, rec.take())
		require.NoError(t, store.Delete(ctx, "delete", "uid-1"))
		events := rec.take()
		requireKinds(t, events, nats.Deleted)
		require.Equal(t, created.ID, events[0].event.Old.ID)
		require.ErrorIs(t, store.Delete(ctx, "delete", "uid-1"), database.ErrNotFound)
	})

	t.Run("delete for project", func(t *testing.T) {
		t.Parallel()
		store, rec := newStore()

		_, err := store.Create(ctx, newCR("dfp-uid"), "uid-1")
		require.NoError(t, err)
		rec.take()

		// Another Project UID leaves the row alone.
		require.NoError(t, store.DeleteForProject(ctx, "dfp-uid", "other-uid"))
		require.Empty(t, rec.take())
		_, err = store.GetUnfiltered(ctx, "dfp-uid")
		require.NoError(t, err)

		// The row's own Project UID deletes it.
		require.NoError(t, store.DeleteForProject(ctx, "dfp-uid", "uid-1"))
		requireKinds(t, rec.take(), nats.Deleted)
		_, err = store.GetUnfiltered(ctx, "dfp-uid")
		require.ErrorIs(t, err, database.ErrNotFound)

		// With no row there is nothing to do.
		require.NoError(t, store.DeleteForProject(ctx, "dfp-uid", "uid-1"))
		require.Empty(t, rec.take())
	})

	t.Run("adopt rebinds a leftover row", func(t *testing.T) {
		t.Parallel()
		store, rec := newStore()
		created, err := store.Create(ctx, newCR("adopt"), "uid-old")
		require.NoError(t, err)
		_, err = store.UpdateStatus(ctx, "adopt", Status{
			ObservedID:         created.ID,
			ObservedGeneration: 1,
			Conditions:         ready,
			WebhookReceivers:   []kargoapi.WebhookReceiverDetails{receiver("gh", "/adopt")},
		})
		require.NoError(t, err)
		token, err := store.RequestRefresh(ctx, "adopt", "uid-old")
		require.NoError(t, err)
		before, err := store.Get(ctx, "adopt", "uid-old")
		require.NoError(t, err)
		rec.take()

		adopted, err := store.AdoptForProject(ctx, "adopt", "uid-new")
		require.NoError(t, err)
		require.True(t, adopted)

		_, err = store.Get(ctx, "adopt", "uid-old")
		require.ErrorIs(t, err, database.ErrNotFound)
		got, err := store.Get(ctx, "adopt", "uid-new")
		require.NoError(t, err)
		require.Equal(t, "uid-new", got.ProjectUID)
		require.Equal(t, before.ID, got.ID)
		require.JSONEq(t, string(before.Spec), string(got.Spec))
		require.Equal(t, before.Generation, got.Generation)
		require.NotNil(t, got.RefreshToken)
		require.Equal(t, token, *got.RefreshToken)
		require.Equal(t, before.StatusObservedGeneration, got.StatusObservedGeneration)
		require.Equal(t, before.StatusLastHandledRefresh, got.StatusLastHandledRefresh)
		require.JSONEq(t, string(before.StatusConditions), string(got.StatusConditions))
		require.Len(t, got.Receivers, 1)
		require.Equal(t, before.Receivers, got.Receivers)
		require.True(t, got.UpdatedAt.After(before.UpdatedAt))

		events := rec.take()
		requireKinds(t, events, nats.Updated)
		require.Equal(t, "uid-old", events[0].event.Old.ProjectUID)
		require.Equal(t, "uid-new", events[0].event.New.ProjectUID)
		require.Equal(t, before.Generation, events[0].event.New.Generation)
		require.Equal(t, token, *events[0].event.New.RefreshToken)

		// The adopted row's receivers resolve once the mirror has the new UID.
		mirror(t, "adopt", "uid-new")
		owner, err := store.GetByWebhookReceiverPath(ctx, "/adopt")
		require.NoError(t, err)
		require.Equal(t, got.ID, owner.ID)
	})

	t.Run("adopt is a no-op for the current Project", func(t *testing.T) {
		t.Parallel()
		store, rec := newStore()
		created, err := store.Create(ctx, newCR("adopt-current"), "uid-new")
		require.NoError(t, err)
		rec.take()

		adopted, err := store.AdoptForProject(ctx, "adopt-current", "uid-new")
		require.NoError(t, err)
		require.False(t, adopted)
		require.Empty(t, rec.take())
		got, err := store.Get(ctx, "adopt-current", "uid-new")
		require.NoError(t, err)
		require.Equal(t, created.UpdatedAt, got.UpdatedAt)
	})

	t.Run("adopt with no row", func(t *testing.T) {
		t.Parallel()
		store, rec := newStore()
		adopted, err := store.AdoptForProject(ctx, "adopt-none", "uid-new")
		require.NoError(t, err)
		require.False(t, adopted)
		require.Empty(t, rec.take())
	})

	t.Run("events are published after commit", func(t *testing.T) {
		t.Parallel()
		observer, err := pgxpool.New(ctx, pool.Config().ConnString())
		require.NoError(t, err)
		t.Cleanup(observer.Close)
		observed := database.New(observer)
		var mu sync.Mutex
		var seen []error
		rec := &recorder{hook: func(e recordedEvent) {
			// A separate connection sees only what has committed.
			var current *database.ProjectConfigRow
			row, getErr := observed.GetProjectConfig(ctx, e.event.Key.Project)
			switch {
			case errors.Is(getErr, pgx.ErrNoRows):
			case getErr != nil:
				mu.Lock()
				seen = append(seen, getErr)
				mu.Unlock()
				return
			default:
				current = &row
			}
			var want *database.ProjectConfigRow
			if e.event.Kind != nats.Deleted {
				want = e.event.New
			}
			if (want == nil) != (current == nil) ||
				(want != nil && (want.ID != current.ID || !want.UpdatedAt.Equal(current.UpdatedAt))) {
				mu.Lock()
				seen = append(seen, errors.New("event for an uncommitted write: "+string(e.event.Kind)))
				mu.Unlock()
			}
		}}
		store := New(pool, rec)
		created, err := store.Create(ctx, newCR("after-commit"), "uid-1")
		require.NoError(t, err)
		_, err = store.Update(ctx, newCR("after-commit"), "uid-1")
		require.NoError(t, err)
		_, err = store.UpdateStatus(ctx, "after-commit", Status{ObservedID: created.ID, ObservedGeneration: 1})
		require.NoError(t, err)
		_, err = store.RequestRefresh(ctx, "after-commit", "uid-1")
		require.NoError(t, err)
		_, err = store.AdoptForProject(ctx, "after-commit", "uid-2")
		require.NoError(t, err)
		require.NoError(t, store.Delete(ctx, "after-commit", "uid-2"))
		requireKinds(
			t,
			rec.take(),
			nats.Created, nats.Updated, nats.Updated, nats.Updated, nats.Updated, nats.Deleted,
		)
		mu.Lock()
		defer mu.Unlock()
		require.Empty(t, seen)
	})

	t.Run("nil publisher", func(t *testing.T) {
		t.Parallel()
		store := New(pool, nil)
		_, err := store.Create(ctx, newCR("nil-pub"), "uid-1")
		require.NoError(t, err)
		created, err := store.Create(ctx, newCR("nil-pub"), "uid-2")
		require.NoError(t, err)
		_, err = store.Update(ctx, newCR("nil-pub"), "uid-2")
		require.NoError(t, err)
		_, err = store.UpdateStatus(ctx, "nil-pub", Status{ObservedID: created.ID, ObservedGeneration: 1})
		require.NoError(t, err)
		_, err = store.RequestRefresh(ctx, "nil-pub", "uid-2")
		require.NoError(t, err)
		require.NoError(t, store.Delete(ctx, "nil-pub", "uid-2"))
		inserted, err := store.Import(ctx, newCR("nil-pub"), "uid-3")
		require.NoError(t, err)
		require.True(t, inserted)
		adopted, err := store.AdoptForProject(ctx, "nil-pub", "uid-4")
		require.NoError(t, err)
		require.True(t, adopted)
		require.NoError(t, store.DeleteForProject(ctx, "nil-pub", "uid-4"))
	})

	t.Run("import", func(t *testing.T) {
		t.Parallel()
		store, rec := newStore()
		other, err := store.Create(ctx, newCR("import-other"), "uid-o")
		require.NoError(t, err)
		_, err = store.UpdateStatus(ctx, "import-other", Status{
			ObservedID:         other.ID,
			ObservedGeneration: 1,
			WebhookReceivers:   []kargoapi.WebhookReceiverDetails{receiver("taken", "/import-shared")},
		})
		require.NoError(t, err)
		rec.take()

		cr := newCR("import")
		cr.Generation = 7
		cr.Annotations[kargoapi.AnnotationKeyRefresh] = "cr-token"
		cr.Status = kargoapi.ProjectConfigStatus{
			Conditions:         ready,
			ObservedGeneration: 7,
			LastHandledRefresh: "cr-token",
			WebhookReceivers: []kargoapi.WebhookReceiverDetails{
				receiver("gh", "/import-gh"),
				receiver("collides", "/import-shared"),
				receiver("gh", "/import-gh-duplicate"),
			},
		}
		inserted, err := store.Import(ctx, cr, "uid-1")
		require.NoError(t, err)
		require.True(t, inserted)

		got, err := store.Get(ctx, "import", "uid-1")
		require.NoError(t, err)
		require.Nil(t, got.StatusObservedGeneration)
		require.Equal(t, int64(1), got.Generation)
		require.Equal(t, map[string]string{"note": "x"}, got.Annotations)
		status := ToAPI(got).Status
		require.Len(t, status.Conditions, 1)
		require.Equal(t, "Synced", status.Conditions[0].Reason)
		require.Equal(t, []kargoapi.WebhookReceiverDetails{receiver("gh", "/import-gh")}, status.WebhookReceivers)
		requireKinds(t, rec.take(), nats.Created)

		// An existing row is never overwritten.
		again := newCR("import")
		again.Labels = map[string]string{"team": "z"}
		again.Status.WebhookReceivers = []kargoapi.WebhookReceiverDetails{receiver("gh", "/import-again")}
		inserted, err = store.Import(ctx, again, "uid-2")
		require.NoError(t, err)
		require.False(t, inserted)
		unchanged, err := store.Get(ctx, "import", "uid-1")
		require.NoError(t, err)
		require.Equal(t, got.UpdatedAt, unchanged.UpdatedAt)
		require.Equal(t, got.Receivers, unchanged.Receivers)
		require.Empty(t, rec.take())
	})
}
