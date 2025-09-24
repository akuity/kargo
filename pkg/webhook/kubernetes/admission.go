package webhook

import (
	"regexp"

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type IsRequestFromKargoControlplaneFn func(admission.Request) bool

func IsRequestFromKargoControlplane(regex *regexp.Regexp) IsRequestFromKargoControlplaneFn {
	return func(req admission.Request) bool {
		// Always return false if regex is not provided
		if regex == nil {
			return false
		}
		return regex.Match([]byte(req.UserInfo.Username))
	}
}
