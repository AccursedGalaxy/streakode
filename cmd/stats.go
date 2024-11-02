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

	totalCommits := 0
	for _, repo := range cache.Cache {
		totalCommits += repo.CommitCount
	}

	fmt.Printf("\n🌟 Total Commits Across All Repos: %d 🌟\n\n", totalCommits)

	for _, repo := range cache.Cache {
		// Extract repository name from path
		repoName := repo.Path[strings.LastIndex(repo.Path, "/")+1:]
		
		fmt.Println("📂 Repository:", repoName)
		fmt.Println("   " + strings.Repeat("─", 40))
		fmt.Printf("   📅 Last Commit: %s\n", repo.LastCommit.Format("Mon Jan 2 15:04:05 2006"))
		fmt.Printf("   ⚡ Commit Count: %d\n", repo.CommitCount)
		
		fmt.Println()
	}
}
