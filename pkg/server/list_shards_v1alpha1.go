package server

import (
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// kargoAgentLabelKey is the label key used by the Akuity Platform agent on
// its per-shard heartbeat ConfigMap. The label value is the shard name.
const kargoAgentLabelKey = "akuity.io/kargo-agent-name"

// agentObservedAtKey is the key in the heartbeat ConfigMap's data field that
// holds the RFC3339 timestamp of the agent's most recent heartbeat.
const agentObservedAtKey = "observedAt"

// shardStatus is the liveness state derived from a heartbeat ConfigMap.
type shardStatus string // @name ShardStatus

const (
	shardStatusAlive shardStatus = "alive"
	shardStatusDead  shardStatus = "dead"
)

// shardInfo describes the liveness of a single shard.
type shardInfo struct {
	Name     string      `json:"name"`
	Status   shardStatus `json:"status"`
	LastSeen *time.Time  `json:"lastSeen,omitempty"`
} // @name ShardInfo

// listShardsResponse is the response body of GET /v1beta1/system/shards.
type listShardsResponse struct {
	Shards           []shardInfo `json:"shards"`
	DefaultShardName string      `json:"defaultShardName,omitempty"`
} // @name ListShardsResponse

// @id ListShards
// @Summary List shard liveness
// @Description List shards known to the system along with a liveness status
// @Description derived from each shard's heartbeat ConfigMap. A shard is
// @Description `alive` when its heartbeat is fresh, and `dead` when the
// @Description heartbeat is stale or unparseable. Shards with no heartbeat
// @Description ConfigMap at all are not represented in the response. The
// @Description `defaultShardName` field, when set, indicates which shard
// @Description Stages with no explicit `spec.shard` should be associated with
// @Description for liveness purposes.
// @Tags System
// @Security BearerAuth
// @Produce json
// @Success 200 {object} listShardsResponse
// @Router /v1beta1/system/shards [get]
func (s *server) listShards(c *gin.Context) {
	ctx := c.Request.Context()

	// The heartbeat ConfigMaps live in the Kargo namespace and may not be
	// readable by typical authenticated users. Use the internal (non-
	// authorizing) client: shard liveness is operational information that
	// should be visible to any authenticated user, and the data exposed
	// here (shard name, alive/dead, last-seen timestamp) is non-sensitive.
	list := &corev1.ConfigMapList{}
	if err := s.client.InternalClient().List(
		ctx,
		list,
		client.InNamespace(s.cfg.KargoNamespace),
		client.HasLabels{kargoAgentLabelKey},
	); err != nil {
		_ = c.Error(err)
		return
	}

	now := time.Now()
	shards := make([]shardInfo, 0, len(list.Items))
	for _, cm := range list.Items {
		name := cm.Labels[kargoAgentLabelKey]
		if name == "" {
			continue
		}
		shards = append(shards, deriveShardInfo(name, cm.Data, now, s.cfg.AgentStatusDeadline))
	}

	slices.SortFunc(shards, func(lhs, rhs shardInfo) int {
		return strings.Compare(lhs.Name, rhs.Name)
	})

	c.JSON(http.StatusOK, listShardsResponse{
		Shards:           shards,
		DefaultShardName: s.cfg.DefaultShardName,
	})
}

// deriveShardInfo computes the liveness of a shard from the data field of its
// heartbeat ConfigMap. A shard is alive when observedAt is present, parseable,
// and within `deadline` of `now`. Otherwise it is dead.
func deriveShardInfo(
	name string,
	data map[string]string,
	now time.Time,
	deadline time.Duration,
) shardInfo {
	info := shardInfo{Name: name, Status: shardStatusDead}
	raw, ok := data[agentObservedAtKey]
	if !ok || raw == "" {
		return info
	}
	observedAt, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return info
	}
	info.LastSeen = &observedAt
	if now.Sub(observedAt) < deadline {
		info.Status = shardStatusAlive
	}
	return info
}
