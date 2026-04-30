#!/bin/bash
set -eo pipefail

if [ -z "$1" ]; then
  echo "Usage: ./ralph/afk.sh <prd-issue-number>"
  exit 1
fi

PRD_ISSUE="$1"
BRANCH="ralph/${PRD_ISSUE}"
WORKTREE_DIR=".worktrees/${BRANCH}"
REPO=$(gh repo view --json nameWithOwner --jq '.nameWithOwner')
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# jq filter to extract streaming text from assistant messages
stream_text='select(.type == "assistant").message.content[]? | select(.type == "text").text // empty | gsub("\n"; "\r\n") | . + "\r\n\n"'

echo "=== Ralph AFK: starting work on PRD #${PRD_ISSUE} ==="

# Create branch and worktree
if git show-ref --verify --quiet "refs/heads/${BRANCH}"; then
  echo "Branch ${BRANCH} already exists, reusing it."
else
  git branch "${BRANCH}"
  echo "Created branch ${BRANCH}."
fi

if [ -d "${WORKTREE_DIR}" ]; then
  echo "Worktree already exists at ${WORKTREE_DIR}, reusing it."
else
  git worktree add "${WORKTREE_DIR}" "${BRANCH}"
  echo "Created worktree at ${WORKTREE_DIR}."
fi

cd "${WORKTREE_DIR}"

# Copy ralph scripts into the worktree (always fresh copy)
rm -rf ralph/
cp -r "${SCRIPT_DIR}" ralph/

echo "=== Working in $(pwd) on branch ${BRANCH} ==="

# Loop until no open sub-issues remain
iteration=0
while true; do
  open_issues=$(gh issue list --repo "${REPO}" --search "parent-issue:${REPO}#${PRD_ISSUE}" --state open --json number --jq 'length')

  if [ "${open_issues}" -eq 0 ]; then
    echo "=== All sub-issues are closed. ==="
    break
  fi

  iteration=$((iteration + 1))
  echo "=== Iteration ${iteration}: ${open_issues} open sub-issue(s) remaining ==="

  commits=$(git log -n 5 --format="%H%n%ad%n%B---" --date=short 2>/dev/null || echo "No commits found")
  prompt=$(cat ralph/prompt.md)
  prompt="${prompt//\{\{PRD_ISSUE\}\}/$PRD_ISSUE}"
  prompt="${prompt//\{\{REPO\}\}/$REPO}"

  tmpfile=$(mktemp)
  trap "rm -f $tmpfile" EXIT

  claude -p \
    --verbose \
    --output-format stream-json \
    --settings ralph/settings.json \
    "$prompt Previous commits on this branch: $commits" \
  | grep --line-buffered '^{' \
  | tee "$tmpfile" \
  | jq --unbuffered -rj "$stream_text" || true

  echo ""
  echo "=== Iteration ${iteration} complete ==="
done

# Push and open PR
echo "=== Pushing branch and opening PR ==="
git push -u origin "${BRANCH}"

PRD_TITLE=$(gh issue view "${PRD_ISSUE}" --repo "${REPO}" --json title --jq '.title')

closed_issues=$(gh issue list --repo "${REPO}" --search "parent-issue:${REPO}#${PRD_ISSUE}" --state closed --json number,title --jq '.[] | "- #\(.number) \(.title)"')

gh pr create \
  --repo "${REPO}" \
  --base main \
  --head "${BRANCH}" \
  --title "${PRD_TITLE}" \
  --label prd \
  --body "$(cat <<EOF
## Summary

Implements PRD #${PRD_ISSUE}: ${PRD_TITLE}

## Resolved sub-issues

${closed_issues}

Closes #${PRD_ISSUE}
EOF
)"

echo "=== Ralph AFK complete. PR opened. ==="

# Clean up worktree
cd -
git worktree remove "${WORKTREE_DIR}"
echo "=== Worktree cleaned up. ==="
