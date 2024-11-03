package cmd

import (
	"bytes"
	"io"
	"os"
	"testing"
	"time"

	"github.com/AccursedGalaxy/streakode/cache"
	"github.com/AccursedGalaxy/streakode/config"
	"github.com/AccursedGalaxy/streakode/scan"
	"github.com/stretchr/testify/assert"
)

func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestDisplayStats(t *testing.T) {
	// Setup test config
	config.AppConfig = config.Config{
		Author: "TestUser",
		DisplayStats: struct {
			ShowWelcomeMessage bool `mapstructure:"show_welcome_message"`
			ShowWeeklyCommits  bool `mapstructure:"show_weekly_commits"`
			ShowMonthlyCommits bool `mapstructure:"show_monthly_commits"`
			ShowTotalCommits   bool `mapstructure:"show_total_commits"`
			ShowActiveProjects bool `mapstructure:"show_active_projects"`
			ShowInsights      bool `mapstructure:"show_insights"`
			MaxProjects       int  `mapstructure:"max_projects"`
		}{
			ShowWelcomeMessage: true,
			ShowWeeklyCommits:  true,
			ShowMonthlyCommits: true,
			ShowTotalCommits:   true,
			ShowActiveProjects: true,
			ShowInsights:      true,
			MaxProjects:       5,
		},
	}

	// Setup test cache
	cache.InitCache()
	now := time.Now()
	cache.Cache = map[string]scan.RepoMetadata{
		"/test/repo1": {
			Path:           "/test/repo1",
			LastCommit:     now,
			CommitCount:    10,
			CurrentStreak:  3,
			WeeklyCommits:  5,
			MonthlyCommits: 8,
		},
		"/test/repo2": {
			Path:           "/test/repo2",
			LastCommit:     now.Add(-24 * time.Hour),
			CommitCount:    5,
			CurrentStreak:  1,
			WeeklyCommits:  2,
			MonthlyCommits: 4,
		},
	}

	output := captureOutput(DisplayStats)

	// Assertions
	assert.Contains(t, output, "TestUser's Coding Activity")
	assert.Contains(t, output, "7 commits this week")
	assert.Contains(t, output, "12 this month")
	assert.Contains(t, output, "15 total")
	assert.Contains(t, output, "repo1")
	assert.Contains(t, output, "repo2")
}