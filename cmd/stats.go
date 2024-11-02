package cmd

import (
	"fmt"
	"strings"

	"github.com/AccursedGalaxy/streakode/cache"
	"github.com/AccursedGalaxy/streakode/config"
)

// DisplayStats - Displays stats for all active repositories in a engaging and colorful userfriendly way
func DisplayStats() {
	fmt.Printf("\n🎉 Welcome back, %s! 🎉\n", config.AppConfig.Author)
	fmt.Println("=======================================")
	fmt.Println("📊 Your Coding Activity Dashboard 📊")
	fmt.Println("=======================================")

	// Calculate totals across all repos
	totalCommits := 0
	totalWeeklyCommits := 0
	totalMonthlyCommits := 0
	streakChampRepo := ""
	highestStreak := 0

	for _, repo := range cache.Cache {
		totalCommits += repo.CommitCount
		totalWeeklyCommits += repo.WeeklyCommits
		totalMonthlyCommits += repo.MonthlyCommits
		if repo.CurrentStreak > highestStreak {
			highestStreak = repo.CurrentStreak
			streakChampRepo = repo.Path[strings.LastIndex(repo.Path, "/")+1:]
		}
	}

	// Display overall statistics
	fmt.Printf("\n📈 Overall Activity\n")
	fmt.Printf("├─ Weekly Commits: %d\n", totalWeeklyCommits)
	fmt.Printf("├─ Monthly Commits: %d\n", totalMonthlyCommits)
	fmt.Printf("└─ Total Commits: %d\n", totalCommits)

	if highestStreak > 0 {
		fmt.Printf("\n🔥 Current Streak Champion: %s (%d days)\n", streakChampRepo, highestStreak)
	}

	// Display per-repository details
	fmt.Printf("\n📦 Active Repositories\n")
	for _, repo := range cache.Cache {
		repoName := repo.Path[strings.LastIndex(repo.Path, "/")+1:]
		fmt.Printf("\n%s\n", repoName)
		fmt.Printf("├─ 📅 Last: %s\n", repo.LastCommit.Format("Jan 2"))
		fmt.Printf("├─ 🎯 Week: %d | Month: %d\n", repo.WeeklyCommits, repo.MonthlyCommits)
		if repo.CurrentStreak > 0 {
			fmt.Printf("├─ 🔥 Streak: %d days\n", repo.CurrentStreak)
		}
		fmt.Printf("└─ 📊 Most Active: %s\n", repo.MostActiveDay)
	}
}
