package login

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReceiveAuthCode(t *testing.T) {
	const (
		testState = "fake-state"
		testCode  = "fake-code"
	)
	ctx := context.Background()
	codeCh := make(chan string)
	errCh := make(chan error)
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	go receiveAuthCode(ctx, listener, testState, codeCh, errCh)
	go func() {
		data := url.Values{
			"state": {testState},
			"code":  {testCode},
		}
		var res *http.Response
		res, err = http.PostForm(
			fmt.Sprintf("http://%s/auth/callback", listener.Addr().String()),
			data,
		)
		require.NoError(t, err)
		defer res.Body.Close()
		require.Equal(t, http.StatusOK, res.StatusCode)
	}()
	select {
	case code := <-codeCh: // Success
		require.Equal(t, testCode, code)
	case err = <-errCh:
		require.FailNow(t, "", "received unexpected error: %s", err)
	case <-time.After(5 * time.Second):
		require.FailNow(t, "timed out waiting for auth code")
	case <-ctx.Done():
		require.FailNow(t, "context canceled while waiting for auth code")
	}
}

func TestCreatePCKEVerifierAndChallenge(t *testing.T) {
	codeVerifier, codeChallenge, err := createPCKEVerifierAndChallenge()
	require.NoError(t, err)
	// Assert that the returned code verifier correctly answers the code challenge
	codeChallengeBytes, err := base64.RawURLEncoding.DecodeString(codeChallenge)
	require.NoError(t, err)
	expectedCodeChallengeBytes := sha256.Sum256([]byte(codeVerifier))
	require.Equal(
		t,
		expectedCodeChallengeBytes[:], // [:] converts [32]byte to []byte
		codeChallengeBytes,
	)
}

func TestRandStringFromCharset(t *testing.T) {
	const (
		minLen   = 10
		interval = 10
		maxLen   = 110
		chartSet = defaultRandStringCharSet
	)
	// Try a variety of lengths to ensure that the random string generator honors
	// the requested length
	for i := minLen; i <= maxLen; i += interval {
		set := map[string]struct{}{}
		// Generate many random strings of the same length to assert randomness
		for j := 0; j < 100; j++ {
			// Assert that the random string is the correct length
			str, err := randStringFromCharset(i, chartSet)
			require.NoError(t, err)
			require.Len(t, str, i)
			// Assert that the random string is composed only of characters from the
			// specified character set
			for _, c := range str {
				require.Contains(t, chartSet, string(c))
			}
			// Assert that the each new random string is not a duplicate of any that
			// preceded it. A duplicate would suggest inadequate randomness.
			_, ok := set[str]
			require.False(t, ok)
			set[str] = struct{}{}
		}
	}
}

func Test_normalizeURL(t *testing.T) {
	tests := []struct {
		address string
		want    string
	}{
		{
			address: "example.com",
			want:    "https://example.com",
		},
		{
			address: "example.com:8080",
			want:    "https://example.com:8080",
		},
		{
			address: "http://example.com",
			want:    "http://example.com",
		},
		{
			address: "https://example.com",
			want:    "https://example.com",
		},
		{
			address: " ftp://example.com",
			want:    "ftp://example.com",
		},
		{
			address: "  example.com/path  ",
			want:    "https://example.com/path",
		},
		{
			address: "localhost",
			want:    "https://localhost",
		},
		{
			address: "localhost:3000",
			want:    "https://localhost:3000",
		},
		{
			address: "",
			want:    "",
		},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeURL(tt.address))
		})
	}
}
