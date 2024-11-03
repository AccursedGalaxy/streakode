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


	// TODO: Actually create more comprehensive stats and insights.
	// TODO: Implement goal tracking and achievemetns
	// TODO: Actual motivation


	// TODO: Better Output depending on config/profile && more clean lookig output.
	// -> currently the (line breaks) we have setup are displayed regardless.
	// -> need to think of a way to dynamically genearte nice compact reports regardless of what user has enabled/disabled

  // TODO: Come up with a uniform way to display emojis and stats across different metrics

	// Always show header, as it provides context
	fmt.Printf("🚀 %s's Coding Activity\n", config.AppConfig.Author)
	fmt.Printf("──────────────────────────────────────\n")
	
	// Activity overview (if any stats are enabled)
	stats := []string{}
	if config.AppConfig.DisplayStats.ShowWeeklyCommits {
		stats = append(stats, fmt.Sprintf("%d commits this week", totalWeeklyCommits))
	}
	if config.AppConfig.DisplayStats.ShowMonthlyCommits {
		stats = append(stats, fmt.Sprintf("%d this month", totalMonthlyCommits))
	}
	if config.AppConfig.DisplayStats.ShowTotalCommits {
		stats = append(stats, fmt.Sprintf("%d total", totalCommits))
	}
	if len(stats) > 0 {
		fmt.Printf("📊 %s\n", strings.Join(stats, " • "))
		fmt.Println("──────────────────────────────────────")
	}

	// Active projects section
	if config.AppConfig.DisplayStats.ShowActiveProjects {
		totalProjects := len(cache.Cache)
		maxDisplay := config.AppConfig.DisplayStats.MaxProjects
		if maxDisplay <= 0 {
			maxDisplay = 5 // default value if not set
		}
		displayCount := min(totalProjects, maxDisplay)
		
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
				summary += fmt.Sprintf(" • 🔥 %d day streak", repo.metadata.CurrentStreak)
			}

			// Add last commit info
			daysAgo := time.Since(repo.lastCommit).Hours() / 24
			if daysAgo < 1 {
				summary += " • today"
			} else if daysAgo < 2 {
				summary += " • yesterday"
			} else {
				summary += fmt.Sprintf(" • %d days ago", int(daysAgo))
			}

			fmt.Println(summary)
		}
	}

	// Insights section
	if config.AppConfig.DisplayStats.ShowInsights && highestStreak > 0 {
		fmt.Println("──────────────────────────────────────")
		fmt.Printf("💫 %s is your most active project with a %d day streak!\n",
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
