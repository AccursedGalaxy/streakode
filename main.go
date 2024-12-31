package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/AccursedGalaxy/streakode/cache"
	"github.com/AccursedGalaxy/streakode/cmd"
	"github.com/AccursedGalaxy/streakode/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

/*
TODO:
- Add automatic update functionality (manually code this cuz it's fun)
- Add easy installation script (curl | bash)
*/

var Version = "dev" // This will be overwritten during build

type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func getCacheFilePath(profile string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	if profile == "" {
		return filepath.Join(home, ".streakode.cache")
	}
	return filepath.Join(home, fmt.Sprintf(".streakode_%s.cache", profile))
}

func ensureCacheRefresh() error {
	// Skip if no refresh interval is configured
	if config.AppConfig.RefreshInterval <= 0 {
		return nil
	}

	interval := time.Duration(config.AppConfig.RefreshInterval) * time.Minute

	// Quick check if refresh is needed
	if cache.QuickNeedsRefresh(interval) {
		cacheFilePath := getCacheFilePath(config.AppState.ActiveProfile)

		// For commands that need fresh data, use sync refresh
		if requiresFreshData() {
			return cache.RefreshCache(
				config.AppConfig.ScanDirectories,
				config.AppConfig.Author,
				cacheFilePath,
				config.AppConfig.ScanSettings.ExcludedPatterns,
				config.AppConfig.ScanSettings.ExcludedPaths,
			)
		}

		// For other commands, use async refresh
		cache.AsyncRefreshCache(
			config.AppConfig.ScanDirectories,
			config.AppConfig.Author,
			cacheFilePath,
			config.AppConfig.ScanSettings.ExcludedPatterns,
			config.AppConfig.ScanSettings.ExcludedPaths,
		)
	}
	return nil
}

func requiresFreshData() bool {
	// Get the command being executed
	cmd := os.Args[1]

	// List of commands that need fresh data
	freshDataCommands := map[string]bool{
		"stats":  true,
		"reload": true,
	}

	return freshDataCommands[cmd]
}

