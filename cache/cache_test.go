package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AccursedGalaxy/streakode/config"
	"github.com/AccursedGalaxy/streakode/scan"
	"github.com/stretchr/testify/assert"
)

func TestInitCache(t *testing.T) {
	InitCache()
	assert.NotNil(t, Cache)
	assert.Empty(t, Cache)
}

func TestSaveAndLoadCache(t *testing.T) {
	// Create temporary file for testing
	tmpFile, err := os.CreateTemp("", "cache-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	// Initialize cache with test data
	InitCache()
	Cache["test/repo"] = scan.RepoMetadata{
		Path:           "test/repo",
		LastCommit:     time.Now(),
		CommitCount:    10,
		CurrentStreak:  3,
		WeeklyCommits:  5,
		MonthlyCommits: 8,
	}

	// Test saving
	err = SaveCache(tmpFile.Name())
	assert.NoError(t, err)

	// Clear cache and test loading
	InitCache()
	err = LoadCache(tmpFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, 1, len(Cache))
	assert.Equal(t, "test/repo", Cache["test/repo"].Path)
	assert.Equal(t, 10, Cache["test/repo"].CommitCount)
}

func TestRefreshCache(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "cache-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	// Test refreshing cache with test directories
	dirs := []string{"./testdata"} // You'll need to create this directory with test repos
	err = RefreshCache(dirs, "test-author", tmpFile.Name(), config.AppConfig.ScanSettings.ExcludedPaths, config.AppConfig.ScanSettings.ExcludedPatterns)
	assert.NoError(t, err)
}

func TestLoadCacheWithCorruptedFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "cache-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	// Write corrupted JSON
	err = os.WriteFile(tmpFile.Name(), []byte("{invalid json}"), 0644)
	assert.NoError(t, err)

	// Test loading corrupted cache
	err = LoadCache(tmpFile.Name())
	assert.NoError(t, err) // Should not return error, but initialize empty cache
	assert.NotNil(t, Cache)
	assert.Empty(t, Cache)
}

func TestSaveCacheErrors(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "cache-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test saving to a non-existent subdirectory (should create it)
	nonExistentPath := filepath.Join(tmpDir, "subdir", "cache.json")
	err = SaveCache(nonExistentPath)
	assert.NoError(t, err)
	
	// Verify the file was created
	_, err = os.Stat(nonExistentPath)
	assert.NoError(t, err)
} 