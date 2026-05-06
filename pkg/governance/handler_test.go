package governance

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-github/v76/github"
	"github.com/stretchr/testify/require"
)

const testWebhookSecret = "test-secret"

func Test_handler_ServeHTTP(t *testing.T) {
	testCases := []struct {
		name           string
		eventType      string
		body           any
		signature      *string
		expectedStatus int
	}{
		{
			name:           "missing signature",
			eventType:      "ping",
			signature:      github.Ptr(""),
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid signature",
			eventType:      "pull_request",
			signature:      github.Ptr("invalid-signature"),
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:      "issue_comment created event with created action",
			eventType: "issue_comment",
			body: github.IssueCommentEvent{
				Action:       github.Ptr("created"),
				Issue:        &github.Issue{Number: github.Ptr(1)},
				Comment:      &github.IssueComment{Body: github.Ptr("/help")},
				Repo:         &github.Repository{FullName: github.Ptr("test/repo")},
				Installation: &github.Installation{ID: github.Ptr(int64(1))},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "issue_comment non-created event with non-created action",
			eventType:      "issue_comment",
			body:           github.IssueCommentEvent{Action: github.Ptr("edited")},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:      "issues opened event with opened action",
			eventType: "issues",
			body: github.IssuesEvent{
				Action: github.Ptr("opened"),
				Issue:  &github.Issue{Number: github.Ptr(1)},
				Repo: &github.Repository{
					Name:  github.Ptr("repo"),
					Owner: &github.User{Login: github.Ptr("test")},
				},
				Installation: &github.Installation{ID: github.Ptr(int64(1))},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "issues non-opened event with non-opened action",
			eventType:      "issues",
			body:           github.IssuesEvent{Action: github.Ptr(issueStateClosed)},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "ping event",
			eventType:      "ping",
			body:           github.PingEvent{},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "pull_request opened event with opened action",
			eventType: "pull_request",
			body: github.PullRequestEvent{
				Action: github.Ptr("opened"),
				PullRequest: &github.PullRequest{
					Number:            github.Ptr(1),
					AuthorAssociation: github.Ptr("NONE"),
				},
				Repo: &github.Repository{
					Name:  github.Ptr("repo"),
					Owner: &github.User{Login: github.Ptr("test")},
				},
				Sender:       &github.User{Login: github.Ptr("someone")},
				Installation: &github.Installation{ID: github.Ptr(int64(1))},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "pull_request non-opened event with non-opened action",
			eventType:      "pull_request",
			body:           github.PullRequestEvent{Action: github.Ptr(prStateClosed)},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "unhandled event type",
			eventType:      "check_run",
			body:           github.CheckRunEvent{},
			expectedStatus: http.StatusNoContent,
		},
	}
	h := &handler{
		webhookSecret: []byte(testWebhookSecret),
		clientFactory: &fakeClientFactory{
			issuesClient: &fakeIssuesClient{},
			prsClient:    &fakePullRequestsClient{},
			reposClient:  &fakeRepositoriesClient{},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			body := []byte(`{}`)
			if testCase.body != "" {
				var err error
				body, err = json.Marshal(testCase.body)
				require.NoError(t, err)
			}
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
			req.Header.Set(github.EventTypeHeader, testCase.eventType)
			if testCase.signature != nil {
				req.Header.Set(github.SHA256SignatureHeader, *testCase.signature)
			} else {
				req.Header.Set(github.SHA256SignatureHeader, signPayload(body))
			}
			h.ServeHTTP(rr, req)
			require.Equal(t, testCase.expectedStatus, rr.Code)
		})
	}
}

func signPayload(payload []byte) string {
	mac := hmac.New(sha256.New, []byte(testWebhookSecret))
	mac.Write(payload) // nolint: errcheck
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
