package cmd

import (
	"github.com/AccursedGalaxy/streakode/scan"
)

// RepoCache represents the interface for accessing repository data
type RepoCache interface {
	GetRepos() map[string]scan.RepoMetadata
}

// StatsCalculator handles the calculation logic for repository statistics
type StatsCalculator interface {
	CalculateCommitTrend(current, previous int) CommitTrend
	ProcessLanguageStats(cache map[string]scan.RepoMetadata) map[string]int
	CalculateTableWidth() int
}

// DefaultStatsCalculator implements StatsCalculator
type DefaultStatsCalculator struct{}

// DefaultRepoCache implements RepoCache
type DefaultRepoCache struct {
	cache map[string]scan.RepoMetadata
} 