func main() {
	var (
		profile string
		debug   bool
	)

	rootCmd := &cobra.Command{
		Use:     "streakode",
		Short:   "A Git activity tracker for monitoring coding streaks",
		Version: Version,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Load the state first to get the active profile
			if err := config.LoadState(); err != nil {
				fmt.Printf("Error loading state: %v\n", err)
			}

			// Set debug mode from flag
			config.AppConfig.Debug = debug
			if debug {
				fmt.Println("Debug mode enabled")
			}

			// Use AppState.ActiveProfile instead of the profile flag
			cacheFilePath := getCacheFilePath(config.AppState.ActiveProfile)
			config.LoadConfig(config.AppState.ActiveProfile)
			cache.InitCache()
			if err := cache.LoadCache(cacheFilePath); err != nil {
				fmt.Printf("Error loading cache: %v\n", err)
			}

			if err := ensureCacheRefresh(); err != nil {
				fmt.Printf("Error refreshing cache: %v\n", err)
			}
		},
	}

	// Add persistent flags to root command
	rootCmd.PersistentFlags().StringVarP(&profile, "profile", "p", "", "Config profile to use (e.g., work, home)")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enable debug mode")

	statsCmd := &cobra.Command{
		Use:   "stats [repository]",
		Short: "Display stats for all active repositories or a specific repository",
		Long: `Display Git activity statistics for your repositories.

Without arguments, shows stats for all active repositories.
With a repository name argument, shows detailed stats for just that repository.

Example:
  streakode stats             # Show stats for all repositories
  streakode stats myproject   # Show stats for only the myproject repository`,
		Args: cobra.MaximumNArgs(1),
		Run: func(cobraCmd *cobra.Command, args []string) {
			var targetRepo string
			if len(args) > 0 {
				targetRepo = args[0]
			}
			cmd.DisplayStats(targetRepo)
		},
	}

	// Define cache command and its subcommands
	cacheCmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage the streakode cache",
	}

	reloadCmd := &cobra.Command{
		Use:   "reload",
		Short: "Reload the streakode cache with fresh data",
		Run: func(cobraCmd *cobra.Command, args []string) {
			if config.AppConfig.Debug {
				fmt.Println("Debug: Starting cache reload...")
			}
			cacheFilePath := getCacheFilePath(profile)
			err := cache.RefreshCache(
				config.AppConfig.ScanDirectories,
				config.AppConfig.Author,
				cacheFilePath,
				config.AppConfig.ScanSettings.ExcludedPatterns,
				config.AppConfig.ScanSettings.ExcludedPaths,
			)
			if err == nil {
				fmt.Println("✨ Cache reloaded successfully!")
			} else {
				fmt.Printf("Error reloading cache: %v\n", err)
			}
		},
	}

	cleanCmd := &cobra.Command{
		Use:   "clean",
		Short: "Remove the streakode cache",
		Run: func(cobraCmd *cobra.Command, args []string) {
			if config.AppConfig.Debug {
				fmt.Println("Debug: Starting cache cleanup...")
			}
			cacheFilePath := getCacheFilePath(profile)
			if err := cache.CleanCache(cacheFilePath); err != nil {
				fmt.Printf("Error cleaning cache: %v\n", err)
			} else {
				fmt.Println("🧹 Cache cleaned successfully!")
			}
		},
	}

	// Add subcommands to cache command
	cacheCmd.AddCommand(reloadCmd)
	cacheCmd.AddCommand(cleanCmd)

	profileCmd := &cobra.Command{
		Use:   "profile [name]",
		Short: "Set or show current profile",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				if config.AppState.ActiveProfile == "" {
					fmt.Println("Using default profile")
				} else {
					fmt.Printf("Using profile: %s\n", config.AppState.ActiveProfile)
				}
				return
			}

			newProfile := args[0]
			if newProfile == "default" || newProfile == "-" {
				newProfile = ""
			}

			// Try to load the new profile's config first
			viper.Reset()
			viper.AddConfigPath("$HOME")
			viper.SetConfigType("yaml")

			// Set config name based on profile
			configName := ".streakodeconfig"
			if newProfile != "" {
				configName = ".streakodeconfig_" + newProfile
			}
			viper.SetConfigName(configName)

			// Try to read the config file
			if err := viper.ReadInConfig(); err != nil {
				fmt.Printf("Error: Could not load profile '%s': %v\n", newProfile, err)
				os.Exit(1)
			}

			// Try to unmarshal and validate the config
			var newConfig config.Config
			if err := viper.Unmarshal(&newConfig); err != nil {
				fmt.Printf("Error: Invalid config format for profile '%s': %v\n", newProfile, err)
				os.Exit(1)
			}

			// Validate the config
			if err := newConfig.ValidateConfig(); err != nil {
				fmt.Printf("Error: Invalid configuration for profile '%s': %v\n", newProfile, err)
				os.Exit(1)
			}

			// If we get here, the config is valid, so we can update the state
			if newProfile == "" {
				fmt.Println("Switched to default profile")
			} else {
				fmt.Printf("Switched to profile: %s\n", newProfile)
			}

			config.AppState.ActiveProfile = newProfile
			if err := config.SaveState(); err != nil {
				fmt.Printf("Warning: Could not save profile state: %v\n", err)
			}

			// Refresh cache for new profile
			cacheFilePath := getCacheFilePath(newProfile)
			cache.InitCache()
			cache.LoadCache(cacheFilePath)
			cache.RefreshCache(
				config.AppConfig.ScanDirectories,
				config.AppConfig.Author,
				cacheFilePath,
				config.AppConfig.ScanSettings.ExcludedPatterns,
				config.AppConfig.ScanSettings.ExcludedPaths,
			)
		},
	}

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Show streakode version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Streakode version %s\n", Version)
		},
	}

	authorCmd := &cobra.Command{
		Use:   "author [name]",
		Short: "Show detailed Git author information and statistics",
		Long: `Display detailed Git author information and statistics.

Without arguments, shows stats for the configured author.
With an author name argument, shows stats for the specified author.

Example:
  streakode author             # Show stats for configured author
  streakode author "John Doe"  # Show stats for John Doe`,
		Run: func(cobraCmd *cobra.Command, args []string) {
			var targetAuthor string
			if len(args) > 0 {
				targetAuthor = args[0]
			}
			cmd.DisplayAuthorInfo(targetAuthor)
		},
	}

	// Add history command
	historyCmd := &cobra.Command{
		Use:   "history [flags]",
		Short: "Interactive Git history search",
		Long: `Search and explore your Git commit history interactively.

Uses fuzzy search to quickly find commits across all repositories.
Press '?' while searching to see keyboard shortcuts.`,
		Example: `  sk history                  # Show commits from last 7 days
  sk history --days 30        # Show last 30 days
  sk history author robin     # Show commits by author
  sk history repo myproject   # Show commits in repository`,
		Run: func(cobraCmd *cobra.Command, args []string) {
			var opts cmd.HistoryOptions
			days, _ := cobraCmd.Flags().GetInt("days")
			format, _ := cobraCmd.Flags().GetString("format")
			opts.Days = days
			opts.Format = format
			if days == 0 {
				opts.Days = 7
			}
			cmd.DisplayHistory(opts)
		},
	}

	// Add persistent flags that will be inherited by all subcommands
	historyCmd.PersistentFlags().IntP("days", "n", 7, "Number of days to show history for")
	historyCmd.PersistentFlags().StringP("format", "f", "default", "Output format (default, detailed, compact)")

	// Add subcommands with cleaner help text
	historyAuthorCmd := &cobra.Command{
		Use:   "author [name]",
		Short: "Show commits by author",
		Example: `  sk history author robin     # Show Robin's commits
  sk history author "John D"  # Show John D's commits`,
		Args: cobra.ExactArgs(1),
		Run: func(cobraCmd *cobra.Command, args []string) {
			var opts cmd.HistoryOptions
			opts.Author = args[0]
			days, _ := cobraCmd.PersistentFlags().GetInt("days")
			format, _ := cobraCmd.PersistentFlags().GetString("format")
			opts.Days = days
			opts.Format = format
			if days == 0 {
				opts.Days = 14
			}
			cmd.DisplayHistory(opts)
		},
	}

	historyRepoCmd := &cobra.Command{
		Use:   "repo [name]",
		Short: "Show commits in repository",
		Example: `  sk history repo myproject   # Show commits in myproject
  sk history repo webapp     # Show commits in webapp`,
		Args: cobra.ExactArgs(1),
		Run: func(cobraCmd *cobra.Command, args []string) {
			var opts cmd.HistoryOptions
			opts.Repository = args[0]
			days, _ := cobraCmd.PersistentFlags().GetInt("days")
			format, _ := cobraCmd.PersistentFlags().GetString("format")
			opts.Days = days
			opts.Format = format
			if days == 0 {
				opts.Days = 14
			}
			cmd.DisplayHistory(opts)
		},
	}

	historyRecentCmd := &cobra.Command{
		Use:   "recent",
		Short: "Show commits from last 24 hours",
		Run: func(cobraCmd *cobra.Command, args []string) {
			var opts cmd.HistoryOptions
			opts.Days = 1
			opts.Format = "detailed"
			cmd.DisplayHistory(opts)
		},
	}

	historyFilesCmd := &cobra.Command{
		Use:   "files [pattern]",
		Short: "Search commits by changed files",
		Example: `  sk history files "*.go"     # Show commits changing Go files
  sk history files config    # Show commits changing config files`,
		Run: func(cobraCmd *cobra.Command, args []string) {
			var opts cmd.HistoryOptions
			opts.Format = "files"
			if len(args) > 0 {
				opts.Query = args[0]
			}
			days, _ := cobraCmd.PersistentFlags().GetInt("days")
			opts.Days = days
			if days == 0 {
				opts.Days = 7
			}
			cmd.DisplayHistory(opts)
		},
	}

	historyStatsCmd := &cobra.Command{
		Use:   "stats",
		Short: "Show commit statistics",
		Run: func(cobraCmd *cobra.Command, args []string) {
			var opts cmd.HistoryOptions
			opts.Format = "stats"
			days, _ := cobraCmd.PersistentFlags().GetInt("days")
			opts.Days = days
			if days == 0 {
				opts.Days = 30
			}
			cmd.DisplayHistory(opts)
		},
	}

	// Add subcommands to history command
	historyCmd.AddCommand(historyAuthorCmd)
	historyCmd.AddCommand(historyRepoCmd)
	historyCmd.AddCommand(historyRecentCmd)
	historyCmd.AddCommand(historyFilesCmd)
	historyCmd.AddCommand(historyStatsCmd)

	// Add all commands to root
	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(cacheCmd)
	rootCmd.AddCommand(profileCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(authorCmd)
	rootCmd.AddCommand(historyCmd)
	rootCmd.Execute()
}
