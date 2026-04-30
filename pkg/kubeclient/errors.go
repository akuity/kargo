package kubeclient

import (
	"k8s.io/apimachinery/pkg/api/errors"
)

// IgnoreInvalid returns nil on Invalid errors.
// All other values that are not Invalid errors or nil are returned unmodified.
func IgnoreInvalid(err error) error {
	if errors.IsInvalid(err) {
		return nil
	}
	return err
}
