package api

import (
	grpchealth "connectrpc.com/grpchealth"
)

func NewHealthChecker() grpchealth.Checker {
	return grpchealth.NewStaticChecker()
}
