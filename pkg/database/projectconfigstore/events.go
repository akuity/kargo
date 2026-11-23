package projectconfigstore

import (
	"context"

	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/logging"
	"github.com/akuity/kargo/pkg/reconciler/nats"
)

// Key identifies a ProjectConfig: the Project it configures, whose name it
// shares. It is what events about a ProjectConfig are keyed by.
type Key struct {
	Project string
}

func (k Key) String() string { return k.Project }

// Events is the topic the store announces every committed write to a
// ProjectConfig on: Created and Deleted when a row is created or deleted,
// and Updated when its spec, status or refresh token changes. Events carry
// the rows as committed.
var Events = nats.NewTopic[Key, database.ProjectConfigRow](
	nats.InternalSubjectPrefix + ".projectconfigs",
)

// The publish helpers announce a committed write. A failure to publish is not
// a failure of the write, which has already committed, so it is logged and
// not returned: the reconciler's resync picks up whatever events are lost.

func (s *store) publishCreated(ctx context.Context, created database.ProjectConfigRow) {
	key := Key{Project: created.ProjectName}
	if err := Events.PublishCreated(s.pub, key, created); err != nil {
		logPublishError(ctx, err, key, nats.Created)
	}
}

func (s *store) publishUpdated(ctx context.Context, old, updated database.ProjectConfigRow) {
	key := Key{Project: updated.ProjectName}
	if err := Events.PublishUpdated(s.pub, key, old, updated); err != nil {
		logPublishError(ctx, err, key, nats.Updated)
	}
}

func (s *store) publishDeleted(ctx context.Context, old database.ProjectConfigRow) {
	key := Key{Project: old.ProjectName}
	if err := Events.PublishDeleted(s.pub, key, old); err != nil {
		logPublishError(ctx, err, key, nats.Deleted)
	}
}

func logPublishError(ctx context.Context, err error, key Key, kind nats.EventKind) {
	logging.LoggerFromContext(ctx).Error(
		err, "error publishing ProjectConfig event",
		"project", key.Project,
		"kind", kind,
	)
}
