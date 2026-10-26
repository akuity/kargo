package webhook

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation/field"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/promotion"
)

func ValidatePromotionSteps(
	f *field.Path,
	steps []kargoapi.PromotionStep,
) field.ErrorList {
	errs := field.ErrorList{}
	indicesByAlias := make(map[string]int)
	for i, step := range steps {
		stepAlias := strings.TrimSpace(step.As)
		if stepAlias == "" {
			continue
		}
		if existingIndex, exists := indicesByAlias[stepAlias]; exists {
			errs = append(
				errs,
				field.Invalid(
					f.Index(i).Child("as"),
					stepAlias,
					fmt.Sprintf(
						"step alias duplicates that of %s",
						f.Index(existingIndex),
					),
				),
			)
		} else {
			indicesByAlias[stepAlias] = i
		}
		if promotion.ReservedStepAliasRegex.MatchString(stepAlias) {
			errs = append(
				errs,
				field.Invalid(
					f.Index(i).Child("as"),
					stepAlias,
					"step alias is reserved",
				),
			)
		}
	}
	return errs
}
