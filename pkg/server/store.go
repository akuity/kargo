package server

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/database"
	libhttp "github.com/akuity/kargo/pkg/http"
)

// promotionStore is the slice of database.Store the API server uses. Targets
// and PromotionRequests live in the database rather than in Kubernetes.
type promotionStore interface {
	ListTargets(context.Context, string) ([]database.Target, error)
	GetTarget(context.Context, database.GetTargetParams) (database.Target, error)
	ListPromotionRequests(context.Context, string) ([]database.PromotionRequestSnapshot, error)
	ListPromotionRequestsByStage(
		context.Context,
		database.ListPromotionRequestsByStageParams,
	) ([]database.PromotionRequestSnapshot, error)
	GetPromotionRequest(
		context.Context,
		database.GetPromotionRequestParams,
	) (database.PromotionRequestSnapshot, error)
	CreatePromotionRequest(
		context.Context,
		database.PromotionRequestCreate,
	) (database.PromotionRequestSnapshot, error)
}

var errDatabaseNotConfigured = libhttp.ErrorStr(
	"database is not configured",
	http.StatusNotImplemented,
)

// authorizeStoreRead runs the SubjectAccessReview a read of a database-backed
// resource requires. Reads from the database carry no authorization of their
// own -- unlike reads through the Kubernetes client -- so every handler that
// reads from the store MUST call this before its first store access.
//
// RBAC rules are just strings, so the rules that grant access to
// promotionrequests and targets keep working although nothing reads those
// custom resources anymore.
func (s *server) authorizeStoreRead(
	ctx context.Context,
	verb string,
	resource string,
	project string,
	name string,
) error {
	if s.authorizeFn == nil {
		return errors.New("authorize function is not configured")
	}
	return s.authorizeFn(
		ctx,
		verb,
		kargoapi.GroupVersion.WithResource(resource),
		"",
		client.ObjectKey{Namespace: project, Name: name},
	)
}

// storeNotFound returns the error a missing database-backed resource
// produces, shaped like the one a missing Kubernetes resource would.
func storeNotFound(resource, name string) error {
	return apierrors.NewNotFound(
		kargoapi.GroupVersion.WithResource(resource).GroupResource(),
		name,
	)
}

// maxResourceVersion returns the greatest of the given resource versions, or
// an empty string when there are none. It serves as a list's resource version
// so that a watch seeded with it reports only what changed afterwards.
func maxResourceVersion(versions ...string) string {
	var (
		greatest int64
		found    bool
	)
	for _, version := range versions {
		v, ok := parseResourceVersion(version)
		if ok && (!found || v > greatest) {
			greatest, found = v, true
		}
	}
	if !found {
		return ""
	}
	return strconv.FormatInt(greatest, 10)
}

// parseResourceVersion parses a database-backed resource's version. It
// reports false for an empty or unparsable version.
func parseResourceVersion(version string) (int64, bool) {
	if version == "" {
		return 0, false
	}
	v, err := strconv.ParseInt(version, 10, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}
