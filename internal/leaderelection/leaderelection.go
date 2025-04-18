package leaderelection

import (
	"crypto/sha256"
	"fmt"

	"k8s.io/apimachinery/pkg/labels"
)

// GenerateID generates a unique ID based on the provided name and additional
// strings. If the name is empty, it returns an empty string. If the name is
// not empty and there are additional strings, it appends a shortened SHA-256
// hash of the additional strings to the name, separated by a hyphen.
func GenerateID(name string, additions ...string) string {
	if name != "" && len(additions) > 0 {
		sum := sha256.New()
		for _, a := range additions {
			_, _ = sum.Write([]byte(a))
		}
		name = name + "-" + fmt.Sprintf("%x", sum.Sum(nil))[:8]
	}
	return name
}

// GenerateIDFromLabelSelector generates a unique ID based on the provided
// name and label selector using GenerateID. If the selector is empty, it
// returns the ID based on the name only.
func GenerateIDFromLabelSelector(name string, selector labels.Selector) string {
	if selector != nil {
		if s := selector.String(); s != "" {
			return GenerateID(name, s)
		}
	}
	return GenerateID(name)
}
