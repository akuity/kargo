package svcv1alpha1

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/akuity/kargo/pkg/x/version"
)

func ToVersionProto(v version.Version) *VersionInfo {
	return &VersionInfo{
		Version:      v.Version,
		GitCommit:    v.GitCommit,
		GitTreeDirty: v.GitTreeDirty,
		BuildTime:    timestamppb.New(v.BuildDate),
		GoVersion:    v.GoVersion,
		Compiler:     v.Compiler,
		Platform:     v.Platform,
	}
}
