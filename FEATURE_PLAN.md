# Streakode Feature Implementation Plan 🚀

This document outlines a phased approach to implementing new features while leveraging the existing config and cache mechanisms.

## Phase 1: Enhanced Metadata Collection 📊

### Current Infrastructure Integration
We'll extend the existing `RepoMetadata` struct in `scan.go` and enhance the caching mechanism without introducing a database.

```go
// scan/scan.go
type RepoMetadata struct {
    // Existing fields...
    CommitTimes    []string          `json:"commit_times"`     // Store commit timestamps
    DailyCommits   map[string]int    `json:"daily_commits"`    // Date -> count
    HourlyCommits  map[int]int       `json:"hourly_commits"`   // Hour -> count
    FileChanges    map[string]int    `json:"file_changes"`     // Extension -> count
}
```

### Config Extension
Add new configuration options to the existing config structure:

```yaml
# .streakodeconfig.yaml
analytics:
  history_days: 90        # How many days of history to maintain
  track_file_types: true  # Track file extension statistics
  track_hours: true      # Track hourly commit patterns
```

## Phase 2: Basic Analytics 📈

### Implementation Strategy
1. Create a new `analytics` package that works with the existing cache
2. Implement analytics calculations during cache refresh
3. Store results in the existing cache structure

```go
// analytics/calculator.go
type AnalyticsResult struct {
    PeakHours      []int
    MostActiveDay  string
    FileTypeStats  map[string]int
    CommitVelocity float64
}

func CalculateFromCache(cache map[string]scan.RepoMetadata) AnalyticsResult {
    // Calculate analytics using existing cache data
}
```

## Phase 3: Achievement System 🏆

### Integration with State
Extend the existing state management to include achievements:

```go
// config/state.go
type State struct {
    ActiveProfile string                 `json:"active_profile"`
    IsValidated   bool                   `json:"is_validated"`
    Achievements  map[string]Achievement `json:"achievements"`
}

type Achievement struct {
    UnlockedAt time.Time `json:"unlocked_at"`
    Progress   float64   `json:"progress"`
}
```

## Phase 4: Command Integration 🛠️

### New Commands Structure
Build upon existing cobra command structure:

```go
// cmd/insights.go
var insightsCmd = &cobra.Command{
    Use:   "insights",
    Short: "Show detailed coding insights",
    Run: func(cmd *cobra.Command, args []string) {
        displayInsights()
    },
}

// cmd/achievements.go
var achievementsCmd = &cobra.Command{
    Use:   "achievements",
    Short: "Manage achievements",
    // Subcommands: list, progress
}
```

## Implementation Priority 📋

### 1. Core Analytics (Weeks 1-2)
- Extend RepoMetadata structure
- Enhance git data collection
- Update cache mechanism
- Add basic analytics calculations

### 2. Insights Command (Weeks 3-4)
- Implement insights command
- Add visualization helpers
- Create analytics package
- Add peak hours detection

### 3. Achievement System (Weeks 5-6)
- Extend state management
- Implement basic achievements
- Add progress tracking
- Create achievement commands

### 4. Reporting (Weeks 7-8)
- Add export functionality
- Implement markdown reports
- Create visualization helpers

## Future-Proofing Considerations 🔮

### Data Management
- Keep JSON-based storage but structure for future DB migration
- Implement data pruning for performance
- Use interfaces for storage abstraction

### Code Organization
```
streakode/
├── analytics/       # New package for analytics logic
├── achievements/    # New package for achievement system
├── cache/          # Existing cache mechanism
├── cmd/            # Extended command structure
├── config/         # Existing configuration
└── scan/           # Enhanced scanning logic
```

### Performance Optimization
1. Implement incremental cache updates
2. Add analytics result caching
3. Optimize git queries
4. Add data pruning mechanisms

### Feature Flags
Add feature flags to config for gradual rollout:
```yaml
features:
  analytics_enabled: true
  achievements_enabled: true
  reporting_enabled: true
```

## Testing Strategy 🧪

1. Unit tests for analytics calculations
2. Integration tests for cache updates
3. Mock git commands for testing
4. Performance benchmarks

## Documentation Updates 📚

1. Update README.md with new features
2. Add configuration documentation
3. Create achievement documentation
4. Add example configurations