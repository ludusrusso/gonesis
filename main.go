package main

import (
	"context"
	"log"
	"os"

	"gonesis/agent"
	"gonesis/provider/gemini"
)

func main() {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY env var required")
	}

	ctx := context.Background()

	p, err := gemini.New(ctx, apiKey, "gemini-3-flash-preview")
	if err != nil {
		log.Fatal(err)
	}

	baseDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	if err := agent.Run(ctx, agent.Config{
		Provider: p,
		BaseDir:  baseDir,
	}); err != nil {
		log.Fatal(err)
	}
}
