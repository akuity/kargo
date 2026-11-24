package server

import (
	"errors"
	"fmt"
	"net/http"

	corev1 "k8s.io/api/core/v1"

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
