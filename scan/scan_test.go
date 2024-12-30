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

func TestDateRanges(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() DateRange
		validate func(*testing.T, DateRange)
	}{
		{
			name: "Current week range starts on Monday and ends on Sunday",
			setup: func() DateRange {
				return GetCurrentWeekRange()
			},
			validate: func(t *testing.T, dr DateRange) {
				assert.Equal(t, time.Monday, dr.Start.Weekday(), "Week should start on Monday")
				assert.Equal(t, time.Sunday, dr.End.AddDate(0, 0, -1).Weekday(), "Week should end on Sunday")
				
				// Verify time components
				assert.Equal(t, 0, dr.Start.Hour(), "Start hour should be 0")
				assert.Equal(t, 0, dr.Start.Minute(), "Start minute should be 0")
				assert.Equal(t, 0, dr.Start.Second(), "Start second should be 0")
				
				// Verify range is exactly 7 days
				diff := dr.End.Sub(dr.Start)
				assert.Equal(t, 7*24*time.Hour, diff, "Week range should be exactly 7 days")
			},
		},
		{
			name: "Previous week range is 7 days before current week",
			setup: func() DateRange {
				return GetPreviousWeekRange()
			},
			validate: func(t *testing.T, dr DateRange) {
				currentWeek := GetCurrentWeekRange()
				assert.Equal(t, currentWeek.Start, dr.End, "Previous week should end where current week starts")
				assert.Equal(t, time.Monday, dr.Start.Weekday(), "Previous week should start on Monday")
				
				// Verify range is exactly 7 days
				diff := dr.End.Sub(dr.Start)
				assert.Equal(t, 7*24*time.Hour, diff, "Week range should be exactly 7 days")
			},
		},
		{
			name: "Month range covers exact calendar month",
			setup: func() DateRange {
				return GetMonthRange(1) // Get previous month
			},
			validate: func(t *testing.T, dr DateRange) {
				// Verify start date is first day of month at 00:00:00
				assert.Equal(t, 1, dr.Start.Day(), "Month should start on day 1")
				assert.Equal(t, 0, dr.Start.Hour(), "Start hour should be 0")
				assert.Equal(t, 0, dr.Start.Minute(), "Start minute should be 0")
				assert.Equal(t, 0, dr.Start.Second(), "Start second should be 0")
				
				// Verify end date is last day of month at 23:59:59
				nextMonth := dr.Start.AddDate(0, 1, 0)
				expectedLastDay := nextMonth.AddDate(0, 0, -1)
				assert.Equal(t, expectedLastDay.Day(), dr.End.Day(), "Should end on last day of month")
				assert.Equal(t, 23, dr.End.Hour(), "End hour should be 23")
				assert.Equal(t, 59, dr.End.Minute(), "End minute should be 59")
				assert.Equal(t, 59, dr.End.Second(), "End second should be 59")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dateRange := tt.setup()
			tt.validate(t, dateRange)
		})
	}
}

func TestCountCommitsInRange(t *testing.T) {
	// Create a fixed reference date for testing
	referenceDate, _ := time.Parse("2006-01-02", "2024-01-15") // A Monday
	
	// Create test dates spanning multiple weeks
	dates := []string{
		referenceDate.Format("2006-01-02 15:04:05 -0700"),                              // Monday (current week)
		referenceDate.AddDate(0, 0, 1).Format("2006-01-02 15:04:05 -0700"),            // Tuesday (current week)
		referenceDate.AddDate(0, 0, -5).Format("2006-01-02 15:04:05 -0700"),           // Previous week (Wednesday)
		referenceDate.AddDate(0, 0, -12).Format("2006-01-02 15:04:05 -0700"),          // Two weeks ago
		referenceDate.AddDate(0, -1, 0).Format("2006-01-02 15:04:05 -0700"),           // Last month
	}

	tests := []struct {
		name     string
		dates    []string
		getRange func(time.Time) DateRange
		want     int
	}{
		{
			name:  "Current week commits",
			dates: dates,
			getRange: func(now time.Time) DateRange {
				return DateRange{
					Start: now,                              // Monday
					End:   now.AddDate(0, 0, 7),            // Next Monday
				}
			},
			want: 2, // Monday and Tuesday of current week
		},
		{
			name:  "Previous week commits",
			dates: dates,
			getRange: func(now time.Time) DateRange {
				return DateRange{
					Start: now.AddDate(0, 0, -7),           // Previous Monday
					End:   now,                             // Current Monday
				}
			},
			want: 1, // One commit from previous week
		},
		{
			name:  "Last month commits",
			dates: dates,
			getRange: func(now time.Time) DateRange {
				return DateRange{
					Start: time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location()),
					End:   time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()),
				}
			},
			want: 1, // One commit from last month
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dateRange := tt.getRange(referenceDate)
			got := countCommitsInRange(tt.dates, dateRange)
			assert.Equal(t, tt.want, got, "Expected %d commits in range, got %d\nRange: %s to %s", 
				tt.want, got, 
				dateRange.Start.Format("2006-01-02 (Mon)"),
				dateRange.End.Format("2006-01-02 (Mon)"))
		})
	}
}

func TestIsInDateRange(t *testing.T) {
	now := time.Now()
	dateRange := DateRange{
		Start: now.AddDate(0, 0, -7),
		End:   now,
	}

	tests := []struct {
		name     string
		date     time.Time
		want     bool
	}{
		{
			name: "Date within range",
			date: now.AddDate(0, 0, -3),
			want: true,
		},
		{
			name: "Date at start of range",
			date: dateRange.Start,
			want: true,
		},
		{
			name: "Date at end of range",
			date: dateRange.End,
			want: false,
		},
		{
			name: "Date before range",
			date: now.AddDate(0, 0, -8),
			want: false,
		},
		{
			name: "Date after range",
			date: now.AddDate(0, 0, 1),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsInDateRange(tt.date, dateRange)
			assert.Equal(t, tt.want, got)
		})
	}
} 