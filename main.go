package main

import (
	"fmt"
	"os"
	"os/exec"
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
		Use:   "author",
		Short: "Show configured Git author information",
		Run: func(cmd *cobra.Command, args []string) {
			// Check global git config
			globalName, _ := exec.Command("git", "config", "--global", "user.name").Output()
			globalEmail, _ := exec.Command("git", "config", "--global", "user.email").Output()

			fmt.Println("Global Git Configuration:")
			fmt.Printf("Name:  %s", string(globalName))
			fmt.Printf("Email: %s", string(globalEmail))

			// Check local git config if in a repository
			if isGitRepo, _ := exec.Command("git", "rev-parse", "--is-inside-work-tree").Output(); len(isGitRepo) > 0 {
				localName, _ := exec.Command("git", "config", "user.name").Output()
				localEmail, _ := exec.Command("git", "config", "user.email").Output()

				if len(localName) > 0 || len(localEmail) > 0 {
					fmt.Println("\nLocal Repository Configuration:")
					if len(localName) > 0 {
						fmt.Printf("Name:  %s", string(localName))
					}
					if len(localEmail) > 0 {
						fmt.Printf("Email: %s", string(localEmail))
					}
				}
			}
		},
	}

	// Add history command
	historyCmd := &cobra.Command{
		Use:   "history [command]",
		Short: "Interactive Git history search and exploration",
		Long: `Interactive Git history search and exploration with powerful filtering and viewing options.

This command provides a fast, interactive interface to explore your Git history across all repositories.
It uses fuzzy finding (fzf) for instant searching through commits, with results loading progressively
as they become available.

Key Features:
- Instant interactive fuzzy search through commit history
- Progressive loading of results for immediate responsiveness
- Rich commit preview with diff and stats
- Filter by repository, author, or time range
- Multiple view formats and sorting options
- Smart caching for faster subsequent searches
- Keyboard shortcuts for efficient navigation

Navigation:
- Type to filter commits instantly
- Ctrl-a: Toggle select all commits
- Ctrl-d/u: Page down/up
- Ctrl-/: Toggle preview panel
- Enter: Select commit(s)
- Esc: Exit search`,
		Example: `  # Interactive search through all commits (last 7 days)
  streakode history

  # Search commits from a specific author
  streakode history -a "Your Name"

  # Search in a specific repository
  streakode history repo myrepo

  # View recent activity (last 24 hours)
  streakode history recent

  # Search through file changes
  streakode history files

  # View commit activity stats
  streakode history stats

  # Show detailed history for the last 30 days
  streakode history --days 30 --format detailed`,
		Run: func(cobraCmd *cobra.Command, args []string) {
			var opts cmd.HistoryOptions

			// Get flags
			author, _ := cobraCmd.Flags().GetString("author")
			days, _ := cobraCmd.Flags().GetInt("days")
			format, _ := cobraCmd.Flags().GetString("format")

			// Set options
			opts.Author = author
			opts.Days = days
			opts.Format = format
			if days == 0 {
				opts.Days = 7 // default to 7 days
			}

			cmd.DisplayHistory(opts)
		},
	}

	// Add subcommands
	historyRepoCmd := &cobra.Command{
		Use:   "repo [repository-name]",
		Short: "Search history in a specific repository",
		Long: `Search through Git history of a specific repository.

The search interface will open immediately and results will load progressively
as they become available. Local commits are shown first, followed by remote commits
if available.`,
		Args: cobra.ExactArgs(1),
		Run: func(cobraCmd *cobra.Command, args []string) {
			var opts cmd.HistoryOptions
			opts.Repository = args[0]

			// Get flags
			author, _ := cobraCmd.Flags().GetString("author")
			days, _ := cobraCmd.Flags().GetInt("days")
			format, _ := cobraCmd.Flags().GetString("format")

			opts.Author = author
			opts.Days = days
			opts.Format = format
			if days == 0 {
				opts.Days = 7
			}

			cmd.DisplayHistory(opts)
		},
	}

	historyRecentCmd := &cobra.Command{
		Use:   "recent",
		Short: "Show recent commit activity (last 24 hours)",
		Long: `Display Git activity from the last 24 hours across all repositories.

Results are shown in detailed format by default and load progressively for
immediate responsiveness.`,
		Run: func(cobraCmd *cobra.Command, args []string) {
			var opts cmd.HistoryOptions
			opts.Days = 1
			opts.Format = "detailed"

			author, _ := cobraCmd.Flags().GetString("author")
			opts.Author = author

			cmd.DisplayHistory(opts)
		},
	}

	historyFilesCmd := &cobra.Command{
		Use:   "files [pattern]",
		Short: "Search through file changes",
		Long: `Search through Git history focusing on file changes.

This command allows you to search through commit history with an emphasis on
file changes. Results show detailed file statistics and can be filtered by
file patterns.

The interface opens immediately with progressive loading of results for
optimal responsiveness.`,
		Run: func(cobraCmd *cobra.Command, args []string) {
			var opts cmd.HistoryOptions
			opts.Format = "files"

			if len(args) > 0 {
				opts.Query = args[0]
			}

			author, _ := cobraCmd.Flags().GetString("author")
			days, _ := cobraCmd.Flags().GetInt("days")
			opts.Author = author
			opts.Days = days
			if days == 0 {
				opts.Days = 7
			}

			cmd.DisplayHistory(opts)
		},
	}

	historyStatsCmd := &cobra.Command{
		Use:   "stats",
		Short: "Show commit activity statistics",
		Long: `Display detailed Git activity statistics across repositories.

This command provides a statistical overview of Git activity, including:
- Commit frequency and patterns
- File change statistics
- Author activity
- Repository contributions

Results are loaded progressively and shown in a detailed format optimized
for statistical analysis. Default time range is 30 days.`,
		Run: func(cobraCmd *cobra.Command, args []string) {
			var opts cmd.HistoryOptions
			opts.Format = "stats"

			author, _ := cobraCmd.Flags().GetString("author")
			days, _ := cobraCmd.Flags().GetInt("days")
			opts.Author = author
			opts.Days = days
			if days == 0 {
				opts.Days = 30 // default to 30 days for stats
			}

			cmd.DisplayHistory(opts)
		},
	}

	// Add flags to history command
	historyCmd.PersistentFlags().StringP("author", "a", "", "Filter history by author")
	historyCmd.PersistentFlags().Int("days", 7, "Number of days of history to show")
	historyCmd.PersistentFlags().String("format", "default", "Output format (default, detailed, compact)")

	// Add subcommands to history
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
