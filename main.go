package main

import (
	"fmt"
	"log"
	"os"

	"github.com/AccursedGalaxy/streakode/cache"
	"github.com/AccursedGalaxy/streakode/config"
	"github.com/AccursedGalaxy/streakode/scan"
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
}
