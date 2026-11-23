package targetstatuses

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database/targetstatusstore"
	libhttp "github.com/akuity/kargo/pkg/http"
)

// nolint: lll
// @id ListTargetStatuses
// @Summary List Target statuses
// @Description List where a project's Targets stand with the Stages that
// @Description govern them, one page at a time: one entry per Target and
// @Description governing Stage, ordered by Target name and then Stage name.
// @Description Nothing records a Target's status. Each entry is derived, on
// @Description request, from the Promotions and PromotionRollouts on record,
// @Description and which Stages govern a Target from the Stages' selectors
// @Description as they are now. A Target no Stage governs has no entry.
// @Description
// @Description Each entry has one state, the first of these that holds:
// @Description Promoting -- the last Promotion to the Target is running.
// @Description Pending -- the last Promotion has yet to start, or a
// @Description PromotionRollout that has not finished names the Target and
// @Description has not reached it yet.
// @Description NotAttempted -- a PromotionRollout that names the Target
// @Description ended, without succeeding, before creating a Promotion to it,
// @Description and there has been no Promotion to it since. A later rollout
// @Description that does not name the Target changes nothing.
// @Description NeverPromoted -- there has never been a Promotion to the
// @Description Target.
// @Description Failed -- the last Promotion failed, errored or was aborted.
// @Description Behind -- the last Promotion succeeded, and the Freight it
// @Description left on the Target is not the Stage's current Freight.
// @Description UpToDate -- the last Promotion succeeded and the Target is
// @Description not known to be behind.
// @Description
// @Description A state says how promotion went, not how the Target is
// @Description doing: Target health is not recorded, so nothing here
// @Description reports it. Entries are read from the database's mirror of
// @Description Promotions, so they trail the cluster by its sync.
// @Tags Core, Project-Level
// @Security BearerAuth
// @Param project path string true "Project name"
// @Param stage query string false "Only entries for this Stage"
// @Param state query []string false "Only entries in one of these states; repeat to give several" Enums(Promoting, Pending, Failed, NotAttempted, NeverPromoted, Behind, UpToDate)
// @Param continue query string false "The previous page's continue, to read the page after it; resend stage and state"
// @Param limit query int false "The page size, at most 500; 100 if omitted"
// @Produce json
// @Success 200 {object} EntryList "A page of Target statuses"
// @Router /v1beta1/projects/{project}/target-statuses [get]
func (h *Handler) list(c *gin.Context) {
	ctx := c.Request.Context()
	project := c.Param(paramProject)

	// The parameters are checked before anything is read, so that a request
	// that is refused costs nothing.
	states, err := parseStates(c.QueryArray(queryState))
	if err != nil {
		_ = c.Error(err)
		return
	}
	limit, err := parseLimit(c.Query(queryLimit))
	if err != nil {
		_ = c.Error(err)
		return
	}
	var after *entryCursor
	if token := c.Query(queryContinue); token != "" {
		after = &entryCursor{}
		if err = decodeToken(token, after); err != nil {
			_ = c.Error(err)
			return
		}
	}

	fleet, err := h.govern(ctx, project, c.Query(queryStage))
	if err != nil {
		_ = c.Error(err)
		return
	}
	// The page is read from the pairs that follow the previous page's last
	// entry. They are found by comparison rather than by position, so a
	// Target or Stage that has come or gone since shifts nothing.
	pairs := fleet.pairs
	if after != nil {
		pairs = pairs[after.search(fleet):]
	}
	entries, err := h.store.List(ctx, fleet.scope(pairs), states, limit)
	if err != nil {
		_ = c.Error(err)
		return
	}

	list := EntryList{Items: make([]Entry, len(entries))}
	for i, entry := range entries {
		list.Items[i] = present(fleet, entry)
	}
	// A full page may have more after it; a short page is the last.
	if len(entries) > 0 && len(entries) == int(limit) {
		last := list.Items[len(list.Items)-1]
		list.Continue = encodeToken(entryCursor{Target: last.Target, Stage: last.Stage})
	}
	c.JSON(http.StatusOK, list)
}

// present returns a stored entry as the API serves it, naming its Target
// and Stage, which the store knows only by id.
func present(fleet *fleet, stored targetstatusstore.Entry) Entry {
	target := fleet.targets[stored.TargetID]
	entry := Entry{
		Target:   target.Name,
		Stage:    fleet.stageName(stored.StageID),
		State:    stored.State,
		InFlight: stored.InFlight,
		Behind:   stored.Behind,
		Freight:  make([]Freight, len(stored.Freight)),
		Rollout:  stored.Rollout,
	}
	if len(target.Labels) > 0 {
		entry.Labels = target.Labels
	}
	for i, freight := range stored.Freight {
		entry.Freight[i] = Freight{
			Origin: kargoapi.FreightOrigin{
				Kind: kargoapi.FreightOriginKind(freight.OriginKind),
				Name: freight.OriginName,
			},
			Name: freight.Name,
		}
	}
	if stored.Last != nil {
		last := presentPromotion(*stored.Last)
		last.Stage = entry.Stage
		entry.LastPromotion = &last
	}
	return entry
}

// entryCursor marks a place in a list of entries by the entry at it: the
// last of the previous page. It is what a continue token holds.
type entryCursor struct {
	Target string `json:"target"`
	Stage  string `json:"stage"`
}

// search returns how many of the fleet's pairs are at or before the cursor:
// the index of the first pair after it.
func (c entryCursor) search(fleet *fleet) int {
	for i, pair := range fleet.pairs {
		if order := strings.Compare(pair.TargetName, c.Target); order > 0 ||
			(order == 0 && strings.Compare(fleet.stageName(pair.StageID), c.Stage) > 0) {
			return i
		}
	}
	return len(fleet.pairs)
}

// parseStates reads the state filter. No values is no filter; a value that
// is not a State is refused rather than matching nothing.
func parseStates(raw []string) ([]targetstatusstore.State, error) {
	states := make([]targetstatusstore.State, 0, len(raw))
	for _, value := range raw {
		state := targetstatusstore.State(value)
		if !state.Valid() {
			return nil, libhttp.ErrorStr(
				fmt.Sprintf("invalid state %q", value),
				http.StatusBadRequest,
			)
		}
		states = append(states, state)
	}
	return states, nil
}
