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
		if r.GitHub != nil {
			receivers = append(receivers, "GitHub")
		}
		if r.GitLab != nil {
			receivers = append(receivers, "GitLab")
		}
		if r.Quay != nil {
			receivers = append(receivers, "Quay")
		}
		if len(receivers) > 1 {
			errs = append(errs, field.Invalid(
				f.Index(i),
				receivers,
				fmt.Sprintf(
					"only one webhook receiver can be defined at a time, found: %s",
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
