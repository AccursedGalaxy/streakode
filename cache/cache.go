package cache

import (
	"encoding/json"
	"log"
	"os"

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
		log.Printf("Cache file not found. Creating new cache.")
		InitCache()
		return nil
	} else if err != nil {
		log.Printf("Error opening cache file: %v", err)
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&Cache); err != nil {
		log.Printf("Error decoding cache file: %v. Recreating cache.", err)
		file.Close() // Close the file before removing
		os.Remove(filePath) // Delete the corrupted cache file
		InitCache()
		return nil
	}

	return nil
}


// SaveCache - saves the in-memory cache to a JSON file
func SaveCache(filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		log.Fatalf("Error creating cache file: %v", err)
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(Cache); err != nil {
		log.Fatalf("Error encoding cache: %v", err)
		return err
	}

	return nil
}


// RefreshCache - scans directories and updates the cache with new metadata
func RefreshCache(dirs []string, author string, cacheFilePath string) error {
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
