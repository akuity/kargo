package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libhttp "github.com/akuity/kargo/pkg/http"
)

func validateGroupByOrderBy(group string, groupBy string, orderBy string) error {
	if group != "" && groupBy == "" {
		return libhttp.Error(
			errors.New("cannot filter by group without group by"),
			http.StatusBadRequest,
		)
	}
	switch groupBy {
	case GroupByImageRepository, GroupByGitRepository, GroupByChartRepository, "":
	default:
		return libhttp.Error(
			fmt.Errorf("invalid group by: %s", groupBy),
			http.StatusBadRequest,
		)
	}
	switch orderBy {
	case OrderByTag:
		if groupBy != GroupByImageRepository && groupBy != GroupByChartRepository {
			return libhttp.Error(
				fmt.Errorf("tag ordering only valid when grouping by: %s, %s",
					GroupByImageRepository, GroupByChartRepository),
				http.StatusBadRequest,
			)
		}
	case OrderByFirstSeen, "":
	default:
		return libhttp.Error(
			fmt.Errorf("invalid order by: %s", orderBy),
			http.StatusBadRequest,
		)
	}

	return nil
}

// validateRepoCredentialSecret validates that a secret is labeled as a valid
// repo credential type. Returns an error suitable for gin context if validation
// fails, or nil if the secret is valid.
func validateRepoCredentialSecret(secret *corev1.Secret) error {
	credType, isCredentials := secret.Labels[kargoapi.LabelKeyCredentialType]
	if !isCredentials {
		return libhttp.ErrorStr(
			fmt.Sprintf(
				"secret %s/%s exists, but is not labeled with %s",
				secret.Namespace,
				secret.Name,
				kargoapi.LabelKeyCredentialType,
			),
			http.StatusConflict,
		)
	}
	if credType != kargoapi.LabelValueCredentialTypeGit &&
		credType != kargoapi.LabelValueCredentialTypeHelm &&
		credType != kargoapi.LabelValueCredentialTypeImage {
		return libhttp.ErrorStr(
			fmt.Sprintf(
				"Kubernetes Secret %s/%s exists, but is labeled as unrecognized credential type %q",
				secret.Namespace,
				secret.Name,
				credType,
			),
			http.StatusConflict,
		)
	}
	return nil
}

// validateGenericCredentialSecret validates that a secret is labeled as a
// generic credential type. Returns an error suitable for gin context if
// validation fails, or nil if the secret is valid.
func validateGenericCredentialSecret(secret *corev1.Secret) error {
	if secret.Labels[kargoapi.LabelKeyCredentialType] != kargoapi.LabelValueCredentialTypeGeneric {
		return libhttp.ErrorStr(
			fmt.Sprintf(
				"Secret %s/%s exists, but is not labeled with %s=%s",
				secret.Namespace,
				secret.Name,
				kargoapi.LabelKeyCredentialType,
				kargoapi.LabelValueCredentialTypeGeneric,
			),
			http.StatusConflict,
		)
	}
	return nil
}

// bindJSONOrError binds JSON from the request body to the target.
// Returns true if successful, or false if an error was added to the gin context.
func bindJSONOrError(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		_ = c.Error(libhttp.Error(err, http.StatusBadRequest))
		return false
	}
	return true
}

// getFreightByNameOrAlias resolves a Freight resource by name or, when no
// Freight has that name, by alias. Errors carry an HTTP status: 404 when
// nothing matches and 409 when the alias matches more than one piece of
// Freight.
func (s *server) getFreightByNameOrAlias(
	ctx context.Context,
	project string,
	nameOrAlias string,
) (*kargoapi.Freight, error) {
	freight := &kargoapi.Freight{}
	err := s.client.Get(
		ctx,
		client.ObjectKey{Name: nameOrAlias, Namespace: project},
		freight,
	)
	if err == nil {
		return freight, nil
	}
	if !apierrors.IsNotFound(err) {
		return nil, err
	}
	return s.getFreightByAlias(ctx, project, nameOrAlias)
}

// getFreightByAlias resolves a Freight resource by alias. Errors carry an
// HTTP status: 404 when nothing matches and 409 when the alias matches more
// than one piece of Freight.
func (s *server) getFreightByAlias(
	ctx context.Context,
	project string,
	alias string,
) (*kargoapi.Freight, error) {
	list := &kargoapi.FreightList{}
	if err := s.client.List(
		ctx,
		list,
		client.InNamespace(project),
		client.MatchingLabels{kargoapi.LabelKeyAlias: alias},
	); err != nil {
		return nil, err
	}
	switch len(list.Items) {
	case 0:
		return nil, libhttp.ErrorStr(
			fmt.Sprintf(
				"Freight with name or alias %q not found in project %q",
				alias, project,
			),
			http.StatusNotFound,
		)
	case 1:
		return &list.Items[0], nil
	default:
		names := make([]string, len(list.Items))
		for i, freight := range list.Items {
			names[i] = freight.Name
		}
		return nil, libhttp.ErrorStr(
			fmt.Sprintf(
				"alias %q is shared by multiple pieces of Freight in project %q (%s); "+
					"refer to the Freight by name or give one of them a new alias",
				alias, project, strings.Join(names, ", "),
			),
			http.StatusConflict,
		)
	}
}
