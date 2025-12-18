package image

import (
	"maps"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewImage(t *testing.T) {
	const testDigest = "fake-digest"
	testDate := time.Now().UTC()
	testCases := []struct {
		name       string
		tag        string
		assertions func(*testing.T, image)
	}{
		{
			name: "tag is not a semver",
			tag:  "fake-tag",
			assertions: func(t *testing.T, image image) {
				require.Equal(t, "fake-tag", image.Tag)
				require.Nil(t, image.Semver)
				require.NotNil(t, image.CreatedAt)
				require.Equal(t, testDate, *image.CreatedAt)
				require.Equal(t, testDigest, image.Digest)
			},
		},
		{
			name: "tag is a semver",
			tag:  "v1.2.3",
			assertions: func(t *testing.T, image image) {
				require.Equal(t, "v1.2.3", image.Tag)
				require.NotNil(t, image.Semver)
				require.Equal(t, "1.2.3", image.Semver.String())
				require.NotNil(t, image.CreatedAt)
				require.Equal(t, testDate, *image.CreatedAt)
				require.Equal(t, testDigest, image.Digest)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.assertions(t, newImage(testCase.tag, testDigest, &testDate))
		})
	}
}

func TestImageGetPlatform(t *testing.T) {
	testConstraint := platformConstraint{
		os:   "linux",
		arch: "amd64",
	}
	testCases := []struct {
		name       string
		image      image
		assertions func(*testing.T, *platform)
	}{
		{
			name: "platforms is nil",
			assertions: func(t *testing.T, p *platform) {
				require.Nil(t, p)
			},
		},
		{
			name:  "platforms is empty",
			image: image{Platforms: []platform{}},
			assertions: func(t *testing.T, p *platform) {
				require.Nil(t, p)
			},
		},
		{
			name:  "no match found",
			image: image{Platforms: []platform{{OS: "linux", Arch: "arm64"}}},
			assertions: func(t *testing.T, p *platform) {
				require.Nil(t, p)
			},
		},
		{
			name: "match found",
			image: image{
				Tag: "v1.0.0",
				Platforms: []platform{
					{OS: "linux", Arch: "arm64"},
					{OS: testConstraint.os, Arch: testConstraint.arch},
				},
			},
			assertions: func(t *testing.T, p *platform) {
				require.NotNil(t, p)
				require.Equal(t, testConstraint.os, p.OS)
				require.Equal(t, testConstraint.arch, p.Arch)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := testCase.image.getPlatform(testConstraint)
			testCase.assertions(t, result)
		})
	}
}

func TestImageGetAnnotations(t *testing.T) {
	testConstraint := platformConstraint{
		os:   "linux",
		arch: "amd64",
	}
	baseAnnotations := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}
	platformAnnotations := map[string]string{
		"key2": "override-value2",
		"key3": "value3",
	}
	testCases := []struct {
		name       string
		image      image
		constraint *platformConstraint
		assertions func(*testing.T, map[string]string)
	}{
		{
			name:  "nil constraint with no annotations",
			image: image{},
			assertions: func(t *testing.T, annotations map[string]string) {
				require.Nil(t, annotations)
			},
		},
		{
			name:  "nil constraint with base annotations only",
			image: image{Annotations: baseAnnotations},
			assertions: func(t *testing.T, annotations map[string]string) {
				require.Equal(t, baseAnnotations, annotations)
				require.NotSame(t, &baseAnnotations, &annotations)
			},
		},
		{
			name: "nil constraint with platform annotations only",
			image: image{
				Platforms: []platform{{
					OS:   "linux",
					Arch: "amd64",
					// Should not be applied because platform was not specified
					Annotations: platformAnnotations,
				}},
			},
			assertions: func(t *testing.T, annotations map[string]string) {
				require.Nil(t, annotations)
			},
		},
		{
			name: "nil constraint with base annotations and platform annotations",
			image: image{
				Annotations: baseAnnotations,
				Platforms: []platform{{
					OS:   testConstraint.os,
					Arch: testConstraint.arch,
					// Should not be applied because platform was not specified
					Annotations: platformAnnotations,
				}},
			},
			assertions: func(t *testing.T, annotations map[string]string) {
				require.Equal(t, baseAnnotations, annotations)
				require.NotSame(t, &baseAnnotations, &annotations)
			},
		},
		{
			name:       "constraint with no annotations",
			image:      image{},
			constraint: &testConstraint,
			assertions: func(t *testing.T, annotations map[string]string) {
				require.Nil(t, annotations)
			},
		},
		{
			name:       "constraint with base annotations only",
			image:      image{Annotations: baseAnnotations},
			constraint: &testConstraint,
			assertions: func(t *testing.T, annotations map[string]string) {
				require.Equal(t, baseAnnotations, annotations)
				require.NotSame(t, &baseAnnotations, &annotations)
			},
		},
		{
			name: "constraint with base annotations, and no matching platform annotations",
			image: image{
				Annotations: baseAnnotations,
				Platforms: []platform{{
					OS:   "linux",
					Arch: "arm64",
					// Should not be applied because platform does not match
					Annotations: platformAnnotations,
				}},
			},
			constraint: &testConstraint,
			assertions: func(t *testing.T, annotations map[string]string) {
				require.Equal(t, baseAnnotations, annotations)
				require.NotSame(t, &baseAnnotations, &annotations)
			},
		},
		{
			name: "constraint with base annotations, and matching platform annotations",
			image: image{
				Annotations: baseAnnotations,
				Platforms: []platform{{
					OS:          testConstraint.os,
					Arch:        testConstraint.arch,
					Annotations: platformAnnotations,
				}},
			},
			constraint: &testConstraint,
			assertions: func(t *testing.T, annotations map[string]string) {
				expected := maps.Clone(baseAnnotations)
				for k, v := range platformAnnotations {
					expected[k] = v
				}
				require.Equal(t, expected, annotations)
			},
		},
		{
			name: "constraint with no base annotations, and matching platform annotations",
			image: image{
				Platforms: []platform{{
					OS:          testConstraint.os,
					Arch:        testConstraint.arch,
					Annotations: platformAnnotations,
				}},
			},
			constraint: &testConstraint,
			assertions: func(t *testing.T, annotations map[string]string) {
				require.Equal(t, platformAnnotations, annotations)
				require.NotSame(t, &platformAnnotations, &annotations)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := testCase.image.getAnnotations(testCase.constraint)
			testCase.assertions(t, result)
		})
	}
}
