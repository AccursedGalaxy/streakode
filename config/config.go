package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Author          string   `mapstructure:"author"`
	DormantThreshold int      `mapstructure:"dormant_threshold"`
	ScanDirectories []string `mapstructure:"scan_directories"`
	RefreshInterval int      `mapstructure:"refresh_interval"`
	DisplayStats    struct {
		ShowWelcomeMessage bool `mapstructure:"show_welcome_message"`
		ShowWeeklyCommits  bool `mapstructure:"show_weekly_commits"`
		ShowMonthlyCommits bool `mapstructure:"show_monthly_commits"`
		ShowTotalCommits   bool `mapstructure:"show_total_commits"`
		ShowActiveProjects bool `mapstructure:"show_active_projects"`
		ShowInsights      bool `mapstructure:"show_insights"`
		MaxProjects       int  `mapstructure:"max_projects"`
	} `mapstructure:"display_stats"`
	GoalSettings    struct {
		WeeklyCommitGoal int `mapstructure:"weekly_commit_goal"`
	} `mapstructure:"goal_settings"`
	Colors struct {
		HeaderColor  string `mapstructure:"header_color"`
		SectionColor string `mapstructure:"section_color"`
		DividerColor string `mapstructure:"divider_color"`
	}
}

type State struct {
	ActiveProfile string `json:"active_profile"`
	IsValidated	  bool   `json:"is_validated"`
}

var (
	AppConfig Config
	AppState  State
)

// Validate Config
func (c *Config) ValidateConfig() error {
	if c.Author == "" {
		return fmt.Errorf("author must be specified")
	}
	if c.DormantThreshold <= 0 {
		return fmt.Errorf("dormant_threshold must be greater than 0")
	}
	if len(c.ScanDirectories) == 0 {
		return fmt.Errorf("at least one scan directory must be specified")
	}
	if c.RefreshInterval <= 0 {
		return fmt.Errorf("refresh_interval must be greater than 0")
	}
	if c.DisplayStats.MaxProjects <= 0 {
		return fmt.Errorf("display_stats.max_projects must be greater than 0")
	}
	if c.GoalSettings.WeeklyCommitGoal < 0 {
		return fmt.Errorf("goal_settings.weekly_commit_goal cannot be negative")
	}

	// Validate colors (optional - can remove this to allow empty colors)
	if c.Colors.HeaderColor == "" {
		c.Colors.HeaderColor = "#FF69B4"
	}
	if c.Colors.SectionColor == "" {
		c.Colors.SectionColor = "#87CEEB"
	}
	if c.Colors.DividerColor == "" {
		c.Colors.DividerColor = "#808080"
	}
	return nil
}

func SaveState() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	
	stateFile := filepath.Join(home, ".streakode.state")
	data, err := json.Marshal(AppState)
	if err != nil {
		return err
	}
	
	return os.WriteFile(stateFile, data, 0644)
}

func LoadState() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	
	stateFile := filepath.Join(home, ".streakode.state")
	data, err := os.ReadFile(stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			AppState = State{} // Initialize empty state
			return nil
		}
		return err
	}
	
	return json.Unmarshal(data, &AppState)
}

// LoadConfig initializes the config with optional profile selection
func LoadConfig(profile string) {
	// Reset Viper's configuration
	viper.Reset()

	// Set up basic Viper configuration
	viper.AddConfigPath("$HOME")
	viper.SetConfigType("yaml")
	viper.SetEnvPrefix("streakode")
	viper.AutomaticEnv()

	// Determine which config file to load
	configName := ".streakodeconfig"
	if profile != "" && profile != "default" && profile != "-" {
		configName = ".streakodeconfig_" + profile
	}
	viper.SetConfigName(configName)

	// Try to read the config file first
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file '%s': %v", configName, err)
	}

	// Only after successful config load, we handle the state
	if err := LoadState(); err != nil {
		log.Printf("Warning: Could not load state: %v", err)
	}

	// Update state with new profile
	if profile != AppState.ActiveProfile {
		AppState.ActiveProfile = profile
		if err := SaveState(); err != nil {
			log.Printf("Warning: Could not save state: %v", err)
		}
	}

	// Unmarshal the config into the AppConfig struct
	if err := viper.Unmarshal(&AppConfig); err != nil {
		log.Fatalf("Unable to decode the config into struct: %v", err)
	}

	// Validate config only if not already validated
	if !AppState.IsValidated {
		if err := AppConfig.ValidateConfig(); err != nil {
			log.Fatalf("Config validation failed: %v", err)
		}
		AppState.IsValidated = true
		if err := SaveState(); err != nil {
			log.Fatalf("Could not save validation state: %v", err)
		}
	}

	// Expand home directory in scan directories
	for i, dir := range AppConfig.ScanDirectories {
		if strings.HasPrefix(dir, "~/") {
			home, err := os.UserHomeDir()
			if err != nil {
				log.Fatalf("Error getting home directory: %v", err)
			}
			AppConfig.ScanDirectories[i] = filepath.Join(home, dir[2:])
		}
	}
}
