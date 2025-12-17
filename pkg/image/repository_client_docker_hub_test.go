//go:build dockerhub

package image

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// All test cases in this file are integration tests that rely on Docker Hub.
// You're very likely to get rate-limited executing these tests, unless you're a
// paying Docker customer, so they're disabled by default.
//
// To use your Docker credentials, set env vars:
// - DOCKER_HUB_USERNAME
// - DOCKER_HUB_PASSWORD (personal access token)

func TestGetTags(t *testing.T) {
	client, err := newRepositoryClient("debian", false, getDockerHubCreds(), true)
	require.NoError(t, err)
	require.NotNil(t, client)
	tags, err := client.getTags(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, tags)
}

func getDockerHubCreds() *Credentials {
	return &Credentials{
		// It's ok if these are empty, but you'll probably get rate limited.
		Username: os.Getenv("DOCKER_HUB_USERNAME"),
		Password: os.Getenv("DOCKER_HUB_PASSWORD"),
	}
}
