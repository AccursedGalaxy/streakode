package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/AccursedGalaxy/streakode/cache"
	"github.com/AccursedGalaxy/streakode/cmd"
	"github.com/AccursedGalaxy/streakode/config"
	"github.com/spf13/cobra"
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
				cache.LoadCache(cacheFilePath)
				cache.RefreshCache(config.AppConfig.ScanDirectories, config.AppConfig.Author, cacheFilePath)
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
			
			profile = args[0]
			if profile == "default" || profile == "-" {
				profile = ""
				fmt.Println("Switched to default profile")
			} else {
				fmt.Printf("Switched to profile: %s\n", profile)
			}
			
			config.AppState.ActiveProfile = profile
			if err := config.SaveState(); err != nil {
				fmt.Printf("Warning: Could not save profile state: %v\n", err)
			}
			
			// Reload configuration with new profile
			config.LoadConfig(profile)
			
			// Refresh cache for new profile
			cacheFilePath := getCacheFilePath(profile)
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
