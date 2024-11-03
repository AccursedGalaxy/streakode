package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		wantError bool
	}{
		{
			name: "Valid Config",
			config: Config{
				Author:           "test-author",
				DormantThreshold: 30,
				ScanDirectories: []string{"/test/dir"},
				RefreshInterval: 60,
				DisplayStats: struct {
					ShowWelcomeMessage bool `mapstructure:"show_welcome_message"`
					ShowWeeklyCommits  bool `mapstructure:"show_weekly_commits"`
					ShowMonthlyCommits bool `mapstructure:"show_monthly_commits"`
					ShowTotalCommits   bool `mapstructure:"show_total_commits"`
					ShowActiveProjects bool `mapstructure:"show_active_projects"`
					ShowInsights      bool `mapstructure:"show_insights"`
					MaxProjects       int  `mapstructure:"max_projects"`
				}{
					MaxProjects: 5,
				},
				GoalSettings: struct {
					WeeklyCommitGoal int `mapstructure:"weekly_commit_goal"`
				}{
					WeeklyCommitGoal: 10,
				},
				Colors: struct {
					HeaderColor  string `mapstructure:"header_color"`
					SectionColor string `mapstructure:"section_color"`
					DividerColor string `mapstructure:"divider_color"`
				}{
					HeaderColor:  "#FFFFFF",
					SectionColor: "#FFFFFF",
					DividerColor: "#FFFFFF",
				},
			},
			wantError: false,
		},
		{
			name: "Invalid Config - Missing Author",
			config: Config{
				DormantThreshold: 30,
				ScanDirectories: []string{"/test/dir"},
				RefreshInterval: 60,
			},
			wantError: true,
		},
		{
			name: "Invalid Config - Zero DormantThreshold",
			config: Config{
				Author:           "test-author",
				DormantThreshold: 0,
				ScanDirectories: []string{"/test/dir"},
				RefreshInterval: 60,
			},
			wantError: true,
		},
		{
			name: "Invalid Config - Empty ScanDirectories",
			config: Config{
				Author:           "test-author",
				DormantThreshold: 30,
				ScanDirectories: []string{},
				RefreshInterval: 60,
			},
			wantError: true,
		},
		{
			name: "Invalid Config - Zero RefreshInterval",
			config: Config{
				Author:           "test-author",
				DormantThreshold: 30,
				ScanDirectories: []string{"/test/dir"},
				RefreshInterval: 0,
			},
			wantError: true,
		},
		{
			name: "Invalid Config - Zero MaxProjects",
			config: Config{
				Author:           "test-author",
				DormantThreshold: 30,
				ScanDirectories: []string{"/test/dir"},
				RefreshInterval: 60,
				DisplayStats: struct {
					ShowWelcomeMessage bool `mapstructure:"show_welcome_message"`
					ShowWeeklyCommits  bool `mapstructure:"show_weekly_commits"`
					ShowMonthlyCommits bool `mapstructure:"show_monthly_commits"`
					ShowTotalCommits   bool `mapstructure:"show_total_commits"`
					ShowActiveProjects bool `mapstructure:"show_active_projects"`
					ShowInsights      bool `mapstructure:"show_insights"`
					MaxProjects       int  `mapstructure:"max_projects"`
				}{
					MaxProjects: 0,
				},
			},
			wantError: true,
		},
		// Add more test cases for other validation rules
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ValidateConfig()
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSaveAndLoadState(t *testing.T) {
	// Setup temporary home directory
	tmpHome, err := os.MkdirTemp("", "streakode-test-home")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpHome)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", oldHome)

	// Test state operations
	AppState = State{
		ActiveProfile: "test-profile",
		IsValidated:  true,
	}

	err = SaveState()
	assert.NoError(t, err)

	// Clear state and reload
	AppState = State{}
	err = LoadState()
	assert.NoError(t, err)
	assert.Equal(t, "test-profile", AppState.ActiveProfile)
	assert.True(t, AppState.IsValidated)
}

func TestLoadConfig(t *testing.T) {
	// Setup temporary home directory
	tmpHome, err := os.MkdirTemp("", "streakode-test-home")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpHome)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", oldHome)

	// Create test config file
	configContent := []byte(`
author: test-user
dormant_threshold: 30
scan_directories:
  - ~/code
refresh_interval: 60
display_stats:
  show_weekly_commits: true
  show_monthly_commits: true
  show_total_commits: true
  show_active_projects: true
  show_insights: true
  max_projects: 5
goal_settings:
  weekly_commit_goal: 10
`)
	err = os.WriteFile(filepath.Join(tmpHome, ".streakodeconfig"), configContent, 0644)
	assert.NoError(t, err)

	// Test loading default config
	LoadConfig("")
	assert.Equal(t, "test-user", AppConfig.Author)
	assert.Equal(t, 30, AppConfig.DormantThreshold)
	assert.Equal(t, 60, AppConfig.RefreshInterval)
	assert.True(t, AppConfig.DisplayStats.ShowWeeklyCommits)
	assert.Equal(t, 5, AppConfig.DisplayStats.MaxProjects)
	assert.Equal(t, 10, AppConfig.GoalSettings.WeeklyCommitGoal)

	// Test loading profile config
	profileConfig := []byte(`
author: profile-user
dormant_threshold: 45
`)
	err = os.WriteFile(filepath.Join(tmpHome, ".streakodeconfig_test"), profileConfig, 0644)
	assert.NoError(t, err)

	LoadConfig("test")
	assert.Equal(t, "profile-user", AppConfig.Author)
	assert.Equal(t, 45, AppConfig.DormantThreshold)
} 