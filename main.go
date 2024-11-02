package main

import (
	"fmt"
	"log"
	"os"

	"github.com/AccursedGalaxy/streakode/cache"
	"github.com/AccursedGalaxy/streakode/cmd"
	"github.com/AccursedGalaxy/streakode/config"
	"github.com/AccursedGalaxy/streakode/scan"
	"github.com/spf13/cobra"
)

func main() {
	config.LoadConfig("")
	cacheFilePath := os.Getenv("HOME") + "/.streakode_cache.json"

	// Init cache and load from file
	cache.InitCache()
	cache.LoadCache(cacheFilePath)

	// Refresh cache periodically (or on command)
	author := config.AppConfig.Author
	cache.RefreshCache(config.AppConfig.ScanDirectories, author, cacheFilePath)

	// Scan directories and throw error if none found
	repos, err := scan.ScanDirectories(config.AppConfig.ScanDirectories, author)
	if err != nil {
		log.Fatalf("Error scanning directories: %v", err)
	}

	// Log info if no active repos found
	if len(repos) == 0 {
		fmt.Println("No active repositories found!")
	}


	rootCmd := &cobra.Command{Use: "streakode"}
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
			err := cache.RefreshCache(config.AppConfig.ScanDirectories, author, cacheFilePath)
			if err == nil {
				fmt.Println("✨ Cache refreshed successfully!")
			}
		},
	}

	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(refreshCmd)
	rootCmd.Execute()
}
