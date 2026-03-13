package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"gonesis/agent"
	"gonesis/homer"
	"gonesis/provider/gemini"
	"gonesis/x/config"
	"gonesis/x/telegram"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var telegramCmd = &cobra.Command{
	Use:   "telegram",
	Short: "Start the Telegram bot interface",
	RunE:  runTelegram,
}

func init() {
	rootCmd.AddCommand(telegramCmd)
}

func runTelegram(cmd *cobra.Command, args []string) error {
	if err := ensureConfigFile(); err != nil {
		return err
	}

	token := viper.GetString("telegram_token")
	if token == "" {
		return fmt.Errorf("telegram_token is required: set TELEGRAM_TOKEN env var or add it to gonesis.yaml")
	}

	apiKey := viper.GetString("gemini_api_key")
	if apiKey == "" {
		return fmt.Errorf("gemini_api_key is required")
	}

	model := viper.GetString("model")

	baseDir := viper.GetString("base_folder")
	if baseDir == "" {
		var err error
		baseDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	globalHome, err := config.GlobalHome()
	if err != nil {
		return fmt.Errorf("global home: %w", err)
	}
	home, err := homer.New(globalHome)
	if err != nil {
		return fmt.Errorf("home homer: %w", err)
	}

	workspace, err := homer.New(filepath.Join(baseDir, config.DirName))
	if err != nil {
		return fmt.Errorf("workspace homer: %w", err)
	}

	ctx := context.Background()

	p, err := gemini.New(ctx, apiKey, model)
	if err != nil {
		return err
	}

	skillsHome, err := homer.New(filepath.Join(globalHome, "skills"))
	if err != nil {
		return fmt.Errorf("skills homer: %w", err)
	}

	// Prepariamo l'agente (carica soul, memory, etc.)
	chatCfg, dbg, err := agent.Prepare(ctx, agent.Config{
		Provider:   p,
		Home:       home,
		Workspace:  workspace,
		SkillsHome: skillsHome,
		HomeDir:    globalHome,
		Debug:      true, // Abilitiamo il debug per i bot di default
	})
	if err != nil {
		return err
	}
	if dbg != nil {
		defer dbg.Close()
	}

	bridge, err := telegram.New(token, chatCfg)
	if err != nil {
		return fmt.Errorf("failed to initialize telegram bridge: %w", err)
	}

	return bridge.Run(ctx)
}
