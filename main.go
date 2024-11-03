package main

import (
	"fmt"
	"os"
	"path/filepath"

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

func main() {
	var profile string

	rootCmd := &cobra.Command{
		Use:   "streakode",
		Short: "A Git activity tracker for monitoring coding streaks",
			Version: Version,
			PersistentPreRun: func(cmd *cobra.Command, args []string) {
				cacheFilePath := getCacheFilePath(profile)
				config.LoadConfig(profile)
				cache.InitCache()
				if err := cache.LoadCache(cacheFilePath); err != nil {
					fmt.Printf("Error loading cache: %v\n", err)
				}
				if err := cache.RefreshCache(config.AppConfig.ScanDirectories, config.AppConfig.Author, cacheFilePath); err != nil {
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
	refreshCmd := &cobra.Command{
		Use: "refresh",
		Short: "Refresh the streakode cache",
		Run: func(cobraCmd *cobra.Command, args []string) {
			cacheFilePath := getCacheFilePath(profile)
			err := cache.RefreshCache(config.AppConfig.ScanDirectories, config.AppConfig.Author, cacheFilePath)
			if err == nil {
				fmt.Println("✨ Cache refreshed successfully!")
			}
		},
	}

	// Add profile command
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
			cache.RefreshCache(config.AppConfig.ScanDirectories, config.AppConfig.Author, cacheFilePath)
		},
	}

	// Add version command
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Show streakode version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Streakode version %s\n", Version)
		},
	}

	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(refreshCmd)
	rootCmd.AddCommand(profileCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.Execute()
}
