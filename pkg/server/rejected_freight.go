package server

import (
	"fmt"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

// rejectedFreightError returns an error explaining that the Freight has been
// rejected and cannot have the given action (e.g. "promoted" or "approved")
// performed on it. It returns nil if the Freight is not rejected.
func rejectedFreightError(freight *kargoapi.Freight, action string) error {
	if !freight.IsRejected() {
		return nil
	}
	return fmt.Errorf(
		"freight %q has been rejected and cannot be %s",
		freight.Name,
		action,
	)
}
