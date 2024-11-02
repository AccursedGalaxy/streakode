package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/AccursedGalaxy/streakode/cache"
	"github.com/AccursedGalaxy/streakode/config"
	"github.com/AccursedGalaxy/streakode/scan"
)

// DisplayStats - Displays stats for all active repositories in a more compact format
func DisplayStats() {

	totalWeeklyCommits := 0
	totalMonthlyCommits := 0
	totalCommits := 0
	highestStreak := 0
	streakChampRepo := ""

	for _, repo := range cache.Cache {
		totalWeeklyCommits += repo.WeeklyCommits
		totalMonthlyCommits += repo.MonthlyCommits
		totalCommits += repo.CommitCount
		if repo.CurrentStreak > highestStreak {
			highestStreak = repo.CurrentStreak
			streakChampRepo = repo.Path
		}
	}

	// Create a more dynamic and compact summary
	fmt.Printf("\n🚀 %s's Coding Activity in the Last %d days\n", config.AppConfig.Author, config.AppConfig.DormantThreshold)
	
	// Activity overview with inline stats
	fmt.Printf("Activity: %d commits this week • %d this month • %d total\n",
		totalWeeklyCommits, totalMonthlyCommits, totalCommits)

	// Active projects section
	totalProjects := len(cache.Cache)
	displayCount := min(totalProjects, 5)
	fmt.Printf("\nActive Projects (%d/%d):\n", displayCount, totalProjects)
	
	// Convert map to slice for sorting
	type repoInfo struct {
		name       string
		metadata   scan.RepoMetadata
		lastCommit time.Time
	}
	repos := make([]repoInfo, 0, len(cache.Cache))
	for path, repo := range cache.Cache {
		repoName := path[strings.LastIndex(path, "/")+1:]
		repos = append(repos, repoInfo{
			name:       repoName,
			metadata:   repo,
			lastCommit: repo.LastCommit,
		})
	}

	// Sort by most recent activity
	sort.Slice(repos, func(i, j int) bool {
		return repos[i].lastCommit.After(repos[j].lastCommit)
	})

	// Display top 5 most recently active repos
	for i := 0; i < displayCount; i++ {
		repo := repos[i]
		repoName := repo.name
		
		// Create activity indicator based on commit frequency
		activity := "⚡"
		if repo.metadata.WeeklyCommits > 5 {
			activity = "🔥"
		} else if repo.metadata.WeeklyCommits == 0 {
			activity = "💤"
		}

		// Create a compact activity summary
		summary := fmt.Sprintf("%s %s: %d↑ this week",
			activity,
			repoName[:min(len(repoName), 15)],
			repo.metadata.WeeklyCommits)

		// Add streak if exists
		if repo.metadata.CurrentStreak > 0 {
			summary += fmt.Sprintf(" • %d day streak", repo.metadata.CurrentStreak)
		}

		// Add last commit info
		daysAgo := time.Now().Sub(repo.lastCommit).Hours() / 24
		if daysAgo < 1 {
			summary += " • today"
		} else if daysAgo < 2 {
			summary += " • yesterday"
		} else {
			summary += fmt.Sprintf(" • %d days ago", int(daysAgo))
		}

		fmt.Println(summary)
	}

	// Add insights if available
	if highestStreak > 0 {
		fmt.Printf("\n💫 %s is your most active project with a %d day streak!\n",
			streakChampRepo[strings.LastIndex(streakChampRepo, "/")+1:],
			highestStreak)
	}
}

// Helper function for string length limiting
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
