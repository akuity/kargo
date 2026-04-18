package governance

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/google/go-github/v76/github"

	"github.com/akuity/kargo/pkg/logging"
)

// commentHandler handles comment-related events for a specific repository
// according to specific configuration.
type commentHandler struct {
	cfg          config
	owner        string
	repo         string
	issuesClient IssuesClient
	prsClient    PullRequestsClient
}

// handleCreated is the handler for the "issue_comment.created" event.
func (h *commentHandler) handleCreated(
	ctx context.Context,
	event *github.IssueCommentEvent,
) error {
	number := event.GetIssue().GetNumber()

	logger := logging.LoggerFromContext(ctx).WithValues("number", number)
	ctx = logging.ContextWithLogger(ctx, logger)

	body := event.GetComment().GetBody()

	// Collect every line that looks like a slash command. Commands may appear
	// on any line, with optional leading whitespace. Multiple commands in a
	// single comment are executed in order of appearance.
	parsedCmds := h.parseSlashCommands(body)
	if len(parsedCmds) == 0 {
		return nil
	}

	// Slash commands are maintainer-only.
	author := event.GetComment().GetAuthorAssociation()
	if !isMaintainer(h.cfg, author) {
		logger.Debug("comment author is not a maintainer, ignoring")
		return nil
	}

	// Determine context: issue or PR, and select the appropriate command map.
	isPR := event.GetIssue().GetPullRequestLinks() != nil
	var commands map[string]commandDef
	if isPR {
		if h.cfg.PullRequests != nil {
			commands = h.cfg.PullRequests.SlashCommands
		}
	} else {
		if h.cfg.Issues != nil {
			commands = h.cfg.Issues.SlashCommands
		}
	}

	// Each command runs independently: one failing command doesn't prevent
	// the next from being tried. Errors are logged at the point of failure
	// and aggregated for the final return value.
	var errs []error
	for _, pc := range parsedCmds {
		cmdLogger := logger.WithValues("command", pc.name)
		cmdCtx := logging.ContextWithLogger(ctx, cmdLogger)

		// /help is a built-in command that generates its response from the
		// command definitions.
		if pc.name == "help" {
			helpBody := buildHelpComment(commands)
			if _, _, err := h.issuesClient.CreateComment(
				cmdCtx, h.owner, h.repo, number,
				&github.IssueComment{Body: github.Ptr(helpBody)},
			); err != nil {
				cmdLogger.Error(err, "error posting help comment")
				errs = append(errs, fmt.Errorf(
					"command %q: %w", pc.name, err,
				))
			}
			continue
		}

		cmd, ok := commands[pc.name]
		if !ok {
			cmdLogger.Debug("unknown slash command, ignoring")
			continue
		}

		if cmd.RequiresArg && pc.arg == "" {
			cmdLogger.Debug("slash command requires an argument, ignoring")
			continue
		}

		templateData := map[string]string{
			"Arg":          pc.arg,
			"RepoFullName": h.owner + "/" + h.repo,
		}

		cmdLogger.Info("executing slash command")
		if err := executeActions(
			cmdCtx,
			h.cfg,
			h.issuesClient,
			h.prsClient,
			h.owner,
			h.repo,
			number,
			isPR,
			cmd.Actions,
			templateData,
		); err != nil {
			cmdLogger.Error(err, "error executing slash command")
			errs = append(errs, fmt.Errorf("command %q: %w", pc.name, err))
		}
	}

	return errors.Join(errs...)
}

type parsedCommand struct {
	name string
	arg  string
}

// parseSlashCommands scans comment body text for lines that look like slash
// commands. Leading whitespace on a line is tolerated. Commands are returned
// in the order they appear.
func (h *commentHandler) parseSlashCommands(body string) []parsedCommand {
	var cmds []parsedCommand
	for line := range strings.SplitSeq(body, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "/") {
			continue
		}
		parts := strings.Fields(line)
		name := strings.TrimPrefix(parts[0], "/")
		if name == "" {
			continue
		}
		var arg string
		if len(parts) > 1 {
			arg = strings.TrimPrefix(parts[1], "#")
		}
		cmds = append(cmds, parsedCommand{name: name, arg: arg})
	}
	return cmds
}

func buildHelpComment(commands map[string]commandDef) string {
	var b strings.Builder
	b.WriteString("## Available Slash Commands\n\n")
	b.WriteString("| Command | Description |\n")
	b.WriteString("|---------|-------------|\n")

	names := make([]string, 0, len(commands))
	for name := range commands {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		cmd := commands[name]
		desc := cmd.Description
		if desc == "" {
			desc = "(no description)"
		}
		argHint := ""
		if cmd.RequiresArg {
			argHint = " #N"
		}
		b.WriteString("| `/")
		b.WriteString(name)
		b.WriteString(argHint)
		b.WriteString("` | ")
		b.WriteString(desc)
		b.WriteString(" |\n")
	}

	b.WriteString("| `/help` | Show this list |\n")
	return b.String()
}
