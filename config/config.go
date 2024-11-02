package config

import (
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
	DisplayStats    []string `mapstructure:"display_stats"`
	GoalSettings    struct {
		WeeklyCommitGoal int `mapstructure:"weekly_commit_goal"`
	} `mapstructure:"goal_settings"`
}

var AppConfig Config

// LoadConfig initializes the config with optional profile selection
func LoadConfig(profile string) {
	viper.SetConfigName(".streakodeconfig")
	viper.AddConfigPath("$HOME")
	viper.SetConfigType("yaml")
	viper.SetEnvPrefix("streakode")
	// override config with env vars if set
	viper.AutomaticEnv()

	// Set Profile if provided
	if profile != "" {
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
