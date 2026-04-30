#!/bin/bash
set -eo pipefail

if [ -z "$1" ]; then
  echo "Usage: ./ralph/once.sh <prd-issue-number>"
  exit 1
fi

PRD_ISSUE="$1"
REPO=$(gh repo view --json nameWithOwner --jq '.nameWithOwner')

commits=$(git log -n 5 --format="%H%n%ad%n%B---" --date=short 2>/dev/null || echo "No commits found")
prompt=$(cat ralph/prompt.md)
prompt="${prompt//\{\{PRD_ISSUE\}\}/$PRD_ISSUE}"
prompt="${prompt//\{\{REPO\}\}/$REPO}"

claude \
  "$prompt Previous commits on this branch: $commits"
