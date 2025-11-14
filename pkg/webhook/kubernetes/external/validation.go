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
	errs = append(errs,
		validateMutuallyExclusive(f, webhookReceivers)...,
	)
	for i, r := range webhookReceivers {
		if r.Generic != nil {
			errs = append(errs,
				validateGenericConfig(i, r.Generic)...,
			)
		}
	}
	return errs
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
		if r.Artifactory != nil {
			receivers = append(receivers, "Artifactory")
		}
		if r.Azure != nil {
			receivers = append(receivers, "Azure")
		}
		if r.Gitea != nil {
			receivers = append(receivers, "Gitea")
		}
		if r.Generic != nil {
			receivers = append(receivers, "Generic")
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

func validateGenericConfig(
	cfgIndex int,
	cfg *kargoapi.GenericWebhookReceiverConfig,
) field.ErrorList {
	var errs field.ErrorList
	for aIndex, action := range cfg.Actions {
		errs = append(errs, validateGenericTargets(cfgIndex, aIndex, action.Targets)...)
	}
	return errs
}

func validateGenericTargets(cfgIndex, actionIndex int, targets []kargoapi.GenericWebhookTarget) []*field.Error {
	var errs field.ErrorList
	for tIndex, target := range targets {
		if targetMisconfigured(&target) {
			targetPath := fmt.Sprintf(
				"spec.webhookReceivers[%d].generic.actions[%d].targets[%d]",
				cfgIndex, actionIndex, tIndex,
			)
			errs = append(errs, field.Invalid(
				field.NewPath(targetPath),
				target,
				"at least one of name, labelSelector, or indexSelector must be specified for target",
			))
		}
	}
	return errs
}

func targetMisconfigured(t *kargoapi.GenericWebhookTarget) bool {
	// these are all optional but at least one must be set
	return t.Name == "" &&
		len(t.LabelSelector.MatchLabels) == 0 &&
		len(t.LabelSelector.MatchExpressions) == 0 &&
		len(t.IndexSelector.MatchIndices) == 0
}
