package controller

import (
	"github.com/akuityio/k8sta/internal/bookkeeper"
	libOS "github.com/akuityio/k8sta/internal/common/os"
)

// bookkeeperClientConfig returns the address of the Bookkeeper server and
// related client connection options.
func bookkeeperClientConfig() (string, bookkeeper.ClientOptions, error) {
	opts := bookkeeper.ClientOptions{}
	address, err := libOS.GetRequiredEnvVar("BOOKKEEPER_ADDRESS")
	if err != nil {
		return address, opts, err
	}
	opts.AllowInsecureConnections, err =
		libOS.GetBoolFromEnvVar("BOOKKEEPER_IGNORE_CERT_WARNINGS", false)
	return address, opts, err
}
