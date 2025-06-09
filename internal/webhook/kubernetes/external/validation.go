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
