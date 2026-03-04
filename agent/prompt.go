package agent

import _ "embed"

//go:embed AGENT.md
var agentPrompt string

//go:embed BOOTSTRAP.md
var bootstrapPrompt string
