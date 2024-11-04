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
	var profile string

	rootCmd := &cobra.Command{
		Use:   "streakode",
		Short: "A Git activity tracker for monitoring coding streaks",
			Version: Version,
			PersistentPreRun: func(cmd *cobra.Command, args []string) {
				// Load the state first to get the active profile
				if err := config.LoadState(); err != nil {
					fmt.Printf("Error loading state: %v\n", err)
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

	// Add profile flag to root command
	rootCmd.PersistentFlags().StringVarP(&profile, "profile", "p", "", "Config profile to use (e.g., work, home)")

	statsCmd := &cobra.Command{
		Use: "stats",
		Short: "Display stats for all active repositories",
		Run: func(cobraCmd *cobra.Command, args []string) {
			cmd.DisplayStats()
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

	// Add all commands to root
	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(cacheCmd)
	rootCmd.AddCommand(profileCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(authorCmd)
	rootCmd.Execute()
}
