package container

import (
	"context"
	"testing"

	"wildgecu/x/config"
)

func TestDefaultFactory(t *testing.T) {
	ctx := context.Background()

	t.Run("MistralReturnsProvider", func(t *testing.T) {
		pc := config.ProviderConfig{
			Type:    "mistral",
			APIKey:  "mk",
			BaseURL: "https://api.mistral.ai/v1",
		}
		p, err := DefaultFactory(ctx, "mistral", pc)
		if err != nil {
			t.Fatalf("DefaultFactory() error = %v", err)
		}
		if p == nil {
			t.Fatal("DefaultFactory() returned nil provider")
		}
	})

	t.Run("RegoloReturnsProvider", func(t *testing.T) {
		pc := config.ProviderConfig{
			Type:    "regolo",
			APIKey:  "rk",
			BaseURL: "https://api.regolo.ai/v1",
		}
		p, err := DefaultFactory(ctx, "regolo", pc)
		if err != nil {
			t.Fatalf("DefaultFactory() error = %v", err)
		}
		if p == nil {
			t.Fatal("DefaultFactory() returned nil provider")
		}
	})

	t.Run("OllamaReturnsProvider", func(t *testing.T) {
		pc := config.ProviderConfig{
			Type:    "ollama",
			BaseURL: "http://localhost:11434/v1",
		}
		p, err := DefaultFactory(ctx, "ollama", pc)
		if err != nil {
			t.Fatalf("DefaultFactory() error = %v", err)
		}
		if p == nil {
			t.Fatal("DefaultFactory() returned nil provider")
		}
	})

	t.Run("UnknownTypeReturnsError", func(t *testing.T) {
		pc := config.ProviderConfig{Type: "unknown"}
		_, err := DefaultFactory(ctx, "x", pc)
		if err == nil {
			t.Fatal("DefaultFactory() expected error for unknown type, got nil")
		}
	})
}
