package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const githubAPIBaseURL = "https://api.github.com"

// MergePullRequest merges the numbered pull request in the given GitHub
// repository using token for authentication. It retries while GitHub reports
// the pull request as not yet mergeable (mergeability still computing or a
// transient head change), up to timeout. It is intended to unblock
// git-wait-for-pr promotion steps, which otherwise never complete in an
// unattended run.
func MergePullRequest(
	ctx context.Context,
	repoURL, token string,
	prNumber int,
	timeout time.Duration,
) error {
	owner, repo, err := gitHubOwnerRepo(repoURL)
	if err != nil {
		return err
	}

	timedCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%d/merge", githubAPIBaseURL, owner, repo, prNumber)
	payload, err := json.Marshal(map[string]string{"merge_method": "merge"})
	if err != nil {
		return err
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	var lastErr error
	for {
		req, err := newGitHubRequest(timedCtx, http.MethodPut, url, token, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		body, status, err := doGitHub(req)
		if err != nil {
			return err
		}
		switch status {
		case http.StatusOK:
			return nil
		case http.StatusMethodNotAllowed, http.StatusConflict:
			lastErr = fmt.Errorf("merge not allowed yet (status %d): %s", status, string(body))
		default:
			return fmt.Errorf("merging pull request %d: unexpected status %d: %s", prNumber, status, string(body))
		}
		select {
		case <-timedCtx.Done():
			return fmt.Errorf("timed out merging pull request %d: %w: %w", prNumber, timedCtx.Err(), lastErr)
		case <-ticker.C:
		}
	}
}

func newGitHubRequest(ctx context.Context, method, url, token string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func doGitHub(req *http.Request) ([]byte, int, error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

// gitHubOwnerRepo parses the owner and repository name from a GitHub HTTPS URL,
// e.g. https://github.com/octocat/repo.git -> ("octocat", "repo").
func gitHubOwnerRepo(repoURL string) (string, string, error) {
	trimmed := strings.TrimSuffix(strings.TrimSuffix(repoURL, ".git"), "/")
	parts := strings.Split(trimmed, "/")
	if len(parts) < 2 || parts[len(parts)-2] == "" || parts[len(parts)-1] == "" {
		return "", "", fmt.Errorf("cannot parse owner/repo from %q", repoURL)
	}
	return parts[len(parts)-2], parts[len(parts)-1], nil
}
