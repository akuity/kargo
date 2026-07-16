package api

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/x/edition"
	"github.com/akuity/kargo/pkg/x/version"
)

func ToVersionProto(
	v version.Version,
	productEdition edition.Edition,
) *svcv1alpha1.VersionInfo {
	return &svcv1alpha1.VersionInfo{
		Version:      v.Version,
		GitCommit:    v.GitCommit,
		GitTreeDirty: v.GitTreeDirty,
		BuildTime:    timestamppb.New(v.BuildDate),
		GoVersion:    v.GoVersion,
		Compiler:     v.Compiler,
		Platform:     v.Platform,
		Edition:      productEditionToProto(productEdition),
	}
}

func productEditionToProto(
	productEdition edition.Edition,
) svcv1alpha1.ProductEdition {
	if productEdition.IsEnterprise() {
		return svcv1alpha1.ProductEdition_PRODUCT_EDITION_ENTERPRISE
	}
	return svcv1alpha1.ProductEdition_PRODUCT_EDITION_COMMUNITY
}
