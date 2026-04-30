# EVOLVING — Harness & Prompt Improvements

A living analysis of WildGecu's harness vs the OSS coding-agent field, with prioritized improvements. Compiled 2026-04-30.

## Reference projects surveyed

- **Claude Code** (Anthropic, TS/Node) — gold-standard reference, ~110 conditional prompt fragments per request, ~165 source files, 12-event hook framework, OS-native sandbox.
- **OpenCode** ([sst/opencode](https://github.com/sst/opencode), TS+Rust) — provider-specific system prompts, client/server architecture, MCP-first.
- **Crush** ([charmbracelet/crush](https://github.com/charmbracelet/crush), Go) — closest cousin: Go + TUI + multi-provider on top of [`fantasy`](https://github.com/charmbracelet/fantasy). Recently shrank tool descriptions by ~120k tokens/session.
- **Aider** (Python) — repo-map (tree-sitter + PageRank), per-model edit-format dispatch, architect/editor split.
- **Cline** (TS, VSCode) — Plan/Act mode pioneer, deepest MCP marketplace, workspace checkpoints.
- **Codex CLI** (OpenAI, Rust) — per-OS sandbox (Seatbelt / bubblewrap / Windows Sandbox), 4 approval modes.
- **Goose** (Block, Rust) — YAML recipes/subrecipes with per-stage model.
- **Gemini CLI** (Google, TS) — `save_memory` tool, GEMINI.md memory.
- **Continue** (TS) — roles abstraction (chat/autocomplete/edit/apply/summarize), tools-as-XML fallback for weak tool-calling.
- **OpenClaw** ([openclaw/openclaw](https://github.com/openclaw/openclaw)) — multi-channel personal AI (chat across WhatsApp/Telegram/Slack), not a coding agent. Uses a similar SOUL.md / AGENTS.md / TOOLS.md pattern, so worth knowing about as parallel evolution.

## Universal patterns the field has converged on

- Markdown system prompt with hierarchical sections / XML-tagged examples.
- Read / Edit (string-replace, must be unique unless `replace_all`) / Write / Bash / Glob / Grep canonical tool set.
- Plan vs execute mode separation, enforced by both prompt fragments AND a read-only tool subset.
- Memory file at project root (`AGENTS.md` is the cross-vendor standard, ~20k repos, stewarded by Linux Foundation's Agentic AI Foundation).
- MCP client for tool extension.
- Subagent for fan-out research with isolated context.
- Per-step approval as the safety baseline.
- Streaming tokens + streaming tool calls.

## What WildGecu has today (snapshot)

**Prompts** (`pkg/agent/*.md`):
- `AGENT.md` (74 lines) — chat-mode base.
- `CODE_AGENT.md` — code-mode base, with `{CWD}` interpolation.
- `BOOTSTRAP.md` — first-time identity interview.
- `MEMORY_AGENT.md` — post-session curator.

**Layered system prompt** (`pkg/agent/soul.go:53` `BuildSystemPrompt`): AGENT → SOUL → MEMORY → USER (workspace-local). Same prompt sent to every provider.

**Tools** (`pkg/agent/tools/`):
- `bash` (30s hard timeout, no output truncation, no denylist).
- `node` (30s timeout).
- `fetch_url` (512KB limit).
- `list_files` / `read_file` / `write_file` / `update_file` (single-edit).
- `list_skills` / `read_skill`.
- `inform_user` (one-way).
- `spawn_agent` (model override + tool subset, depth-1 only).
- `todo_create` / `todo_update` (session-scoped).
- `list_models`.
- `get_current_time`.
- Bootstrap-only: `write_soul`. Memory-only: `write_memory`.

**Harness primitives**:
- Parallel tool calling (`pkg/provider/agent.go`).
- Streaming.
- `RequestReminder` callback (`pkg/session/session.go:25-29`) — Claude Code-style cache-friendly reminder injection. **Currently unused.**
- Subagents with model override + tool subset, depth-1.
- Cron (in-process scheduler).
- Skills (lazy-loaded MD with frontmatter).
- Slash commands (`/help`, `/clean`, `/status`, `/todos`).
- Telegram bridge.
- Daemon mode + IPC socket.
- Self-update.
- Outer-loop pattern in `ralph/` (shell-driven Claude Code calls for PRD sub-issues).

## Universal-pattern gaps in WildGecu (must-have parity)

| Missing | Where in field | Severity |
|---|---|---|
| `grep` tool | Universal | High — model shells out to `grep`/`rg` via bash, breaks portability + eats context |
| `glob` tool | Universal | High — `list_files` only walks one directory |
| Multi-edit (batched string-replace, atomic) | Claude Code, Crush | Medium — 5x latency win on refactors |
| Bash output truncation | Universal | High — 100MB log → 100MB into LLM |
| Configurable bash timeout | Universal | High — 30s breaks `make test`, `go build` |
| Denylist / approval prompts on destructive bash | Universal | High — `rm -rf /`, `git push --force` run silently |
| `web_search` companion to `fetch_url` | Universal | Medium — config has `google_search: true` half-wired |
| Image read | Most | Low — `read_file` rejects non-UTF8 (`files.go:115`) |
| `AGENTS.md` walk | Universal | Medium — free interop with ~20k repos |
| Prompt caching | Universal | High — 70-90% input-token cost reduction available |
| `/compact` + 5-bucket continuation summary | Claude Code, OpenCode | Medium — no mid-session compaction |
| `AskUserQuestion` first-class tool | Claude Code, OpenCode | Medium — `inform_user` is one-way only |
| MCP client | Universal | High — single biggest ecosystem leverage |
| Per-tool side-effect classification + approval | Universal | High — README says "safe by design" but it's aspirational |
| Hooks (PreToolUse / PostToolUse / SessionStart) | Claude Code (12 events), Crush (PreToolUse only) | Medium — solves automated-behavior gap that prompts can't |
| Workspace checkpoints (per-step rollback) | Cline | Medium — cheap insurance for autonomous mode |

## Half-built features in the codebase (free leverage — finish these first)

| What's there | Where | Wire-up needed |
|---|---|---|
| **System reminders** (Claude Code's killer pattern) | `pkg/session/session.go:25-29` `RequestReminder` callback with cache-friendly `applyReminder`/`stripReminder` | Currently nothing populates it. Wire to: stale-todo nudge, file-modified-by-user notice, plan-mode active marker, memory-curation reminder, tool-budget warning. Each ~10-30 lines. |
| **Per-role models** | `pkg/agent/agent.go:23` `MemoryModel` field | Generalize to a `Roles` map: `memory` / `bootstrap` / `subagent_default` / `compaction` / `architect` / `editor`. Continue's roles abstraction is the cleanest reference. |
| **Web search** | `wildgecu.yaml` accepts `google_search: true` for Gemini | Expose as explicit `web_search` tool that uses Gemini grounding when active provider is Gemini, falls back to Tavily/Serper otherwise. |
| **Cache markers** | Provider abstractions | Anthropic / OpenAI / Gemini all support prompt caching; emit cache breakpoints around the system prompt prefix and the tool list. |
| **Outer-loop / Ralph pattern** | `ralph/*.sh` | This is exactly Aider's architect/editor split, externalized. Productize as `wildgecu work-issue <N>` with the architect being one model and editor another, sharing the Soul. |
| **Subagent registry** | `pkg/agent/tools/subagent.go:14` `defaultSubagentSystemPrompt = "You are a helpful assistant..."` | Replace with named subagent types loaded from `.wildgecu/agents/*.md` (Claude Code's pattern). Anonymous spawn keeps depth-1; named types can recurse. |
| **Slash commands infrastructure** | `pkg/command/` | Add `/cost` (token use), `/model` (mid-session switch), `/diff` (changes since session start), `/plan`, `/compact`. All trivial given existing parser/registry. |

## Strategic positioning question

The README pitches "AI agent framework" with cron, daemon, telegram, skills, soul/memory — that's framework-first. The `code` mode + `ralph/` + file tools say coding-agent. The truly distinctive features (Soul, persistent identity, daemon, Telegram bridge, cron) are the framework side.

**If framework-first** (recommended): differentiator is "safe-by-design personal agent that's also good at coding". Priority order below assumes this. Coding parity is table-stakes, not the bet.

**If coding-first**: flip Tier 5 to Tier 1. But then you're head-to-head with Crush (also Go) without obvious advantage.

## Prioritized roadmap (framework-first interpretation)

### Tier 0 — One-shot strategic decision
- Pick framework-first vs coding-first. Affects everything below.

### Tier 1 — Deliver "safe by design" (the brand promise)
1. **Permission layer with side-effect classes** — each tool declares `pure | fs_read | fs_write | network | shell | destructive`. Config has `auto_allow | ask | deny` per class with glob/regex overrides (e.g., `Bash(npm:*) = auto_allow`). TUI shows inline approval with "Allow once / Always / Deny". Decisions persist in `.wildgecu/permissions.yaml`. Wraps `provider.ToolExecutor`.
2. **Hooks framework** (start with 4 events: PreToolUse, PostToolUse, SessionStart, Stop). Shell scripts return JSON to allow/deny/inject-context. Configured in `~/.wildgecu/hooks/`.
3. **Workspace checkpoints** — `git stash` per write tool call, surfaced via `/undo`. ~1 day.
4. **Bash hardening** — per-call timeout (default 30s, max 600s), `run_in_background` + `read_output`, output cap (~30k chars head+tail), denylist regex.

### Tier 2 — Join the ecosystem
5. **MCP client** — multi-week, but unlocks 100s of community tools (filesystem, github, postgres, slack, linear, sentry, …) without writing a single Go tool.
6. **`AGENTS.md` walk** — ~2 hours. Insert between MEMORY and USER in `BuildSystemPrompt`. Free interop with ~20k existing repos including this one's `CLAUDE.md`.
7. **Prompt fragment assembler** — half a day. Generalize `BuildSystemPrompt` from string concatenation to a fragment registry with conditional gates (mode, provider, feature flags). Unlocks everything below.
8. **Per-provider system prompts** — half a day on top of #7. `anthropic.md` / `openai.md` / `gemini.md` / `ollama.md` / `generic.md`. Direct delivery on the multi-provider promise.

### Tier 3 — Coding table-stakes
9. **`grep` + `glob` + `multi_edit`** — few hours each. Mirror Claude Code's API so models trained on those patterns work better.
10. **`read_file` image support** — drop the non-UTF8 reject in `files.go:115`; pass through to vision-capable models.
11. **`web_search` tool** — finish the half-wired `google_search: true` config. ~half a day.

### Tier 4 — Context engineering (pure cost/quality wins, no positioning bet)
12. **Prompt caching** — emit cache markers around system prompt + tool list. Big input-token cost cut on long sessions where SOUL/MEMORY/AGENT are stable.
13. **Wire up `RequestReminder`** — already-built mechanism, just needs reminders to populate it. Each one is ~10-30 lines.
14. **`/compact` + 5-bucket continuation summary** (Goal / Constraints / Progress / Key Decisions / Failed Approaches). The "failed approaches" line specifically prevents resume loops.
15. **Token-math abstraction** — replace ad-hoc heuristics with `usable = total − reserved_output(32k) − safety(20k)`. ~30 lines, makes overflow handling testable.
16. **Slash commands**: `/cost` (token use), `/model` (mid-session switch), `/diff` (changes since session start), `/plan`, `/compact`. All trivial given existing infrastructure.
17. **`AskUserQuestion` tool** — first-class structured prompt for the agent to pause and ask. Composable with plan mode and approvals.
18. **`save_memory` tool** (Gemini-style) — let the agent itself persist facts mid-session, complementing the post-session memory agent.

### Tier 5 — Coding excellence (only if going coding-first)
19. **Aider-style repo-map** — tree-sitter (Go bindings exist) + PageRank, fit to a token budget. Aider claims 4-6% context utilization vs 54-70% for iterative-search agents.
20. **Edit-format dispatch per model** — start with `update_file` (string-replace) + `unified_diff`; route by provider family. udiff for GPT-4 Turbo (less lazy), search/replace for most, fenced for Gemini.
21. **Architect/editor split as a built-in pattern** — different model for planning vs code generation. Productize the `ralph/` shell scripts as a `wildgecu work-issue` command using `Roles.architect` + `Roles.editor`.
22. **Subagent types as files** (`.wildgecu/agents/*.md`) — anonymous `spawn_agent` stays depth-1; named types can recurse. Mirrors Claude Code.

## Tensions and tradeoffs the field disagrees on

| Question | Field positions | Decision rule for WildGecu |
|---|---|---|
| Verbose vs terse tool descriptions | Claude Code/OpenCode keep verbose (defensive); Crush slashed by ~120k tokens (bets on strong models) | **Per-provider verbosity.** Verbose default for Ollama/older; terse overlay for Sonnet 4.x / GPT-5 / Gemini 2.5. Composes naturally with per-provider prompts. |
| One edit format vs many | Aider routes per model | Start with two (current + udiff); skip the long tail until benchmarks show wins. |
| Tools-as-native vs tools-as-XML | Continue uses XML to work around weak tool-calling | Native by default; XML fallback only for Ollama models with broken tool support. Detect at provider init. |
| Per-session memory curation vs threshold | Universal: not per-session. WildGecu currently per-session | **Mode-dependent.** Keep per-session for chat (personal AI use). Skip in code mode unless N tool-calls or N tokens exceeded. |
| Subagent recursion | Claude Code: forbidden for ad-hoc, allowed for named types. WildGecu: forbidden universally | Mirror Claude Code: named types can recurse, anonymous `spawn_agent` stays depth-1. |
| OS-native sandbox vs per-tool approval | Codex/Claude Code: OS-native. Cline: none, widely used anyway | Start with permission layer + denylist + reversibility classification. OS-native (landlock/seatbelt) later as opt-in. |
| Plan mode: prompt-only or read-only tool gating | Universal: belt-and-suspenders | Both. Inject system reminder via `RequestReminder` AND swap to a `Subset`-restricted tool registry. |
| Subagent context: same session vs new session | Crush: new session (cleaner). Claude Code: isolated context, conceptually same session | New session — cleaner, easier to test, easier to fan out. (Already what `subagent.go:99` does.) |

## Prompt-quality notes (Claude Code conventions worth lifting verbatim)

These are short, load-bearing instructions from Claude Code's prompt that map almost 1:1 to WildGecu's needs. Drop them into `AGENT.md` and `CODE_AGENT.md`:

- "Text you output outside of tool use is displayed to the user as terminal markdown."
- "Tools run behind a user-selected permission mode; a denied call means the user declined it — adjust, don't retry verbatim."
- "If the conversation grows long, automatic context compaction will be triggered."
- "Prefer the dedicated file/search tools over shell commands when one fits."
- "Independent tool calls can run in parallel in one response."
- "Reference code as `file_path:line_number` — it's clickable."
- "Assume users can't see most tool calls or thinking — only your text output. Before your first tool call, state in one sentence what you're about to do."
- "Default to writing no comments. Never write multi-paragraph docstrings or multi-line comment blocks — one short line max."
- "End-of-turn summary: one or two sentences covering what changed and what's next — nothing more."

## Single-paragraph strategic recommendation

If forced to pick *one* thing, build the **prompt fragment assembler** first. Claude Code's pattern of ~110 conditional fragments isn't fancy — it's the unlock. Once you have it, per-provider prompts are trivial, per-mode prompts (chat/code/plan/bootstrap/memory) become refactorable, conditional feature blocks (telegram-on / MCP-on / skills-loaded) drop in cleanly, system reminders piggyback on the same primitives, and Crush-style aggressive shrinking becomes a per-model overlay rather than a rewrite. Without it, AGENT.md and CODE_AGENT.md will keep drifting and duplicating. After that: permission layer + hooks (deliver "safe by design"), MCP client (join the ecosystem), AGENTS.md walk (free interop), per-provider prompts (the multi-provider promise made real). Coding-table-stakes (grep / glob / multi_edit) are easy day-of work whenever you want them.

## Sources

- [Claude Code system prompts (Piebald-AI extraction)](https://github.com/Piebald-AI/claude-code-system-prompts)
- [How Claude Code Builds a System Prompt — Drew Breunig](https://www.dbreunig.com/2026/04/04/how-claude-code-builds-a-system-prompt.html)
- [Under the Hood of Claude Code — Pierce Freeman](https://pierce.dev/notes/under-the-hood-of-claude-code)
- [Claude Code subagents docs](https://code.claude.com/docs/en/sub-agents)
- [Claude Code sandboxing docs](https://code.claude.com/docs/en/sandboxing)
- [Claude Code hooks reference](https://code.claude.com/docs/en/hooks)
- [Claude Code plan mode (ClaudeLog)](https://claudelog.com/mechanics/plan-mode/)
- [OpenCode prompt construction gist (rmk40)](https://gist.github.com/rmk40/cde7a98c1c90614a27478216cc01551f)
- [OpenCode context management & compaction (DeepWiki)](https://deepwiki.com/sst/opencode/2.4-context-management-and-compaction)
- [OpenCode agents docs](https://opencode.ai/docs/agents/)
- [OpenCode tools docs](https://opencode.ai/docs/tools/)
- [OpenCode MCP servers docs](https://opencode.ai/docs/mcp-servers/)
- [Crush GitHub repo](https://github.com/charmbracelet/crush)
- [Crush DeepWiki overview](https://deepwiki.com/charmbracelet/crush)
- [Crush agent delegation (DeepWiki)](https://deepwiki.com/charmbracelet/crush/6.7-agent-delegation-and-nested-tools)
- [Crush permission system source](https://github.com/charmbracelet/crush/blob/main/internal/permission/permission.go)
- [Charmbracelet Fantasy](https://github.com/charmbracelet/fantasy)
- [Aider edit formats](https://aider.chat/docs/more/edit-formats.html)
- [Aider architect/editor split](https://aider.chat/2024/09/26/architect.html)
- [Aider repo-map with tree-sitter](https://aider.chat/2023/10/22/repomap.html)
- [Cline GitHub](https://github.com/cline/cline)
- [Cline Plan & Act docs](https://docs.cline.bot/core-workflows/plan-and-act)
- [Goose architecture/extensions](https://block.github.io/goose/docs/goose-architecture/extensions-design/)
- [Goose subrecipes](https://block.github.io/goose/docs/guides/recipes/subrecipes/)
- [Codex CLI features](https://developers.openai.com/codex/cli/features)
- [Codex CLI sandboxing](https://developers.openai.com/codex/concepts/sandboxing)
- [Codex CLI AGENTS.md guide](https://developers.openai.com/codex/guides/agents-md)
- [OpenAI: Unrolling the Codex agent loop](https://openai.com/index/unrolling-the-codex-agent-loop/)
- [Gemini CLI tools API](https://google-gemini.github.io/gemini-cli/docs/core/tools-api.html)
- [Gemini CLI tools list](https://google-gemini.github.io/gemini-cli/docs/tools/)
- [Continue agent how-it-works](https://docs.continue.dev/ide-extensions/agent/how-it-works)
- [AGENTS.md spec site](https://agents.md/)
- [InfoQ: AGENTS.md as open standard](https://www.infoq.com/news/2025/08/agents-md/)
- [OpenClaw GitHub](https://github.com/openclaw/openclaw)
