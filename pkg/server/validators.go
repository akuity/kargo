package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libhttp "github.com/akuity/kargo/pkg/http"
	"github.com/akuity/kargo/pkg/server/validation"
)

func validateFieldNotEmpty(fieldName string, fieldValue string) error {
	if fieldValue == "" {
		return connect.NewError(
			connect.CodeInvalidArgument,
			fmt.Errorf("%s should not be empty", fieldName),
		)
	}
	return nil
}

func (s *server) validateSystemLevelOrProject(
	systemLevel bool,
	project string,
) error {
	if !systemLevel && project == "" {
		return connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("project must be specified when system level is false"),
		)
	}
	return nil
}

func (s *server) validateProjectExists(ctx context.Context, project string) error {
	var cl client.Client = s.client
	if s.client != nil && s.client.InternalClient() != nil {
		cl = s.client.InternalClient()
	}
	if err := s.externalValidateProjectFn(ctx, cl, project); err != nil {
		if errors.Is(err, validation.ErrProjectNotFound) {
			return connect.NewError(connect.CodeNotFound, err)
		}
		var fieldErr *field.Error
		if ok := errors.As(err, &fieldErr); ok {
			return connect.NewError(connect.CodeInvalidArgument, err)
		}
		return fmt.Errorf("validate project: %w", err)
	}
	return nil
}

func validateGroupByOrderBy(group string, groupBy string, orderBy string) error {
	if group != "" && groupBy == "" {
		return connect.NewError(
			connect.CodeInvalidArgument,
			errors.New("cannot filter by group without group by"),
		)
	}
	switch groupBy {
	case GroupByImageRepository, GroupByGitRepository, GroupByChartRepository, "":
	default:
		return connect.NewError(
			connect.CodeInvalidArgument,
			fmt.Errorf("invalid group by: %s", groupBy),
		)
	}
	switch orderBy {
	case OrderByTag:
		if groupBy != GroupByImageRepository && groupBy != GroupByChartRepository {
			return connect.NewError(connect.CodeInvalidArgument,
				fmt.Errorf("tag ordering only valid when grouping by: %s, %s",
					GroupByImageRepository, GroupByChartRepository))
		}
	case OrderByFirstSeen, "":
	default:
		return connect.NewError(
			connect.CodeInvalidArgument,
			fmt.Errorf("invalid order by: %s", orderBy),
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

// requireSecretManagement checks if secret management is enabled. Returns true
// if enabled, or false if an error was added to the gin context.
func (s *server) requireSecretManagement(c *gin.Context) bool {
	if !s.cfg.SecretManagementEnabled {
		_ = c.Error(errSecretManagementDisabled)
		return false
	}
	return true
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

// getFreightByNameOrAliasForGin resolves a Freight resource by name or alias.
// It first tries to get by name, and if not found, tries to find by alias.
// Returns the Freight if found, or nil with an error added to the gin context
// if not found or an error occurred.
func (s *server) getFreightByNameOrAliasForGin(
	c *gin.Context,
	project string,
	nameOrAlias string,
) *kargoapi.Freight {
	ctx := c.Request.Context()

	// Try getting by name first
	freight := &kargoapi.Freight{}
	err := s.client.Get(
		ctx,
		client.ObjectKey{Name: nameOrAlias, Namespace: project},
		freight,
	)
	if err == nil {
		return freight
	}
	if !apierrors.IsNotFound(err) {
		_ = c.Error(err)
		return nil
	}

	// Try getting by alias
	list := &kargoapi.FreightList{}
	if err := s.client.List(
		ctx,
		list,
		client.InNamespace(project),
		client.MatchingLabels{kargoapi.LabelKeyAlias: nameOrAlias},
	); err != nil {
		_ = c.Error(err)
		return nil
	}
	if len(list.Items) == 0 {
		_ = c.Error(libhttp.ErrorStr(
			fmt.Sprintf(
				"Freight with name or alias %q not found in project %q",
				nameOrAlias, project,
			),
			http.StatusNotFound,
		))
		return nil
	}
	return &list.Items[0]
}
