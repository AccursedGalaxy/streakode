package cmd

import (
	"fmt"

	"github.com/AccursedGalaxy/streakode/cache"
	"github.com/AccursedGalaxy/streakode/config"
)

// DisplayStats - Displays stats for all active repositories in a engaging and colorful userfriendly way
func DisplayStats() {
	fmt.Printf("Hey %s\n", config.AppConfig.Author)
	fmt.Println("================================")

	for _, repo := range cache.Cache {
		fmt.Printf("Repo: %s\n", repo.Path)
		fmt.Println("------------------")
		fmt.Println("Last Commit: ", repo.LastCommit)
		fmt.Println("Commit Count: ", repo.CommitCount)
		fmt.Println("------------------")
	}
}

