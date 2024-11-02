## Project Overview

**Streakode** is a Go-based shell plugin that motivates developers by tracking their Git activity, showing statistics, and displaying commit streaks. This project will be architected with future expansion in mind, focusing on building a strong foundation for configuration management, repository scanning, and caching. 

The goal of this dev plan is to:
1. Guide the setup of a scalable project structure in Go.
2. Outline an efficient approach to configuration, scanning, and caching.
3. Define best practices for implementing future-friendly code that is easy to extend with more complex stats and output.

---

## Project Structure

To keep the code clean, organized, and scalable, we’ll use a modular structure. Each major functionality will be its own package or module. Here’s a suggested directory layout:

```
streakode/
├── cmd/                    # Command definitions (init, stats, refresh)
│   ├── init.go             # Defines the init command
│   ├── stats.go            # Defines the stats command
│   └── refresh.go          # Defines the refresh command
├── config/                 # Configuration handling
│   ├── config.go           # Configuration loader and profile manager
├── scan/                   # Repository scanning and filtering
│   ├── scan.go             # Main scanning function with filters
├── cache/                  # Caching and local storage management
│   ├── cache.go            # Cache handling, loading, and saving
└── utils/                  # Helper functions (date parsing, etc.)
    ├── helpers.go
main.go                     # Main entry point for CLI
```

- **cmd**: Contains all CLI commands, each defined in a separate file.
- **config**: Manages configurations and profiles using the Viper library.
- **scan**: Handles repository scanning and filtering logic.
- **cache**: Manages local storage and caching of repo metadata.
- **utils**: Helper functions to keep code DRY (Don’t Repeat Yourself).

---

## Step-by-Step Development Plan

### Step 1: Configuration Management

#### Goal
Set up a configuration system that is flexible, allows for multiple profiles, and supports optional parameters.

#### Key Points
- Use **Viper** for loading and managing configurations. This makes it easy to support multiple profiles and read config values consistently across the app.
- Allow users to define custom config profiles (e.g., `work`, `personal`), which can be selected via a command-line flag.
- Define a few initial config options, including:
  - `scan_directories`: List of directories to scan.
  - `refresh_interval`: Time in hours before a repo's data is refreshed.
  - `display_stats`: Array of stats to display on shell startup.
  - `goal_settings`: User-defined commit goals.

#### Code Structure

- **config/config.go**:
  - Load the configuration and handle profile selection.
  - Create a default configuration and load values from the `.streakodeconfig` file.
  - Use Viper’s `WatchConfig()` to detect any config changes and reload dynamically.

`````go
package config

import (
    "github.com/spf13/viper"
    "log"
)

type Config struct {
    ScanDirectories []string `mapstructure:"scan_directories"`
    RefreshInterval int      `mapstructure:"refresh_interval"`
    DisplayStats    []string `mapstructure:"display_stats"`
    GoalSettings    struct {
        WeeklyCommitGoal int `mapstructure:"weekly_commit_goal"`
    } `mapstructure:"goal_settings"`
}

var AppConfig Config

func LoadConfig(profile string) {
    viper.SetConfigName(".streakodeconfig")
    viper.AddConfigPath("$HOME")
    viper.SetConfigType("yaml")
    viper.SetEnvPrefix("streakode")
    viper.AutomaticEnv()

    if profile != "" {
        viper.SetConfigName(".streakodeconfig_" + profile)
    }

    if err := viper.ReadInConfig(); err != nil {
        log.Fatalf("Error loading config file: %v", err)
    }

    if err := viper.Unmarshal(&AppConfig); err != nil {
        log.Fatalf("Unable to decode config: %v", err)
    }
}
`````

### Step 2: Repository Scanning and Filtering

#### Goal
Scan directories for Git repositories, but only include repos that the user has actively contributed to. Filter out old or dormant repositories.

#### Key Points
- Implement scanning in **scan/scan.go**. The function will:
  - Traverse specified directories to locate `.git` folders.
  - Run `git log --author=<username>` to verify user activity within each repo.
  - Track metadata like `last_commit`, `commit_count`, `last_activity_date`, and `dormant` status.
