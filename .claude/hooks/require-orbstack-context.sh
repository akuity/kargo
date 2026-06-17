#!/bin/bash
# PreToolUse hook: require orbstack k8s context for kubectl and helm cluster operations.
#
# Always allowed regardless of context:
#   kubectl config, kubectl version, kubectl api-resources, kubectl api-versions,
#   kubectl explain
#   helm template, helm show, helm repo, helm search, helm version, helm env,
#   helm lint, helm package, helm pull, helm dependency
#
# Everything else (kubectl get/apply/delete/..., helm install/upgrade/uninstall/...)
# requires the current k8s context to be "orbstack".

set -euo pipefail

INPUT=$(cat)
COMMAND=$(echo "$INPUT" | jq -r '.tool_input.command // empty')

# Strip leading whitespace and env var assignments (e.g. FOO=bar kubectl ...)
NORMALIZED=$(echo "$COMMAND" | sed 's/^[[:space:]]*\([A-Za-z_][A-Za-z0-9_]*=[^[:space:]]* \)*//')

# Extract the base command
BASE=$(echo "$NORMALIZED" | awk '{print $1}')

case "$BASE" in
  kubectl)
    SUBCMD=$(echo "$NORMALIZED" | awk '{print $2}')
    # Always allowed kubectl subcommands
    case "$SUBCMD" in
      config|version|api-resources|api-versions|explain|completion)
        exit 0
        ;;
    esac
    ;;
  helm)
    SUBCMD=$(echo "$NORMALIZED" | awk '{print $2}')
    # Always allowed helm subcommands (read-only / local-only)
    case "$SUBCMD" in
      template|show|repo|search|version|env|lint|package|pull|dependency|completion)
        exit 0
        ;;
    esac
    ;;
  *)
    # Not kubectl or helm -- not our concern
    exit 0
    ;;
esac

# If we're here, it's a kubectl or helm command that requires context check
CURRENT_CONTEXT=$(kubectl config current-context 2>/dev/null || echo "unknown")

if [ "$CURRENT_CONTEXT" != "orbstack" ]; then
  jq -n \
    --arg reason "Blocked: $BASE commands that touch the cluster require the orbstack context (current: $CURRENT_CONTEXT). Run: kubectl config set-context orbstack" \
    '{
      "hookSpecificOutput": {
        "hookEventName": "PreToolUse",
        "permissionDecision": "deny",
        "permissionDecisionReason": $reason
      }
    }'
fi
