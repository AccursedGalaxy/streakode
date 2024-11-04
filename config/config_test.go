package config

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// getTestConfig loads the valid_config.yaml and returns a Config struct
func getTestConfig(t *testing.T) Config {
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetConfigFile("testdata/valid_config.yaml")
	
	if err := v.ReadInConfig(); err != nil {
		t.Fatalf("Failed to read test config: %v", err)
	}
	
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		t.Fatalf("Failed to unmarshal test config: %v", err)
	}
	
	return config
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name         string
		modifyConfig func(*Config)
		wantError    bool
		errorMsg     string
	}{
		{
			name:         "Valid Config",
			modifyConfig: func(c *Config) {},
			wantError:    false,
		},
		{
			name: "Missing Author",
			modifyConfig: func(c *Config) {
				c.Author = ""
			},
			wantError: true,
			errorMsg:  "author must be specified",
		},
		{
			name: "Invalid DormantThreshold",
			modifyConfig: func(c *Config) {
				c.DormantThreshold = 0
			},
			wantError: true,
			errorMsg:  "dormant_threshold must be greater than 0",
		},
		// Add more test cases as needed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Load the base config from valid_config.yaml
			config := getTestConfig(t)

			// Apply test-specific modifications
			tt.modifyConfig(&config)

			// Validate
			err := config.ValidateConfig()
			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSaveAndLoadState(t *testing.T) {
	setup := SetupTestEnvironment(t)
	defer setup.Cleanup()

	// Test state operations
	AppState = State{
		ActiveProfile: "test-profile",
		IsValidated:  true,
	}

	err := SaveState()
	assert.NoError(t, err)

	// Clear state and reload
	AppState = State{}
	err = LoadState()
	assert.NoError(t, err)
	assert.Equal(t, "test-profile", AppState.ActiveProfile)
	assert.True(t, AppState.IsValidated)
}

func TestLoadConfig(t *testing.T) {
	setup := SetupTestEnvironment(t)
	defer setup.Cleanup()

	// Load test config from testdata
	configContent := setup.LoadTestConfig("valid_config.yaml")
	setup.CreateConfigFile(".streakodeconfig", configContent)

	// Test loading default config
	LoadConfig("")
	
	// Compare with expected values from valid_config.yaml
	expectedConfig := getTestConfig(t)
	assert.Equal(t, expectedConfig.Author, AppConfig.Author)
	assert.Equal(t, expectedConfig.DormantThreshold, AppConfig.DormantThreshold)
	assert.Equal(t, expectedConfig.RefreshInterval, AppConfig.RefreshInterval)
	assert.Equal(t, expectedConfig.DisplayStats.ShowWelcomeMessage, AppConfig.DisplayStats.ShowWelcomeMessage)
	assert.Equal(t, expectedConfig.DisplayStats.MaxProjects, AppConfig.DisplayStats.MaxProjects)
	assert.Equal(t, expectedConfig.GoalSettings.WeeklyCommitGoal, AppConfig.GoalSettings.WeeklyCommitGoal)

	// Test profile config
	profileConfig := []byte(`
author: profile-user
dormant_threshold: 45
`)
	setup.CreateConfigFile(".streakodeconfig_test", profileConfig)

	LoadConfig("test")
	assert.Equal(t, "profile-user", AppConfig.Author)
	assert.Equal(t, 45, AppConfig.DormantThreshold)
} 