# gonesis

A bootstrappable AI agent with personality and identity, built in Go. On first run, the agent interviews you to discover who it should be, then writes its own soul to disk. Every session after that, it wakes up already knowing itself.

## Features

- **Soul system** — The agent bootstraps its own identity through a conversational interview, stored as `SOUL.md`
- **Provider abstraction** — LLM-agnostic design behind a simple `Provider` interface (ships with Google Gemini)
- **Streaming TUI** — Real-time chat interface built with [Bubble Tea](https://github.com/charmbracelet/bubbletea), with streaming token output
- **Tool framework** — Agents can call tools during conversation; the bootstrap itself uses a `write_soul` tool
- **Agent loop** — Built-in agentic loop that handles tool calls, execution, and re-prompting automatically

## How it works

```
First run:                              Every run after:

┌─────────────┐                         ┌─────────────┐
│  No SOUL.md │                         │ Load SOUL.md│
└──────┬──────┘                         └──────┬──────┘
       │                                       │
       ▼                                       ▼
┌─────────────────┐                     ┌─────────────────┐
│   Bootstrap TUI │                     │  Build system   │
│  (interview you)│                     │  prompt from    │
│                 │                     │  AGENT + SOUL   │
└──────┬──────────┘                     │  + USER (opt.)  │
       │                                └──────┬──────────┘
       ▼                                       │
┌─────────────────┐                            ▼
│  Agent calls    │                     ┌─────────────────┐
│  write_soul     │                     │    Chat TUI     │
│  → .gonesis/    │                     │  (normal mode)  │
│    SOUL.md      │                     └─────────────────┘
└──────┬──────────┘
       │
       ▼
    Chat TUI
```

**Bootstrap phase**: The agent receives a system prompt (BOOTSTRAP.md) that guides it to ask about your agent's name, purpose, personality, expertise, and boundaries. After a few exchanges, it calls the `write_soul` tool to persist its identity.

**Normal mode**: The system prompt is assembled from three parts — base behavior (AGENT.md), the agent's identity (SOUL.md), and optional user preferences (USER.md).

## Prerequisites

- Go 1.25.5+
- A [Google Gemini API key](https://aistudio.google.com/apikey)

## Getting started

```bash
git clone https://github.com/ludusrusso/gonesis.git
cd gonesis

export GEMINI_API_KEY="your-api-key"

go run .
```

On first run, the agent will start a bootstrap conversation to establish its identity. Answer a few questions and it will write `.gonesis/SOUL.md` automatically, then switch to normal chat mode.

## Configuration

All instance-specific files live in `.gonesis/` (gitignored by default):

| File | Purpose |
|---|---|
| `.gonesis/SOUL.md` | Agent identity — created during bootstrap |
| `.gonesis/USER.md` | Optional user preferences — create manually to pass context about yourself |

Delete `SOUL.md` to re-run the bootstrap and give your agent a new identity.

## Project structure

```
gonesis/
├── main.go                # Entry point: env, provider setup, agent.Run()
├── agent/
│   ├── agent.go           # Run() function — orchestrates bootstrap → chat flow
│   ├── bootstrap.go       # Bootstrap interview config and write_soul tool
│   ├── soul.go            # Soul I/O and system prompt assembly
│   ├── prompt.go          # Embeds AGENT.md and BOOTSTRAP.md
│   ├── AGENT.md           # Base agent behavior prompt
│   └── BOOTSTRAP.md       # Bootstrap conversation prompt
├── provider/
│   ├── provider.go        # Provider interface, Message, Tool, Response types
│   ├── agent.go           # RunAgentLoop / RunAgentLoopStream
│   └── gemini/
│       └── gemini.go      # Google Gemini implementation
├── chat/
│   └── chat.go            # Config, RunTurn, RunTurnStream
└── tui/
    ├── tui.go             # Bubble Tea Model, Init, Update, View
    ├── messages.go        # Internal message types
    └── styles.go          # Lipgloss styling
```

## Adding a new provider

Implement the `provider.Provider` interface:

```go
type Provider interface {
    Generate(ctx context.Context, params *GenerateParams) (*Response, error)
}
```

For streaming support, also implement `StreamProvider`:

```go
type StreamProvider interface {
    Provider
    GenerateStream(ctx context.Context, params *GenerateParams) (<-chan StreamChunk, <-chan error)
}
```

Then wire it up in `main.go` instead of the Gemini provider.

## License

See [LICENSE](LICENSE) for details.
