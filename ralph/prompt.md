# Ralph — autonomous issue worker

You are working through sub-issues of a PRD (Product Requirements Document) tracked as a GitHub issue.

## Inputs

- PRD issue number: {{PRD_ISSUE}}
- Repository: {{REPO}}

## Step 1 — Understand the PRD

Fetch the PRD issue:

```
gh issue view {{PRD_ISSUE}}
```

Read it carefully. Understand the problem, solution, and implementation decisions.

## Step 2 — Discover open sub-issues

List all open sub-issues:

```
gh issue list --search "parent-issue:{{REPO}}#{{PRD_ISSUE}}" --state open
```

If there are no open sub-issues, output exactly: `NO MORE TASKS` and stop.

## Step 3 — Pick the best next sub-issue

Read the body of every open sub-issue with `gh issue view <number>`. Analyze dependencies between them and pick the one that should be implemented next. Prefer issues that have no dependencies on other open issues.

## Step 4 — Explore the codebase

Explore the repo to understand the existing code relevant to the chosen sub-issue. Read files, search for patterns, understand the architecture.

## Step 5 — Implement

Implement the changes required by the sub-issue. Follow the existing code style and conventions. Use /tdd skill if needed.

## Step 6 — Feedback loops

Run the feedback loops before committing:

- `make test` to run the tests
- `make lint` to run the linter

Fix any issues until both pass cleanly.

## Step 7 — Review

Run `/review` to review your changes. Address any findings.

Then run `/simplify` to simplify and refine the code. Address any findings.

Run `make test` and `make lint` again after changes.

## Step 8 — Commit

Make a git commit. The commit message must:

1. Reference the sub-issue number (e.g., `feat: add setup flow (#56)`)
2. Include key decisions made
3. Be concise but informative

## Step 9 — Close the sub-issue

Close the sub-issue:

```
gh issue close <number>
```

## Rules

- ONLY WORK ON A SINGLE SUB-ISSUE per invocation.
- Do NOT open a PR — that is handled externally.
- Do NOT push — that is handled externally.
- If there are no open sub-issues, just output `NO MORE TASKS` and stop.
