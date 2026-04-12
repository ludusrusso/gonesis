package container

import (
	"context"
	"fmt"

	"github.com/ludusrusso/wildgecu/pkg/provider"
	"github.com/ludusrusso/wildgecu/pkg/provider/gemini"
	"github.com/ludusrusso/wildgecu/pkg/provider/openai"
	"github.com/ludusrusso/wildgecu/x/config"
)

// DefaultFactory creates a provider.Provider from a ProviderConfig using the
// real provider constructors.
func DefaultFactory(ctx context.Context, _, model string, pc config.ProviderConfig) (provider.Provider, error) {
	switch pc.Type {
	case "gemini":
		var opts []gemini.Option
		if pc.GoogleSearch {
			opts = append(opts, gemini.WithGoogleSearch())
		}
		return gemini.New(ctx, pc.APIKey, model, opts...)

	case "openai", "mistral", "regolo":
		var opts []openai.Option
		if pc.BaseURL != "" {
			opts = append(opts, openai.WithBaseURL(pc.BaseURL))
		}
		return openai.New(pc.APIKey, model, opts...), nil

	case "ollama":
		var opts []openai.Option
		if pc.BaseURL != "" {
			opts = append(opts, openai.WithBaseURL(pc.BaseURL))
		}
		return openai.New("", model, opts...), nil

	default:
		return nil, fmt.Errorf("unknown provider type %q", pc.Type)
	}
}
