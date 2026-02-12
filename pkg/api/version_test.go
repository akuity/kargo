package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/timestamppb"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	"github.com/akuity/kargo/pkg/x/version"
)

func TestToVersionProto(t *testing.T) {
	fixedTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		input    version.Version
		expected *svcv1alpha1.VersionInfo
	}{
		{
			name: "complete version info",
			input: version.Version{
				Version:      "v1.0.0",
				GitCommit:    "a1b2c3d",
				GitTreeDirty: true,
				BuildDate:    fixedTime,
				GoVersion:    "go1.21.0",
				Compiler:     "gc",
				Platform:     "linux/amd64",
			},
			expected: &svcv1alpha1.VersionInfo{
				Version:      "v1.0.0",
				GitCommit:    "a1b2c3d",
				GitTreeDirty: true,
				BuildTime:    timestamppb.New(fixedTime),
				GoVersion:    "go1.21.0",
				Compiler:     "gc",
				Platform:     "linux/amd64",
			},
		},
		{
			name: "empty version info",
			input: version.Version{
				BuildDate: time.Time{},
			},
			expected: &svcv1alpha1.VersionInfo{
				BuildTime: timestamppb.New(time.Time{}),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToVersionProto(tt.input)

			assert.Equal(t, tt.expected.Version, result.Version)
			assert.Equal(t, tt.expected.GitCommit, result.GitCommit)
			assert.Equal(t, tt.expected.GitTreeDirty, result.GitTreeDirty)
			assert.Equal(t, tt.expected.GoVersion, result.GoVersion)
			assert.Equal(t, tt.expected.Compiler, result.Compiler)
			assert.Equal(t, tt.expected.Platform, result.Platform)

			if tt.expected.BuildTime != nil && result.BuildTime != nil {
				assert.True(
					t,
					tt.expected.BuildTime.AsTime().Equal(result.BuildTime.AsTime()),
					"BuildTime timestamps should match",
				)
			} else {
				assert.Equal(t, tt.expected.BuildTime, result.BuildTime)
			}
		})
	}
}
