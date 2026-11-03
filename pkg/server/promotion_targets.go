package server

import (
	"context"
	"net/http"

	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	libhttp "github.com/akuity/kargo/pkg/http"
)

var errDatabaseNotConfigured = libhttp.ErrorStr(
	"database is not configured",
	http.StatusNotImplemented,
)

func (s *server) getListTargetsFn() (func(context.Context, string) ([]kargoapi.Target, error), error) {
	if s.listTargetsFn == nil {
		return nil, errDatabaseNotConfigured
	}
	return s.listTargetsFn, nil
}
