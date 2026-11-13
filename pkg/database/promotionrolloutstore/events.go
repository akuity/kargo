package promotionrolloutstore

import (
	"github.com/akuity/kargo/pkg/database"
	"github.com/akuity/kargo/pkg/reconciler/nats"
)

// Events is the topic PromotionRollout events travel on. The store publishes
// an event after each committed write, carrying the row as the database
// returned it: Created when a rollout is created, and Updated, with the row
// before and after, when its status changes. A controller that drives
// rollouts subscribes to it to learn of new or changed rollouts without
// polling, and still lists the store periodically, since core NATS delivers
// an event only to the subscribers connected when it is published.
//
// Rollouts are never deleted on their own, so no Deleted event is published.
var Events = nats.NewTopic[Key, database.PromotionRolloutRow](
	nats.InternalSubjectPrefix + ".promotionrollouts",
)
