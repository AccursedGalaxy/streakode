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
)

// setupTestRepo creates a temporary git repository with predefined commits
func setupTestRepo(t *testing.T) (string, func()) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "streakode-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Initialize git repo
	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.name", "Test User"},
		{"git", "config", "user.email", "test@example.com"},
	}

	for _, cmd := range cmds {
		command := exec.Command(cmd[0], cmd[1:]...)
		command.Dir = tmpDir // Set working directory for git commands
		if err := command.Run(); err != nil {
			os.RemoveAll(tmpDir)
			t.Fatalf("Failed to run %v: %v", cmd, err)
		}
	}

	// Create cleanup function
	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

// createTestCommit creates a commit with a specific date
func createTestCommit(t *testing.T, repoPath string, date time.Time, message string) {
	// Create a test file with unique content to force changes
	filename := filepath.Join(repoPath, fmt.Sprintf("test_%d.txt", time.Now().UnixNano()))
	if err := os.WriteFile(filename, []byte(message), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Stage and commit with specific date
	cmds := [][]string{
		{"git", "add", "."},
		{"git", "commit", "--date", date.Format(time.RFC3339), "-m", message},
	}

	for _, cmd := range cmds {
		command := exec.Command(cmd[0], cmd[1:]...)
		command.Dir = repoPath // Set working directory for git commands
		command.Env = append(os.Environ(),
			"GIT_AUTHOR_DATE="+date.Format(time.RFC3339),
			"GIT_COMMITTER_DATE="+date.Format(time.RFC3339),
		)
		if err := command.Run(); err != nil {
			t.Fatalf("Failed to run %v: %v", cmd, err)
		}
	}
}

func TestDateRangeCalculations(t *testing.T) {
	// Test current week range
	weekRange := GetCurrentWeekRange()

	// Verify week starts on Monday
	if weekRange.Start.Weekday() != time.Monday {
		t.Errorf("Week should start on Monday, got %v", weekRange.Start.Weekday())
	}

	// Verify week is 7 days
	weekDiff := weekRange.End.Sub(weekRange.Start).Hours() / 24
	if weekDiff != 7 {
		t.Errorf("Week range should be 7 days, got %.1f", weekDiff)
	}

	// Test previous week range
	prevWeek := GetPreviousWeekRange()
	if prevWeek.End != weekRange.Start {
		t.Error("Previous week should end where current week starts")
	}
}

func TestCommitCounting(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	now := time.Now().UTC()
	testCases := []struct {
		daysAgo int
		message string
	}{
		{0, "today's commit"},
		{1, "yesterday's commit"},
		{2, "two days ago commit"},
		{7, "week ago commit"},
		{14, "two weeks ago commit"},
	}

	t.Logf("Creating test commits relative to: %s", now.Format(time.RFC3339))

	// Create test commits
	for _, tc := range testCases {
		date := now.AddDate(0, 0, -tc.daysAgo)
		t.Logf("Creating commit for %d days ago: %s", tc.daysAgo, date.Format(time.RFC3339))
		createTestCommit(t, repoPath, date, tc.message)
	}

	// Test commit counting
	meta := FetchRepoMetadata(repoPath)

	// Verify commit count
	if meta.CommitCount != len(testCases) {
		t.Errorf("Expected %d commits, got %d", len(testCases), meta.CommitCount)
	}

	// Verify weekly commits (commits from 0-7 days ago, inclusive)
	expectedWeekly := 0
	for _, tc := range testCases {
		if tc.daysAgo <= 7 {
			expectedWeekly++
		}
	}
	if meta.WeeklyCommits != expectedWeekly {
		t.Errorf("Expected %d weekly commits, got %d", expectedWeekly, meta.WeeklyCommits)
		t.Logf("Weekly commits breakdown:")
		for _, tc := range testCases {
			if tc.daysAgo <= 7 {
				t.Logf("- %d days ago: %s", tc.daysAgo, tc.message)
			}
		}
	}
}

func TestStreakCalculation(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	now := time.Now().UTC()
	// Create a streak pattern: 3 days streak, 1 day gap, 2 days streak
	commits := []struct {
		daysAgo int
		message string
	}{
		{0, "today"},
		{1, "yesterday"},
		{2, "two days ago"},
		{4, "after gap"},
		{5, "before gap"},
	}

	for _, c := range commits {
		date := now.AddDate(0, 0, -c.daysAgo)
		createTestCommit(t, repoPath, date, c.message)
	}

	meta := FetchRepoMetadata(repoPath)

	// Verify current streak
	expectedStreak := 3 // today, yesterday, and two days ago
	if meta.CurrentStreak != expectedStreak {
		t.Errorf("Expected current streak of %d, got %d", expectedStreak, meta.CurrentStreak)
	}
}

func TestLanguageStats(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	// Create test files with different extensions
	files := map[string]string{
		"main.go":    "package main\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n}\n",
		"test.py":    "def test():\n    print('test')\n",
		"README.md":  "# Test Repo\n\nThis is a test\n",
		"style.css":  "body {\n    margin: 0;\n}\n",
		"index.html": "<!DOCTYPE html>\n<html>\n<body>\n</body>\n</html>\n",
	}

	for name, content := range files {
		path := filepath.Join(repoPath, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file %s: %v", name, err)
		}
	}

	// Stage and commit files
	cmds := [][]string{
		{"git", "add", "."},
		{"git", "commit", "-m", "Add test files"},
	}

	for _, cmd := range cmds {
		command := exec.Command(cmd[0], cmd[1:]...)
		command.Dir = repoPath
		if err := command.Run(); err != nil {
			t.Fatalf("Failed to run %v: %v", cmd, err)
		}
	}

	// Configure excluded extensions for the test
	config.AppConfig.LanguageSettings.ExcludedExtensions = []string{".md"}
	config.AppConfig.LanguageSettings.MinimumLines = 1

	meta := FetchRepoMetadata(repoPath)

	// Verify language statistics
	expectedExtensions := []string{".go", ".py", ".css", ".html"}
	for _, ext := range expectedExtensions {
		if _, ok := meta.Languages[ext]; !ok {
			t.Errorf("Expected to find %s in language stats", ext)
		}
	}

	// Verify excluded extensions
	if lines, ok := meta.Languages[".md"]; ok {
		t.Errorf("Markdown files should be excluded from language stats, but found %d lines", lines)
	}

	// Log all found languages for debugging
	t.Log("Found languages:")
	for ext, lines := range meta.Languages {
		t.Logf("- %s: %d lines", ext, lines)
	}
}

func TestDateParsing(t *testing.T) {
	testCases := []struct {
		input    string
		expected time.Time
		valid    bool
	}{
		{
			"2024-01-01T12:00:00+00:00|abc123|Test User|test@example.com|Test commit",
			time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			true,
		},
		{
			"invalid|abc123|Test User|test@example.com|Test commit",
			time.Time{},
			false,
		},
		{
			"2024-13-01T12:00:00+00:00|abc123|Test User|test@example.com|Test commit", // invalid month
			time.Time{},
			false,
		},
	}

	for _, tc := range testCases {
		parts := strings.Split(tc.input, "|")
		if len(parts) >= 1 {
			parsed, err := time.Parse(time.RFC3339, parts[0])
			if tc.valid {
				if err != nil {
					t.Errorf("Expected valid date parsing for %s, got error: %v", tc.input, err)
				} else if !parsed.Equal(tc.expected) {
					t.Errorf("Expected %v, got %v", tc.expected, parsed)
				}
			} else if err == nil {
				t.Errorf("Expected error parsing invalid date: %s", tc.input)
			}
		}
	}
}

func TestAuthorFiltering(t *testing.T) {
	repoPath, cleanup := setupTestRepo(t)
	defer cleanup()

	now := time.Now().UTC()
	commits := []struct {
		daysAgo int
		author  string
		email   string
		message string
	}{
		{0, "Test User", "test@example.com", "commit by test user"},
		{1, "Other User", "other@example.com", "commit by other user"},
		{2, "Test User", "test@example.com", "another test user commit"},
	}

	t.Logf("Creating test commits with different authors")

	// Create test commits with different authors
	for _, c := range commits {
		date := now.AddDate(0, 0, -c.daysAgo)

		// Create a unique file
		filename := filepath.Join(repoPath, fmt.Sprintf("test_%d.txt", time.Now().UnixNano()))
		if err := os.WriteFile(filename, []byte(c.message), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		// Stage and commit with specific author
		cmds := [][]string{
			{"git", "add", "."},
			{"git", "commit", "--author", fmt.Sprintf("%s <%s>", c.author, c.email),
				"--date", date.Format(time.RFC3339), "-m", c.message},
		}

		for _, cmd := range cmds {
			command := exec.Command(cmd[0], cmd[1:]...)
			command.Dir = repoPath
			command.Env = append(os.Environ(),
				"GIT_AUTHOR_DATE="+date.Format(time.RFC3339),
				"GIT_COMMITTER_DATE="+date.Format(time.RFC3339),
			)
			if err := command.Run(); err != nil {
				t.Fatalf("Failed to run %v: %v", cmd, err)
			}
		}
	}

	// Verify the commits were created correctly
	cmd := exec.Command("git", "log", "--format=%an <%ae>")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to get git log: %v", err)
	}
	t.Logf("Git log output:\n%s", string(output))

	// Test with author filter
	meta := fetchRepoMeta(repoPath, "Test User")

	// Should only count commits from Test User
	expectedCount := 2
	if meta.CommitCount != expectedCount {
		t.Errorf("Expected %d commits from Test User, got %d", expectedCount, meta.CommitCount)
		t.Logf("Commits found:")
		for _, commit := range meta.CommitHistory {
			t.Logf("- %s by %s: %s", commit.Date.Format(time.RFC3339), commit.Author, commit.MessageHead)
		}
	}

	// Test with different author
	meta = fetchRepoMeta(repoPath, "Other User")
	expectedCount = 1
	if meta.CommitCount != expectedCount {
		t.Errorf("Expected %d commit from Other User, got %d", expectedCount, meta.CommitCount)
		t.Logf("Commits found:")
		for _, commit := range meta.CommitHistory {
			t.Logf("- %s by %s: %s", commit.Date.Format(time.RFC3339), commit.Author, commit.MessageHead)
		}
	}
}
