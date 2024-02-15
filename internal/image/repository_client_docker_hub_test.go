//go:build dockerhub
// +build dockerhub

package image

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
)

// All test cases in this file are integration tests that rely on Docker Hub.
// You're very likely to get rate-limited executing these tests, unless you're a
// paying Docker customer, so they're disabled by default.
//
// To use your Docker credentials, set env vars:
// - DOCKER_HUB_USERNAME
// - DOCKER_HUB_USERNAME (personal access token)

func TestGetChallengeManager(t *testing.T) {
	challengeManager, err := getChallengeManager(
		"https://registry-1.docker.io",
		http.DefaultTransport,
	)
	require.NoError(t, err)
	require.NotNil(t, challengeManager)
}

func TestGetTags(t *testing.T) {
	client, err := newRepositoryClient("debian", getDockerHubCreds())
	require.NoError(t, err)
	require.NotNil(t, client)
	tags, err := client.getTags(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, tags)
}

func TestGetManifestByTag(t *testing.T) {
	client, err := newRepositoryClient("debian", getDockerHubCreds())
	require.NoError(t, err)
	require.NotNil(t, client)
	// Note: This is only going to come back with a manifest list. It won't
	// follow the references found therein.
	manifest, err := client.getManifestByTag(context.Background(), "latest")
	require.NoError(t, err)
	require.NotNil(t, manifest)
}

func TestGetManifestByDigest(t *testing.T) {
	// This is a real digest for a debian bookworm image
	// nolint: lll
	// https://hub.docker.com/layers/library/debian/bookworm/images/sha256-bd989d36e94ef694541231541b04c8c89bc6ccb8d015f12a715b605c64edde4a
	const testDigest = "sha256:bd989d36e94ef694541231541b04c8c89bc6ccb8d015f12a715b605c64edde4a" // nolint: gosec
	client, err := newRepositoryClient("debian", getDockerHubCreds())
	require.NoError(t, err)
	m, err :=
		client.getManifestByDigest(context.Background(), testDigest)
	require.NoError(t, err)
	_, manifestBytes, err := m.Payload()
	require.NoError(t, err)
	require.Equal(
		t,
		testDigest,
		digest.FromBytes(manifestBytes).String(),
	)
}

func getDockerHubCreds() *Credentials {
	return &Credentials{
		// It's ok if these are empty, but you'll probably get rate limited.
		Username: os.Getenv("DOCKER_HUB_USERNAME"),
		Password: os.Getenv("DOCKER_HUB_PASSWORD"),
	}
}
