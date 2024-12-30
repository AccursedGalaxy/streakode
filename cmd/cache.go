package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/AccursedGalaxy/streakode/cache"
	"github.com/AccursedGalaxy/streakode/config"
	"github.com/spf13/cobra"
)

func getCacheFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting home directory: %v\n", err)
		os.Exit(1)
	}

	if config.AppState.ActiveProfile == "" {
		return filepath.Join(home, ".streakode.cache")
	}
	return filepath.Join(home, fmt.Sprintf(".streakode_%s.cache", config.AppState.ActiveProfile))
}

// cacheCmd represents the cache command
var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage the repository cache",
	Long: `Cache management commands for Streakode.
This includes operations like reloading and cleaning the cache.`,
}

// reloadCmd represents the reload command
var reloadCmd = &cobra.Command{
	Use:   "reload",
	Short: "Reload the repository cache",
	Long: `Reload scans all configured directories and updates the repository cache.
This is useful when you want to refresh the data without waiting for the automatic refresh.`,
	Run: func(cmd *cobra.Command, args []string) {
		if config.AppConfig.Debug {
			fmt.Println("Debug: Starting cache reload...")
		}
		
		cacheFilePath := getCacheFilePath()
		err := cache.RefreshCache(
			config.AppConfig.ScanDirectories,
			config.AppConfig.Author,
			cacheFilePath,
			config.AppConfig.ScanSettings.ExcludedPatterns,
			config.AppConfig.ScanSettings.ExcludedPaths,
		)
		if err != nil {
			fmt.Printf("Error reloading cache: %v\n", err)
			return
		}
		fmt.Println("✨ Cache reloaded successfully!")
	},
}

// cleanCmd represents the clean command
var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean the repository cache",
	Long: `Clean removes all cached repository data.
This is useful when you want to start fresh or if you encounter any cache-related issues.`,
	Run: func(cmd *cobra.Command, args []string) {
		if config.AppConfig.Debug {
			fmt.Println("Debug: Starting cache cleanup...")
		}
		
		cacheFilePath := getCacheFilePath()
		if err := cache.CleanCache(cacheFilePath); err != nil {
			fmt.Printf("Error cleaning cache: %v\n", err)
			return
		}
		fmt.Println("🧹 Cache cleaned successfully!")
	},
}

func init() {
	rootCmd.AddCommand(cacheCmd)
	cacheCmd.AddCommand(reloadCmd)
	cacheCmd.AddCommand(cleanCmd)
} 