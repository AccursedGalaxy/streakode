package config

import (
	"encoding/json"
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
}

type State struct {
	ActiveProfile string `json:"active_profile"`
}

var (
	AppConfig Config
	AppState  State
)

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
	// Load the state first
	if err := LoadState(); err != nil {
		log.Printf("Warning: Could not load state: %v", err)
	}

	// If no profile is provided, use the one from state
	if profile == "" {
		profile = AppState.ActiveProfile
	} else {
		// Update state with new profile
		AppState.ActiveProfile = profile
		if err := SaveState(); err != nil {
			log.Printf("Warning: Could not save state: %v", err)
		}
	}

	// Always start with default config name
	viper.SetConfigName(".streakodeconfig")
	viper.AddConfigPath("$HOME")
	viper.SetConfigType("yaml")
	viper.SetEnvPrefix("streakode")
	viper.AutomaticEnv()

	// Only append profile suffix if it's not empty and not "default"
	if profile != "" && profile != "default" && profile != "-" {
		viper.SetConfigName(".streakodeconfig_" + profile)
	}

	// Read in Config and Check for Errors
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	// Unmarshal the config into the AppConfig struct
	if err := viper.Unmarshal(&AppConfig); err != nil {
		log.Fatalf("Unable to decode the config into struct: %v", err)
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
