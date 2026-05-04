package governance

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/goccy/go-yaml"
	"github.com/google/go-github/v76/github"
)

const actionKindComment = "comment"

func init() {
	defaultActionRunnerRegistry.MustRegister(actionRunnerRegistration{
		Name:  actionKindComment,
		Value: commentRunner{},
	})
}

// commentRunner posts a comment on the issue or pull request. Its config
// is a string template (text/template syntax). Available template
// variables depend on the calling context (e.g. .Arg, .RepoFullName for
// slash commands; .IssueNumber, .BlockingLabels for OnBlockedIssue).
type commentRunner struct{}

func (commentRunner) run(
	ctx context.Context,
	ac *actionContext,
	cfg []byte,
) error {
	var tmpl string
	if err := yaml.Unmarshal(cfg, &tmpl); err != nil {
		return fmt.Errorf("decoding comment config: %w", err)
	}
	if tmpl == "" {
		return nil
	}
	body, err := renderTemplate(tmpl, ac.templateData)
	if err != nil {
		return fmt.Errorf("error rendering comment template: %w", err)
	}
	if _, _, err := ac.issuesClient.CreateComment(
		ctx, ac.owner, ac.repo, ac.number,
		&github.IssueComment{Body: github.Ptr(body)},
	); err != nil {
		return fmt.Errorf("error posting comment: %w", err)
	}
	return nil
}

// renderTemplate evaluates a text/template against the supplied data. If
// data is nil, the template string is returned as-is.
func renderTemplate(tmpl string, data map[string]string) (string, error) {
	if data == nil {
		return tmpl, nil
	}
	t, err := template.New("").Parse(tmpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