- Set a “dormant” flag for repositories with no recent activity (e.g., no commits in the last 90 days).
- Save filtered repository metadata to the cache for fast access later.

#### Code Structure

- **scan/scan.go**:
  - Define a `RepoMeta` struct for storing repo metadata.
  - Implement `ScanDirectories()` to traverse directories, find git repos, and verify user contributions.

`````go
package scan

import (
    "os"
    "path/filepath"
    "log"
)

type RepoMeta struct {
    Path          string `json:"path"`
    LastCommit    string `json:"last_commit"`
    CommitCount   int    `json:"commit_count"`
    LastActivity  string `json:"last_activity"`
    AuthorVerified bool  `json:"author_verified"`
    Dormant       bool   `json:"dormant"`
}

// Scans directories for repositories and filters by user activity
func ScanDirectories(dirs []string, author string) []RepoMeta {
    var repos []RepoMeta

    for _, dir := range dirs {
        filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
            if info.IsDir() && info.Name() == ".git" {
                meta := fetchRepoMeta(path, author)
                if meta.AuthorVerified && !meta.Dormant {
                    repos = append(repos, meta)
                }
            }
            return nil
        })
    }

    return repos
}

// Fetches metadata for a single repo, including commit count and last activity
func fetchRepoMeta(repoPath string, author string) RepoMeta {
    // Logic to fetch commit count, last commit, etc.
    return RepoMeta{}
}
`````

### Step 3: Local Storage and Caching

#### Goal
Cache repository metadata locally to avoid frequent rescanning and to support fast shell startup.

#### Key Points
- Use a **JSON cache file** for simplicity, storing minimal but expandable metadata for each repo.
- Implement a two-level cache:
  - **In-memory cache** for fast shell startup.
  - **Disk cache** for persistent storage, periodically updated with new scan results.
- Set an expiration time for cached data (e.g., 24 hours) to determine when data should be refreshed.

#### Code Structure

- **cache/cache.go**:
  - Load cache data from the JSON file on startup and populate the in-memory cache.
  - Define a function to save updated metadata back to the JSON file as needed.

`````go
package cache

import (
    "encoding/json"
    "os"
    "log"
)

var Cache map[string]RepoMeta

func LoadCache(filePath string) {
    file, err := os.Open(filePath)
    if err != nil {
        log.Printf("Error loading cache file: %v", err)
        return
    }
    defer file.Close()

    decoder := json.NewDecoder(file)
    if err := decoder.Decode(&Cache); err != nil {
        log.Printf("Error decoding cache: %v", err)
    }
}

func SaveCache(filePath string) {
    file, err := os.Create(filePath)
    if err != nil {
        log.Printf("Error creating cache file: %v", err)
        return
    }
    defer file.Close()

    encoder := json.NewEncoder(file)
    if err := encoder.Encode(Cache); err != nil {
        log.Printf("Error encoding cache: %v", err)
    }
}
`````

### Step 4: Basic Stats and Insights Display

#### Goal
Display basic stats (like last commit, commit count, and streak) without rescanning, using cached data.

#### Key Points
- Implement a CLI command (`streakode stats`) to fetch and display stats from the cache.
- Use Go’s `text/template` for formatting output, making it easy to enhance the display later.

#### Code Structure

- **cmd/stats.go**:
  - Define the `stats` command to read and display data from the cache.
  - Fetch active repos and display their stats based on user preferences.

`````go
package cmd

import (
    "fmt"
    "streakode/cache"
)

func DisplayStats() {
    for _, repo := range cache.Cache {
        fmt.Printf("Repo: %s, Last Commit: %s, Commit Count: %d\n",
            repo.Path, repo.LastCommit, repo.CommitCount)
    }
}
`````

---

## Future Considerations

