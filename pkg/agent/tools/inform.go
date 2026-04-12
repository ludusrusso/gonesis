package tools

import (
	"context"

	"github.com/ludusrusso/wildgecu/pkg/provider"
	"github.com/ludusrusso/wildgecu/pkg/provider/tool"
)

// InformTools returns the inform_user tool.
func InformTools() []tool.Tool {
	return []tool.Tool{informUserTool}
}

type informInput struct {
	Message string `json:"message" description:"The message to display to the user"`
}

type informOutput struct{}

var informUserTool = tool.NewTool("inform_user",
	"Send a message to the user without stopping the current task. Use this to provide progress updates during long-running operations.",
	func(ctx context.Context, in informInput) (informOutput, error) {
		if fn := provider.GetInformFunc(ctx); fn != nil {
			fn(in.Message)
		}
		return informOutput{}, nil
	},
)
