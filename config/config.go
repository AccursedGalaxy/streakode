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
	ScanSettings struct {
		ExcludedPatterns []string `mapstructure:"excluded_patterns"` // e.g., ["node_modules", "dist", ".git"]
		ExcludedPaths    []string `mapstructure:"excluded_paths"`    // Full paths to exclude
	} `mapstructure:"scan_settings"`
	RefreshInterval int      `mapstructure:"refresh_interval"`
	DisplayStats    struct {
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
	} `mapstructure:"display_stats"`
	GoalSettings    struct {
		WeeklyCommitGoal int `mapstructure:"weekly_commit_goal"`
	} `mapstructure:"goal_settings"`
	Colors struct {
		HeaderColor  string `mapstructure:"header_color"`
	}
	DetailedStats bool `mapstructure:"detailed_stats"`
	Debug         bool `mapstructure:"debug"`
	LanguageSettings struct {
		ExcludedExtensions []string `mapstructure:"excluded_extensions"` // e.g., [".yaml", ".txt", ".md"]
		ExcludedLanguages  []string `mapstructure:"excluded_languages"`  // e.g., ["YAML", "Text", "Markdown"]
		MinimumLines       int      `mapstructure:"minimum_lines"`       // Minimum lines for a language to be included
		ShowDividers       bool     `mapstructure:"show_dividers"`       // Display dividers between languages in output

		LanguageDisplay struct {
			GoDisplay     string `mapstructure:"go_display"`        // Display name/icon for Go (e.g., "🔵 Go")
			PythonDisplay string `mapstructure:"python_display"`    // Display name/icon for Python
			LuaDisplay    string `mapstructure:"lua_display"`       // Display name/icon for Lua
			JavaScriptDisplay string `mapstructure:"javascript_display"` // Display name/icon for JavaScript
			TypeScriptDisplay string `mapstructure:"typescript_display"` // Display name/icon for TypeScript
			RustDisplay    string `mapstructure:"rust_display"`     // Display name/icon for Rust
			CppDisplay     string `mapstructure:"cpp_display"`      // Display name/icon for C++
			CDisplay       string `mapstructure:"c_display"`        // Display name/icon for C
			JavaDisplay    string `mapstructure:"java_display"`     // Display name/icon for Java
			RubyDisplay    string `mapstructure:"ruby_display"`     // Display name/icon for Ruby
			PHPDisplay     string `mapstructure:"php_display"`      // Display name/icon for PHP
			HTMLDisplay    string `mapstructure:"html_display"`     // Display name/icon for HTML
			CSSDisplay     string `mapstructure:"css_display"`      // Display name/icon for CSS
			ShellDisplay   string `mapstructure:"shell_display"`    // Display name/icon for Shell
			DefaultDisplay string `mapstructure:"default_display"`  // Display for any unspecified language
		} `mapstructure:"language_display"`
	} `mapstructure:"language_settings"`
	ShowDividers bool `mapstructure:"show_dividers"`
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

	// Set default activity indicators if not specified
	if c.DisplayStats.ActivityIndicators.HighActivity == "" {
		c.DisplayStats.ActivityIndicators.HighActivity = "🔥"
	}
	if c.DisplayStats.ActivityIndicators.NormalActivity == "" {
		c.DisplayStats.ActivityIndicators.NormalActivity = "⚡"
	}
	if c.DisplayStats.ActivityIndicators.NoActivity == "" {
		c.DisplayStats.ActivityIndicators.NoActivity = "💤"
	}
	if c.DisplayStats.ActivityIndicators.StreakRecord == "" {
		c.DisplayStats.ActivityIndicators.StreakRecord = "🏆"
	}
	if c.DisplayStats.ActivityIndicators.ActiveStreak == "" {
		c.DisplayStats.ActivityIndicators.ActiveStreak = "🔥"
	}

	// Validate thresholds
	if c.DisplayStats.Thresholds.HighActivity <= 0 {
		c.DisplayStats.Thresholds.HighActivity = 10
	}

	// Validate insight settings
	if c.DisplayStats.InsightSettings.TopLanguagesCount <= 0 {
		c.DisplayStats.InsightSettings.TopLanguagesCount = 3
	}

	// Validate language settings
	if c.LanguageSettings.MinimumLines < 0 {
		c.LanguageSettings.MinimumLines = 0
	}

	// Normalize excluded extensions
	for i, ext := range c.LanguageSettings.ExcludedExtensions {
		if !strings.HasPrefix(ext, ".") {
			c.LanguageSettings.ExcludedExtensions[i] = "." + ext
		}
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

// InitConfig reads in config file and ENV variables if set.
func InitConfig(cfgFile string) {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".streakodeconfig" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".streakodeconfig")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		if AppConfig.Debug {
			fmt.Println("Using config file:", viper.ConfigFileUsed())
		}
	}

	if err := viper.Unmarshal(&AppConfig); err != nil {
		fmt.Println("Error parsing config:", err)
		os.Exit(1)
	}

	// Set default values after unmarshal
	setDefaults()
}

// setDefaults sets default values for configuration options
func setDefaults() {
	// Set default dormant threshold if not specified
	if AppConfig.DormantThreshold <= 0 {
		AppConfig.DormantThreshold = 30 // 30 days default
	}

	// Set default refresh interval if not specified
	if AppConfig.RefreshInterval <= 0 {
		AppConfig.RefreshInterval = 60 // 60 minutes default
	}

	// Set default activity indicators if not specified
	if AppConfig.DisplayStats.ActivityIndicators.HighActivity == "" {
		AppConfig.DisplayStats.ActivityIndicators.HighActivity = "🔥"
	}
	if AppConfig.DisplayStats.ActivityIndicators.NormalActivity == "" {
		AppConfig.DisplayStats.ActivityIndicators.NormalActivity = "⚡"
	}
	if AppConfig.DisplayStats.ActivityIndicators.NoActivity == "" {
		AppConfig.DisplayStats.ActivityIndicators.NoActivity = "💤"
	}
	if AppConfig.DisplayStats.ActivityIndicators.StreakRecord == "" {
		AppConfig.DisplayStats.ActivityIndicators.StreakRecord = "🏆"
	}
	if AppConfig.DisplayStats.ActivityIndicators.ActiveStreak == "" {
		AppConfig.DisplayStats.ActivityIndicators.ActiveStreak = "🔥"
	}

	// Set default thresholds
	if AppConfig.DisplayStats.Thresholds.HighActivity <= 0 {
		AppConfig.DisplayStats.Thresholds.HighActivity = 10
	}

	// Set default insight settings
	if AppConfig.DisplayStats.InsightSettings.TopLanguagesCount <= 0 {
		AppConfig.DisplayStats.InsightSettings.TopLanguagesCount = 3
	}

	// Set default language settings
	if AppConfig.LanguageSettings.MinimumLines < 0 {
		AppConfig.LanguageSettings.MinimumLines = 0
	}

	// Set default header color if not specified
	if AppConfig.Colors.HeaderColor == "" {
		AppConfig.Colors.HeaderColor = "#FF69B4"
	}

	// Set default max projects if not specified
	if AppConfig.DisplayStats.MaxProjects <= 0 {
		AppConfig.DisplayStats.MaxProjects = 10
	}
}
