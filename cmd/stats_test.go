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
		Author:           "TestUser",
		DormantThreshold: 30,
		RefreshInterval:  3600,
		ScanDirectories:  []string{"/test/dir"},
		DisplayStats: struct {
			ShowWelcomeMessage bool `mapstructure:"show_welcome_message"`
			ShowWeeklyCommits  bool `mapstructure:"show_weekly_commits"`
			ShowMonthlyCommits bool `mapstructure:"show_monthly_commits"`
			ShowTotalCommits   bool `mapstructure:"show_total_commits"`
			ShowActiveProjects bool `mapstructure:"show_active_projects"`
			ShowInsights      bool `mapstructure:"show_insights"`
			MaxProjects       int  `mapstructure:"max_projects"`
			TableStyle        struct {
				ShowBorder        bool   `mapstructure:"show_border"`
				ColumnSeparator   string `mapstructure:"column_separator"`
				CenterSeparator   string `mapstructure:"center_separator"`
				HeaderAlignment   string `mapstructure:"header_alignment"`
				ShowHeaderLine    bool   `mapstructure:"show_header_line"`
				ShowRowLines      bool   `mapstructure:"show_row_lines"`
				MinColumnWidths   struct {
					Repository int `mapstructure:"repository"`
					Weekly    int `mapstructure:"weekly"`
					Streak    int `mapstructure:"streak"`
					Changes   int `mapstructure:"changes"`
					Activity  int `mapstructure:"activity"`
				} `mapstructure:"min_column_widths"`
			} `mapstructure:"table_style"`
			ActivityIndicators struct {
				HighActivity    string `mapstructure:"high_activity"`
				NormalActivity  string `mapstructure:"normal_activity"`
				NoActivity      string `mapstructure:"no_activity"`
				StreakRecord    string `mapstructure:"streak_record"`
				ActiveStreak    string `mapstructure:"active_streak"`
			} `mapstructure:"activity_indicators"`
			Thresholds struct {
				HighActivity int `mapstructure:"high_activity"`
			} `mapstructure:"thresholds"`
			InsightSettings struct {
				TopLanguagesCount int  `mapstructure:"top_languages_count"`
				ShowDailyAverage  bool `mapstructure:"show_daily_average"`
				ShowTopLanguages  bool `mapstructure:"show_top_languages"`
				ShowPeakCoding    bool `mapstructure:"show_peak_coding"`
				ShowWeeklySummary bool `mapstructure:"show_weekly_summary"`
				ShowWeeklyGoal    bool `mapstructure:"show_weekly_goal"`
				ShowMostActive    bool `mapstructure:"show_most_active"`
			} `mapstructure:"insight_settings"`
		}{
			ShowWelcomeMessage: true,
			ShowWeeklyCommits:  true,
			ShowMonthlyCommits: true,
			ShowTotalCommits:   false,
			ShowActiveProjects: true,
			ShowInsights:      true,
			MaxProjects:       5,
			TableStyle: struct {
				ShowBorder        bool   `mapstructure:"show_border"`
				ColumnSeparator   string `mapstructure:"column_separator"`
				CenterSeparator   string `mapstructure:"center_separator"`
				HeaderAlignment   string `mapstructure:"header_alignment"`
				ShowHeaderLine    bool   `mapstructure:"show_header_line"`
				ShowRowLines      bool   `mapstructure:"show_row_lines"`
				MinColumnWidths   struct {
					Repository int `mapstructure:"repository"`
					Weekly    int `mapstructure:"weekly"`
					Streak    int `mapstructure:"streak"`
					Changes   int `mapstructure:"changes"`
					Activity  int `mapstructure:"activity"`
				} `mapstructure:"min_column_widths"`
			}{
				ShowBorder:      true,
				ColumnSeparator: "|",
				CenterSeparator: "+",
				HeaderAlignment: "center",
				ShowHeaderLine:  true,
				ShowRowLines:    false,
				MinColumnWidths: struct {
					Repository int `mapstructure:"repository"`
					Weekly    int `mapstructure:"weekly"`
					Streak    int `mapstructure:"streak"`
					Changes   int `mapstructure:"changes"`
					Activity  int `mapstructure:"activity"`
				}{
					Repository: 20,
					Weekly:     8,
					Streak:     8,
					Changes:    13,
					Activity:   10,
				},
			},
			ActivityIndicators: struct {
				HighActivity    string `mapstructure:"high_activity"`
				NormalActivity  string `mapstructure:"normal_activity"`
				NoActivity      string `mapstructure:"no_activity"`
				StreakRecord    string `mapstructure:"streak_record"`
				ActiveStreak    string `mapstructure:"active_streak"`
			}{
				HighActivity:   "🔥",
				NormalActivity: "⚡",
				NoActivity:     "💤",
				StreakRecord:   "🏆",
				ActiveStreak:   "🔥",
			},
			Thresholds: struct {
				HighActivity int `mapstructure:"high_activity"`
			}{
				HighActivity: 10,
			},
			InsightSettings: struct {
				TopLanguagesCount int  `mapstructure:"top_languages_count"`
				ShowDailyAverage  bool `mapstructure:"show_daily_average"`
				ShowTopLanguages  bool `mapstructure:"show_top_languages"`
				ShowPeakCoding    bool `mapstructure:"show_peak_coding"`
				ShowWeeklySummary bool `mapstructure:"show_weekly_summary"`
				ShowWeeklyGoal    bool `mapstructure:"show_weekly_goal"`
				ShowMostActive    bool `mapstructure:"show_most_active"`
			}{
				TopLanguagesCount: 3,
				ShowDailyAverage:  true,
				ShowTopLanguages:  true,
				ShowPeakCoding:    true,
				ShowWeeklySummary: true,
				ShowWeeklyGoal:    true,
				ShowMostActive:    true,
			},
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

	// Updated assertions to match new format
	assert.Contains(t, output, "🚀 TestUser's Coding Activity")
	assert.Contains(t, output, "7 commits this week")
	assert.Contains(t, output, "12 this month")
	assert.Contains(t, output, "⚡ repo1")
	assert.Contains(t, output, "⚡ repo2")
	assert.Contains(t, output, "5↑")    // Weekly commits for repo1
	assert.Contains(t, output, "2↑")    // Weekly commits for repo2
	assert.Contains(t, output, "3d🔥")  // Streak for repo1
	assert.Contains(t, output, "1d🔥")  // Streak for repo2
	assert.Contains(t, output, "today") // Activity for repo1
	assert.Contains(t, output, "1d ago") // Activity for repo2
	assert.Contains(t, output, "🌟 Most active: repo1") // Insights section
}