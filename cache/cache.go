package cache

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AccursedGalaxy/streakode/scan"
)

// TODO: rego through all the logic where we create/interact with cache, make sure we can handle version upgrades easily without issues.
// -> Currently when a major version upgrade happens we need to manually delete cache file and do a refresh.

// in-memory cache to hold repo metadata during runtime
var Cache map[string]scan.RepoMetadata

// InitCache - Initializes the in memory cache
func InitCache() {
	Cache = make(map[string]scan.RepoMetadata)
}

// LoadCache - loads repository metadata from a JSON cache file into memory
func LoadCache(filePath string) error {
	file, err := os.Open(filePath)
	if os.IsNotExist(err) {
		InitCache()
		return nil
	}
	if err != nil {
		return fmt.Errorf("error opening cache file: %v", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&Cache); err != nil {
		InitCache()
		return nil
	}

	return nil
}

// SaveCache - saves the in-memory cache to a JSON file
func SaveCache(filePath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating directory: %v", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error creating cache file: %v", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(Cache); err != nil {
		return fmt.Errorf("error encoding cache: %v", err)
	}

	return nil
}

// Add new method to check if cache needs refresh
func NeedsRefresh(path string, lastCommit time.Time) bool {
	if cached, exists := Cache[path]; exists {
		// Only refresh if new commits exist
		return lastCommit.After(cached.LastCommit)
	}
	return true
}

// Clean Cache
func CleanCache(cacheFilePath string) error {
	//Reset in-memory cache
	Cache = make(map[string]scan.RepoMetadata)

	// Remove cache file if present
	if err := os.Remove(cacheFilePath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("something went wrong removing the cache file: %v", err)
		}
	}

	return nil
}

// Modified RefreshCache to support exclusions
func RefreshCache(dirs []string, author string, cacheFilePath string, excludedPatterns []string, excludedPaths []string) error {
	// Clean cache and handle potential errors
	if err := CleanCache(cacheFilePath); err != nil {
		return fmt.Errorf("failed to clean cache: %v", err)
	}

	// Create a function to check if a path should be excluded
	shouldExclude := func(path string) bool {
		// Check full path exclusions
		for _, excludedPath := range excludedPaths {
			if strings.HasPrefix(path, excludedPath) {
				return true
			}
		}

		// Check pattern-based exclusions
		for _, pattern := range excludedPatterns {
			if strings.Contains(path, pattern) {
				return true
			}
		}
		return false
	}

	// Filter out excluded directories before scanning
	var filteredDirs []string
	for _, dir := range dirs {
		if !shouldExclude(dir) {
			filteredDirs = append(filteredDirs, dir)
		}
	}

	repos, err := scan.ScanDirectories(filteredDirs, author, shouldExclude)
	if err != nil {
		log.Printf("Error scanning directories: %v", err)
		return err
	}

	// Only update changed repositories
	for _, repo := range repos {
		if NeedsRefresh(repo.Path, repo.LastCommit) {
			Cache[repo.Path] = repo
		}
	}

	return SaveCache(cacheFilePath)
}
