package client

import (
	"errors"
	"fmt"

	"github.com/akuity/kargo/pkg/client/generated/models"
)

type errorResponseGetter interface {
	error
	GetPayload() *models.ErrorResponse
}

// FormatAPIError returns a concise CLI error using the server-provided message
// when a generated REST response includes Kargo's standard ErrorResponse body.
func FormatAPIError(action string, err error) error {
	if err == nil {
		return nil
	}
	var responseErr errorResponseGetter
	if errors.As(err, &responseErr) {
		payload := responseErr.GetPayload()
		if payload != nil && payload.Error != "" {
			return fmt.Errorf("%s: %s", action, payload.Error)
		}
	}
	return fmt.Errorf("%s: %w", action, err)
}
