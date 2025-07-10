package external

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/validation/field"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func ValidateWebhookReceivers(
	f *field.Path,
	webhookReceivers []kargoapi.WebhookReceiverConfig,
) field.ErrorList {
	errs := append(field.ErrorList{},
		validateUniqueNames(f, webhookReceivers)...,
	)
	return append(errs,
		validateMutuallyExclusive(f, webhookReceivers)...,
	)
}

func validateMutuallyExclusive(
	f *field.Path,
	webhookReceivers []kargoapi.WebhookReceiverConfig,
) field.ErrorList {
	var errs field.ErrorList
	for i, r := range webhookReceivers {
		var receivers []string
		if r.Bitbucket != nil {
			receivers = append(receivers, "Bitbucket")
		}
		if r.DockerHub != nil {
			receivers = append(receivers, "DockerHub")
		}
		if r.GitHub != nil {
			receivers = append(receivers, "GitHub")
		}
		if r.GitLab != nil {
			receivers = append(receivers, "GitLab")
		}
		if r.Quay != nil {
			receivers = append(receivers, "Quay")
		}
		if r.Gitea != nil {
			receivers = append(receivers, "Gitea")
		}
		if len(receivers) > 1 {
			errs = append(errs, field.Forbidden(
				f.Index(i),
				fmt.Sprintf(
					"cannot define a receiver that is of more than one type, found %d: %s",
					len(receivers),
					receivers,
				),
			))
		}
	}
	return errs
}

func validateUniqueNames(
	f *field.Path,
	webhookReceivers []kargoapi.WebhookReceiverConfig,
) field.ErrorList {
	var errs field.ErrorList
	dupes := make(map[string]int)
	for i, r := range webhookReceivers {
		if existingIndex, exists := dupes[r.Name]; exists {
			errs = append(errs, field.Invalid(
				f.Index(i).Child("name"),
				r.Name,
				fmt.Sprintf(
					"webhook receiver name already defined at %s",
					f.Index(existingIndex),
				),
			))
			continue
		}
		dupes[r.Name] = i
	}
	return errs
}