### Additional Features
1. **Enhanced Stats and Gamification**: Add more detailed stats, commit streak tracking, and achievements.
2. **Periodic Refresh and Background Updates**: Implement background processes to keep data fresh without impacting performance.
3. **Improved Output and Visualization**: Add ASCII charts, progress bars, or visual indicators for achievements.


### Best Practices

To keep the code clean, maintainable, and scalable, follow these best practices throughout the project.

#### 1. **Modular Code Design**

Break the application into discrete modules based on functionality (e.g., `cmd` for CLI commands, `config` for configuration, `cache` for storage). Each module should handle only one aspect of the application, making it easy to understand and extend.

#### 2. **Separation of Concerns**

Each package or file should focus on a single responsibility:
- **cmd**: Handles command-line interface logic only.
- **config**: Manages configuration loading, saving, and profile switching.
- **scan**: Contains logic for repository scanning and filtering.
- **cache**: Manages data storage, loading, and caching.
- **utils**: Houses helper functions that can be reused across multiple modules.

#### 3. **Use Interfaces to Decouple Components**

Define interfaces for core components, such as caching and scanning. This allows for easier testing and future flexibility if you want to swap implementations.

Example:
`````go
package cache

type CacheManager interface {
    Load(filePath string) error
    Save(filePath string) error
    GetRepoMeta(repoPath string) (RepoMeta, error)
}
`````

By programming to interfaces, you can test components independently or swap them out with alternate implementations.

#### 4. **Consistent Error Handling**

Go’s error handling emphasizes explicit error checking. Always handle errors where they occur and avoid swallowing errors. Log any errors that occur so users and developers can diagnose issues.

Example:
`````go
package config

func LoadConfig(filePath string) error {
    if err := viper.ReadInConfig(); err != nil {
        log.Printf("Error loading config file %s: %v", filePath, err)
        return err
    }
    return nil
}
`````

#### 5. **Use Structs for Organized Data Management**

When working with related data, group fields in structs. This approach provides clarity and keeps related data together.

Example:
`````go
package scan

type RepoMeta struct {
    Path           string `json:"path"`
    LastCommit     string `json:"last_commit"`
    CommitCount    int    `json:"commit_count"`
    LastActivity   string `json:"last_activity"`
    AuthorVerified bool   `json:"author_verified"`
    Dormant        bool   `json:"dormant"`
}
`````

#### 6. **Keep Code DRY (Don’t Repeat Yourself)**

Avoid duplicating code by using helper functions or utility packages. Place commonly used functions, such as date formatting or file handling, in a `utils` package.

#### 7. **Readable and Self-Documenting Code**

Use descriptive variable names, clear function names, and comment where necessary. Aim for self-documenting code, where the purpose of each function and variable is clear from its name alone.

Example:
`````go
// FetchRepoMeta retrieves metadata for a single repository and verifies user activity.
func FetchRepoMeta(repoPath string, author string) RepoMeta {
    // Implementation here
}
`````

#### 8. **Testing and Debugging**

Write tests for each function, especially for complex logic such as scanning and filtering. Testing is essential for reliability, and Go has a built-in testing package that makes this straightforward. Start with unit tests and expand as needed.

Example of a basic test:
`````go
package scan

import "testing"

func TestFetchRepoMeta(t *testing.T) {
    // Set up test case
    repoPath := "/path/to/test/repo"
    author := "test-author"

    meta := FetchRepoMeta(repoPath, author)
    if !meta.AuthorVerified {
        t.Errorf("Expected author to be verified for repo %s", repoPath)
    }
}
`````

#### 9. **Keep Configuration Flexible and Extensible**

Use a configuration system that is easy to extend with new options. Load the config once at startup and reload as needed if settings change.

#### 10. **Optimize for Performance Early**

To ensure fast shell startup, consider performance during development. Use caching and background updates to minimize impact on the shell.

---

With these best practices in mind, the code will remain easy to understand, test, and extend, allowing for future feature development without significant rewrites. This approach will also help you gain deeper familiarity with Go’s idiomatic patterns and best practices.