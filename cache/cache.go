package cache

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/AccursedGalaxy/streakode/scan"
)

// TODO: implement functionality to clean the cache on demand if needed.
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

// Modified RefreshCache
func RefreshCache(dirs []string, author string, cacheFilePath string) error {
	if Cache == nil {
		InitCache()
	}

	repos, err := scan.ScanDirectories(dirs, author)
	if err != nil {
		log.Fatalf("Error scanning directories: %v", err)
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
