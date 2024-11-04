package scan

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/AccursedGalaxy/streakode/config"
	"github.com/stretchr/testify/assert"
)

func setupGitRepo(t *testing.T, path, author string) error {
	// Initialize git repo
	cmds := []struct {
		name string
		args []string
		env  []string
	}{
		{"git", []string{"init", "--initial-branch=main"}, nil},
		{"git", []string{"config", "user.name", author}, nil},
		{"git", []string{"config", "user.email", author + "@test.com"}, nil},
		{"git", []string{"config", "commit.gpgsign", "false"}, nil},
		// Create and commit a file
		{"sh", []string{"-c", "echo 'test' > README.md"}, nil},
		{"git", []string{"add", "README.md"}, nil},
		{"git", []string{"commit", "-m", "Initial commit"}, []string{
			"GIT_AUTHOR_NAME=" + author,
			"GIT_AUTHOR_EMAIL=" + author + "@test.com",
			"GIT_COMMITTER_NAME=" + author,
			"GIT_COMMITTER_EMAIL=" + author + "@test.com",
		}},
	}

	for _, cmd := range cmds {
		command := exec.Command(cmd.name, cmd.args...)
		command.Dir = path
		if cmd.env != nil {
			command.Env = append(os.Environ(), cmd.env...)
		}
		if output, err := command.CombinedOutput(); err != nil {
			t.Logf("Command failed: %s %v", cmd.name, cmd.args)
			t.Logf("Output: %s", string(output))
			return fmt.Errorf("command failed: %v", err)
		}
	}
	return nil
}

func TestScanDirectories(t *testing.T) {
	// Create test directories
	baseDir, err := os.MkdirTemp("", "streakode-scan-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(baseDir)

	// Set up test config to prevent repos from being marked as dormant
	config.AppConfig.DormantThreshold = 365 // Set a large threshold

	// Create multiple test repositories
	repos := []string{"repo1", "repo2", "repo3"}
	for _, repo := range repos {
		repoPath := filepath.Join(baseDir, repo)
		err := os.MkdirAll(repoPath, 0755)
		assert.NoError(t, err)
		
		// Initialize git repo with a commit
		err = setupGitRepo(t, repoPath, "test-author")
		assert.NoError(t, err)

		// Verify the repository was created correctly
		_, err = os.Stat(filepath.Join(repoPath, ".git"))
		assert.NoError(t, err, "Git directory not created in %s", repoPath)
	}
	// Test scanning
	results, err := ScanDirectories([]string{baseDir}, "test-author", func(path string) bool {
		for _, excluded := range config.AppConfig.ScanSettings.ExcludedPaths {
			if strings.HasPrefix(path, excluded) {
				return true
			}
		}
		for _, pattern := range config.AppConfig.ScanSettings.ExcludedPatterns {
			if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
				return true
			}
		}
		return false
	})
	if err != nil {
		t.Fatalf("ScanDirectories failed: %v", err)
	}

	// Debug output
	if len(results) != len(repos) {
		t.Logf("Expected %d repos, got %d", len(repos), len(results))
		t.Logf("Base directory: %s", baseDir)
		t.Logf("Found repositories:")
		for _, result := range results {
			t.Logf("  - %s (verified: %v, dormant: %v)", result.Path, result.AuthorVerified, result.Dormant)
		}
	}

	assert.Equal(t, len(repos), len(results), "Number of repositories found doesn't match")
	
	// Verify each repository was found and has correct metadata
	for _, result := range results {
		assert.True(t, result.AuthorVerified, "Repository not verified: %s", result.Path)
		assert.Equal(t, 1, result.CommitCount, "Incorrect commit count for %s", result.Path)
		assert.False(t, result.Dormant, "Repository marked as dormant: %s", result.Path)
	}
}

func TestCalculateStreak(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name     string
		dates    []string
		expected StreakInfo
	}{
		{
			name: "Continuous streak",
			dates: []string{
				now.Format("2006-01-02 15:04:05 -0700"),
				now.Add(-24 * time.Hour).Format("2006-01-02 15:04:05 -0700"),
				now.Add(-48 * time.Hour).Format("2006-01-02 15:04:05 -0700"),
			},
			expected: StreakInfo{Current: 3, Longest: 3},
		},
		{
			name: "Broken streak",
			dates: []string{
				now.Format("2006-01-02 15:04:05 -0700"),
				now.Add(-48 * time.Hour).Format("2006-01-02 15:04:05 -0700"),
			},
			expected: StreakInfo{Current: 1, Longest: 1},
		},
		{
			name: "Multiple commits same day",
			dates: []string{
				now.Format("2006-01-02 15:04:05 -0700"),
					now.Format("2006-01-02 15:04:05 -0700"),
					now.Add(-24 * time.Hour).Format("2006-01-02 15:04:05 -0700"),
			},
			expected: StreakInfo{Current: 2, Longest: 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			streak := calculateStreakInfo(tt.dates)
			assert.Equal(t, tt.expected, streak)
		})
	}
}

func TestCountRecentCommits(t *testing.T) {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	twoDaysAgo := now.Add(-48 * time.Hour)
	
	dates := []string{
		now.Format("2006-01-02 15:04:05 -0700"),
		yesterday.Format("2006-01-02 15:04:05 -0700"),
		twoDaysAgo.Format("2006-01-02 15:04:05 -0700"),
	}

	tests := []struct {
		name     string
		days     int
		expected int
	}{
		{"Weekly commits", 7, 3},
		{"Monthly commits", 30, 3},
		{"Daily commits", 1, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := countRecentCommits(dates, tt.days)
			assert.Equal(t, tt.expected, count)
		})
	}
} 