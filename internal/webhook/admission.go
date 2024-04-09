package webhook

import (
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apiserver/pkg/authentication/serviceaccount"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type IsRequestFromKargoControlplaneFn func(admission.Request) bool

func IsRequestFromKargoControlplane(
	knownServiceAccounts map[types.NamespacedName]struct{},
) IsRequestFromKargoControlplaneFn {
	return func(req admission.Request) bool {
		for account := range knownServiceAccounts {
			if serviceaccount.MatchesUsername(account.Namespace, account.Name, req.UserInfo.Username) {
				return true
			}
		}
		return false
	}
}
