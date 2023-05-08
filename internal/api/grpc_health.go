package api

import (
	grpchealth "github.com/bufbuild/connect-grpchealth-go"
)

func NewHealthChecker() grpchealth.Checker {
	return grpchealth.NewStaticChecker()
}
