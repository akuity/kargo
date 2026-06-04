package server

import (
	"fmt"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
)

func rejectedFreightPromotionError(freight *kargoapi.Freight) error {
	if !freight.IsRejected() {
		return nil
	}
	return fmt.Errorf(
		"freight %q has been rejected and cannot be promoted",
		freight.Name,
	)
}

func rejectedFreightApprovalError(freight *kargoapi.Freight) error {
	if !freight.IsRejected() {
		return nil
	}
	return fmt.Errorf(
		"freight %q has been rejected and cannot be approved",
		freight.Name,
	)
}
