package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/AccursedGalaxy/streakode/config"
)

var (
	cfgFile string
	debug   bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "streakode",
	Short: "A CLI tool to track your coding streaks and activity",
	Long: `Streakode is a CLI tool that helps you track your coding activity and streaks.
It scans your Git repositories and provides insights about your coding patterns.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Initialize config before any command runs
		config.InitConfig(cfgFile)
		// Set debug mode from flag
		config.AppConfig.Debug = debug
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.streakodeconfig.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "enable debug mode")
} 