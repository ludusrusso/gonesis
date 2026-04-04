package cmd

import (
	"os"

	"wildgecu/x/config"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Version is the build version, settable via -ldflags.
var Version = "dev"

var cfgFile string
var debugFlag bool

var rootCmd = &cobra.Command{
	Use:   "wildgecu",
	Short: "Wildgecu - an AI-powered coding agent",
	RunE:  runChat,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ./wildgecu.yaml)")
	rootCmd.Flags().BoolVar(&debugFlag, "debug", false, "enable debug logging to ~/.wildgecu/debug/<timestamp>.md")
}

func initConfig() {
	viper.SetDefault("model", "gemini-3-flash-preview")
	viper.SetDefault("gemini_api_key", "")
	viper.SetDefault("base_folder", "")

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("wildgecu")
		viper.SetConfigType("yaml")

		if home, err := config.GlobalHome(); err == nil {
			viper.AddConfigPath(home)
		}
	}

	_ = viper.BindEnv("gemini_api_key", "GEMINI_API_KEY")
	viper.AutomaticEnv()

	_ = viper.ReadInConfig()
}

