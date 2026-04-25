package governance

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"text/template"

	"github.com/google/go-github/v76/github"
)

// executeActions performs a sequence of actions: add labels, remove labels,
// post a comment (with template rendering), convert the PR to a draft, apply
// PR policy, and/or close the issue/PR. Actions are applied in order. isPR
// determines whether a close action routes to the Pull Requests API or the
// Issues API. cfg is needed for actions that reference other parts of the
// configuration (e.g. ApplyPRPolicy).
func executeActions(
	ctx context.Context,
	cfg config,
	issuesClient IssuesClient,
	prsClient PullRequestsClient,
	owner string,
	repo string,
	number int,
	isPR bool,
	actions []action,
	templateData map[string]string,
) error {
	for _, a := range actions {
		if len(a.AddLabels) > 0 {
			if _, _, err := issuesClient.AddLabelsToIssue(
				ctx,
				owner,
				repo,
				number,
				a.AddLabels,
			); err != nil {
				return fmt.Errorf("error adding labels: %w", err)
			}
		}

		for _, label := range a.RemoveLabels {
			_, err := issuesClient.RemoveLabelForIssue(
				ctx,
				owner,
				repo,
				number,
				label,
			)
			if err != nil {
				// GitHub returns 404 when the label isn't attached to the
				// issue. Treat that as a no-op — the end state ("label not
				// present") is already what we wanted.
				var gerr *github.ErrorResponse
				if errors.As(err, &gerr) &&
					gerr.Response != nil &&
					gerr.Response.StatusCode == http.StatusNotFound {
					continue
				}
				return fmt.Errorf(
					"error removing label %q: %w", label, err,
				)
			}
		}

		if a.Comment != "" {
			body, err := renderTemplate(a.Comment, templateData)
			if err != nil {
				return fmt.Errorf("error rendering comment template: %w", err)
			}
			if _, _, err := issuesClient.CreateComment(
				ctx, owner, repo, number,
				&github.IssueComment{Body: github.Ptr(body)},
			); err != nil {
				return fmt.Errorf("error posting comment: %w", err)
			}
		}

		if a.ConvertToDraft && isPR {
			if err := prsClient.ConvertToDraft(ctx, owner, repo, number); err != nil {
				return fmt.Errorf("error converting PR to draft: %w", err)
			}
		}

		if a.ApplyPRPolicy && isPR {
			pr, _, err := prsClient.Get(ctx, owner, repo, number)
			if err != nil {
				return fmt.Errorf("error fetching PR for policy check: %w", err)
			}
			if err := applyPRPolicy(
				ctx,
				cfg,
				issuesClient,
				prsClient,
				owner,
				repo,
				pr,
				pr.GetUser().GetLogin(),
			); err != nil {
				return fmt.Errorf("error applying PR policy: %w", err)
			}
		}

		if a.Close {
			if isPR {
				if _, _, err := prsClient.Edit(
					ctx,
					owner,
					repo,
					number,
					&github.PullRequest{State: github.Ptr(prStateClosed)},
				); err != nil {
					return fmt.Errorf("error closing PR: %w", err)
				}
			} else {
				stateReason := "not_planned"
				if _, _, err := issuesClient.Edit(
					ctx,
					owner,
					repo,
					number,
					&github.IssueRequest{
						State:       github.Ptr(prStateClosed),
						StateReason: &stateReason,
					},
				); err != nil {
					return fmt.Errorf("error closing issue: %w", err)
				}
			}
		}
	}
	return nil
}

func renderTemplate(tmpl string, data map[string]string) (string, error) {
	if data == nil {
		return tmpl, nil
	}
	t, err := template.New("").Parse(tmpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err = t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
