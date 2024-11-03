package cache

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/AccursedGalaxy/streakode/scan"
)

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

// RefreshCache - scans directories and updates the cache with new metadata
func RefreshCache(dirs []string, author string, cacheFilePath string) error {
	// Initialize a new empty cache
	InitCache()

	repos, err := scan.ScanDirectories(dirs, author)
	if err != nil {
		log.Fatalf("Error scanning directories: %v", err)
		return err
	}
	for _, repo := range repos {
		Cache[repo.Path] = repo
	}
	SaveCache(cacheFilePath)
	return nil
}
