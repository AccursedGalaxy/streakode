package cmd

import (
	"testing"
	"time"

	"github.com/AccursedGalaxy/streakode/cache"
	"github.com/AccursedGalaxy/streakode/config"
	"github.com/AccursedGalaxy/streakode/scan"
	"github.com/stretchr/testify/assert"
)

func TestCalculateCommitTrend(t *testing.T) {
	calculator := &DefaultStatsCalculator{}
	
	tests := []struct {
		name     string
		current  int
		previous int
		want     CommitTrend
	}{
		{
			name:     "Increasing trend",
			current:  10,
			previous: 5,
			want:     CommitTrend{"↗️", "up 5"},
		},
		{
			name:     "Decreasing trend",
			current:  5,
			previous: 10,
			want:     CommitTrend{"↘️", "down 5"},
		},
		{
			name:     "No change",
			current:  5,
			previous: 5,
			want:     CommitTrend{"-", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculator.CalculateCommitTrend(tt.current, tt.previous)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestProcessLanguageStats(t *testing.T) {
	calculator := &DefaultStatsCalculator{}
	
	testCache := map[string]scan.RepoMetadata{
		"repo1": {
			Languages: map[string]int{
				"go":   100,
				"rust": 50,
			},
		},
		"repo2": {
			Languages: map[string]int{
				"go":     200,
				"python": 150,
			},
		},
	}

	want := map[string]int{
		"go":     300,
		"rust":   50,
		"python": 150,
	}

	got := calculator.ProcessLanguageStats(testCache)
	assert.Equal(t, want, got)
}

type MockRepoCache struct {
	repos map[string]scan.RepoMetadata
}

func (m *MockRepoCache) GetRepos() map[string]scan.RepoMetadata {
	return m.repos
}

func TestBuildProjectsSection(t *testing.T) {
	config.AppConfig = config.Config{
		DisplayStats: struct {
			ShowWelcomeMessage bool `mapstructure:"show_welcome_message"`
			ShowActiveProjects bool `mapstructure:"show_active_projects"`
			ShowInsights      bool `mapstructure:"show_insights"`
			MaxProjects       int  `mapstructure:"max_projects"`
			TableStyle struct {
				UseTableHeader 	bool 		`mapstructure:"use_table_header"`
				Style			string		`mapstructure:"style"`
				Options struct {
					DrawBorder	bool	`mapstructure:"draw_border"`
					SeparateColumns bool	`mapstructure:"separate_columns"`
					SeparateHeader bool	`mapstructure:"separate_header"`
					SeparateRows bool	`mapstructure:"separate_rows"`
				} `mapstructure:"options"`
			} `mapstructure:"table_style"`
			ActivityIndicators struct {
				HighActivity    string `mapstructure:"high_activity"`
				NormalActivity  string `mapstructure:"normal_activity"`
				NoActivity      string `mapstructure:"no_activity"`
				StreakRecord   string `mapstructure:"streak_record"`
				ActiveStreak   string `mapstructure:"active_streak"`
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
			ShowActiveProjects: true,
			MaxProjects: 10,
			TableStyle: struct {
				UseTableHeader 	bool 		`mapstructure:"use_table_header"`
				Style			string		`mapstructure:"style"`
				Options struct {
					DrawBorder	bool	`mapstructure:"draw_border"`
					SeparateColumns bool	`mapstructure:"separate_columns"`
					SeparateHeader bool	`mapstructure:"separate_header"`
					SeparateRows bool	`mapstructure:"separate_rows"`
				} `mapstructure:"options"`
			}{
				UseTableHeader: true,
				Options: struct {
					DrawBorder	bool	`mapstructure:"draw_border"`
					SeparateColumns bool	`mapstructure:"separate_columns"`
					SeparateHeader bool	`mapstructure:"separate_header"`
					SeparateRows bool	`mapstructure:"separate_rows"`
				}{
					DrawBorder: true,
					SeparateColumns: true,
				},
			},
			ActivityIndicators: struct {
				HighActivity    string `mapstructure:"high_activity"`
				NormalActivity  string `mapstructure:"normal_activity"`
				NoActivity      string `mapstructure:"no_activity"`
				StreakRecord   string `mapstructure:"streak_record"`
				ActiveStreak   string `mapstructure:"active_streak"`
			}{
				NormalActivity: "⚡",
			},
		},
	}

	mockCache := &MockRepoCache{
		repos: map[string]scan.RepoMetadata{
			"test-repo": {
				WeeklyCommits:  10,
				CurrentStreak:  5,
				LongestStreak: 7,
				LastCommit:    time.Now(), // Need this for sorting
			},
		},
	}

	cache.Cache = mockCache.GetRepos()
	output := buildProjectsSection()
	assert.Contains(t, output, "test-repo")
	assert.Contains(t, output, "10")
	assert.Contains(t, output, "5d")
} 