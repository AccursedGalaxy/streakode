package cache

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/AccursedGalaxy/streakode/config"
	"github.com/AccursedGalaxy/streakode/scan"
)

// Global cache manager instance
var (
	manager *CacheManager
	mutex   sync.RWMutex
)

// InitCache - Initializes the cache manager
func InitCache() {
	mutex.Lock()
	defer mutex.Unlock()

	if manager != nil {
		return
	}

	manager = NewCacheManager(getCacheFilePath())
	if err := manager.Load(); err != nil {
		log.Printf("Error loading cache: %v\n", err)
	}
}

// LoadCache - loads repository metadata from cache file
func LoadCache(filePath string) error {
	mutex.Lock()
	defer mutex.Unlock()

	if manager == nil {
		manager = NewCacheManager(filePath)
	}

	return manager.Load()
}

// SaveCache - saves the cache to disk
func SaveCache(filePath string) error {
	mutex.Lock()
	defer mutex.Unlock()

	if manager == nil {
		return fmt.Errorf("cache manager not initialized")
	}

	return manager.Save()
}

// RefreshCache - updates the cache with fresh data
func RefreshCache(dirs []string, author string, cacheFilePath string, excludedPatterns []string, excludedPaths []string) error {
	mutex.Lock()
	defer mutex.Unlock()

	if manager == nil {
		manager = NewCacheManager(cacheFilePath)
	}

	// Create exclusion function
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

	// Scan directories for repositories
	repos, err := scan.ScanDirectories(dirs, author, shouldExclude)
	if err != nil {
		return fmt.Errorf("error scanning directories: %v", err)
	}

	// Convert repos slice to map
	reposMap := make(map[string]scan.RepoMetadata)
	for _, repo := range repos {
		reposMap[repo.Path] = repo
	}

	// Update cache with new data using the manager's method
	manager.updateCacheData(reposMap)

	return manager.Save()
}

// AsyncRefreshCache performs a non-blocking cache refresh
func AsyncRefreshCache(dirs []string, author string, cacheFilePath string, excludedPatterns []string, excludedPaths []string) {
	go func() {
		if err := RefreshCache(dirs, author, cacheFilePath, excludedPatterns, excludedPaths); err != nil {
			log.Printf("Background cache refresh failed: %v", err)
		}
	}()
}

// QuickNeedsRefresh performs a fast check if refresh is needed
func QuickNeedsRefresh(refreshInterval time.Duration) bool {
	mutex.RLock()
	defer mutex.RUnlock()

	if manager == nil || manager.cache == nil {
		return true
	}

	return time.Since(manager.cache.LastSync) > refreshInterval
}

// CleanCache removes the cache file and resets the in-memory cache
func CleanCache(cacheFilePath string) error {
	mutex.Lock()
	defer mutex.Unlock()

	if manager != nil {
		manager.cache = newCommitCache()
	}

	// Remove cache file if present
	if err := os.Remove(cacheFilePath); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("error removing cache file: %v", err)
		}
	}

	// Remove metadata file if present
	metaFile := cacheFilePath + ".meta"
	if err := os.Remove(metaFile); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("error removing metadata file: %v", err)
		}
	}

	return nil
}

// Helper function to get cache file path
func getCacheFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	if config.AppState.ActiveProfile == "" {
		return filepath.Join(home, ".streakode.cache")
	}
	return filepath.Join(home, fmt.Sprintf(".streakode_%s.cache", config.AppState.ActiveProfile))
}

// Cache is now a proxy to the manager's cache
var Cache = &cacheProxy{}

type cacheProxy struct{}

func (cp *cacheProxy) Get(key string) (scan.RepoMetadata, bool) {
	mutex.RLock()
	defer mutex.RUnlock()

	if manager == nil || manager.cache == nil {
		return scan.RepoMetadata{}, false
	}

	repo, exists := manager.cache.Repositories[key]
	return repo, exists
}

func (cp *cacheProxy) GetDisplayStats() *DisplayStats {
	mutex.RLock()
	defer mutex.RUnlock()

	if manager == nil || manager.cache == nil {
		return nil
	}

	return &manager.cache.DisplayStats
}

func (cp *cacheProxy) Set(key string, value scan.RepoMetadata) {
	mutex.Lock()
	defer mutex.Unlock()

	if manager == nil || manager.cache == nil {
		return
	}

	manager.cache.Repositories[key] = value
}

func (cp *cacheProxy) Delete(key string) {
	mutex.Lock()
	defer mutex.Unlock()

	if manager == nil || manager.cache == nil {
		return
	}

	delete(manager.cache.Repositories, key)
}

func (cp *cacheProxy) Range(f func(key string, value scan.RepoMetadata) bool) {
	mutex.RLock()
	defer mutex.RUnlock()

	if manager == nil || manager.cache == nil {
		return
	}

	for k, v := range manager.cache.Repositories {
		if !f(k, v) {
			break
		}
	}
}

func (cp *cacheProxy) Len() int {
	mutex.RLock()
	defer mutex.RUnlock()

	if manager == nil || manager.cache == nil {
		return 0
	}

	return len(manager.cache.Repositories)
}
