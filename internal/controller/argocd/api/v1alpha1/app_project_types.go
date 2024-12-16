package v1alpha1

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/akuity/kargo/internal/controller/argocd/util/glob"
)

//+kubebuilder:object:root=true

type AppProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []AppProject `json:"items"`
}

//+kubebuilder:object:root=true

type AppProject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              AppProjectSpec `json:"spec"`
}

func globMatch(pattern string, val string, allowNegation bool, separators ...rune) bool { // nolint: unparam
	if allowNegation && isDenyPattern(pattern) {
		return !glob.Match(pattern[1:], val, separators...)
	}

	if pattern == "*" {
		return true
	}
	return glob.Match(pattern, val, separators...)
}

func isDenyPattern(pattern string) bool {
	return strings.HasPrefix(pattern, "!")
}
