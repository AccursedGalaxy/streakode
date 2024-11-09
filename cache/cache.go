package cache

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/AccursedGalaxy/streakode/scan"
)

// in-memory cache to hold repo metadata during runtime
var (
	Cache map[string]scan.RepoMetadata
	metadata CacheMetadata
	mutex sync.RWMutex
)

type CacheMetadata struct {
	LastRefresh	time.Time
	Version		string		// For Future Version Tracking
}

// InitCache - Initializes the in memory cache
func InitCache() {
	Cache = make(map[string]scan.RepoMetadata)
}

// check if refresh is needed
func ShouldAutoRefresh(refreshInterval time.Duration) bool {
	mutex.RLock()
	defer mutex.RUnlock()

	if metadata.LastRefresh.IsZero() {
		return true
	}
	return time.Since(metadata.LastRefresh) > refreshInterval
}

// LoadCache - loads repository metadata from a JSON cache file into memory
func LoadCache(filePath string) error {
	mutex.Lock()
	defer mutex.Unlock()

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

	// Load metadata from a separate file
	metadataPath := filePath + ".meta"
	metaFile, err := os.Open(metadataPath)
	if os.IsNotExist(err) {
		metadata = CacheMetadata{LastRefresh: time.Time{}}
		return nil
	}
	if err != nil {
		return fmt.Errorf("error opening metadata file: %v", err)
	}
	defer metaFile.Close()

	decoder = json.NewDecoder(metaFile)
	if err := decoder.Decode(&metadata); err != nil {
		metadata = CacheMetadata{LastRefresh: time.Time{}}
		return nil
	}

	return nil
}

// SaveCache - saves the in-memory cache to a JSON file
func SaveCache(filePath string) error {
	mutex.Lock()
	defer mutex.Unlock()

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

	// Save metadata to a separate file
	metadataPath := filePath + ".meta"
	metaFile, err := os.Create(metadataPath)
	if err != nil {
		return fmt.Errorf("error creating metadata file: %v", err)
	}
	defer metaFile.Close()

	metadata.LastRefresh = time.Now()
	encoder = json.NewEncoder(metaFile)
	if err := encoder.Encode(metadata); err != nil {
		return fmt.Errorf("error encoding metadata: %v", err)
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

// AsyncRefreshCache performs a non-blocking cache refresh
func AsyncRefreshCache(dirs []string, author string, cacheFilePath string, excludedPatterns []string, excludedPaths []string) {
	go func() {
		if err := RefreshCache(dirs, author, cacheFilePath, excludedPatterns, excludedPaths); err != nil {
			log.Printf("Background cache refresh failed: %v", err)
		}
	}()
}

// QuickNeedsRefresh performs a fast check if refresh is needed without scanning repositories
func QuickNeedsRefresh(refreshInterval time.Duration) bool {
	mutex.RLock()
	defer mutex.RUnlock()

	if metadata.LastRefresh.IsZero() {
		return true
	}

	// Check if cache file exists and its modification time
	if time.Since(metadata.LastRefresh) > refreshInterval {
		return true
	}

	return false
}
