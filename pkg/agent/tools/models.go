package tools

import (
	"context"
	"slices"

	"github.com/ludusrusso/wildgecu/pkg/provider/tool"
)

// ModelInfo holds model configuration visible to the agent.
type ModelInfo struct {
	DefaultModel string            `json:"default_model"`
	Providers    []string          `json:"providers"`
	Models       map[string]string `json:"models"`
}

// ModelTools returns the list_models tool. Returns nil if info is nil.
func ModelTools(info *ModelInfo) []tool.Tool {
	if info == nil {
		return nil
	}
	return []tool.Tool{newListModelsTool(info)}
}

type listModelsInput struct{}

func newListModelsTool(info *ModelInfo) tool.Tool {
	return tool.NewTool("list_models",
		"List available models, providers, and aliases from the configuration. Use this to discover which models you can pass to spawn_agent.",
		func(_ context.Context, _ listModelsInput) (ModelInfo, error) {
			providers := make([]string, len(info.Providers))
			copy(providers, info.Providers)
			slices.Sort(providers)

			models := make(map[string]string, len(info.Models))
			for k, v := range info.Models {
				models[k] = v
			}

			return ModelInfo{
				DefaultModel: info.DefaultModel,
				Providers:    providers,
				Models:       models,
			}, nil
		},
	)
}
