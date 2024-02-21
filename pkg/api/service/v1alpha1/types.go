package svcv1alpha1

import (
	"time"

	"github.com/akuity/kargo/internal/version"
)

func ToVersionProto(v version.Version) *VersionInfo {
	return &VersionInfo{
		Version:          v.Version,
		GitCommit:        v.GitCommit,
		GitTreeDirty:     v.GitTreeDirty,
		GoVersion:        v.GoVersion,
		Compiler:         v.Compiler,
		Platform:         v.Platform,
		VersionBuildTime: v.BuildDate.UTC().Format(time.RFC3339),
	}
}
