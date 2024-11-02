package main

import (
	"fmt"
	"log"

	"github.com/AccursedGalaxy/streakode/config"
	"github.com/AccursedGalaxy/streakode/scan"
)

func main() {
	config.LoadConfig("")
	fmt.Printf("Config loaded: %+v\n", config.AppConfig)

	author := config.AppConfig.Author
	fmt.Printf("Searching for repositories with author: %s\n", author)

	repos, err := scan.ScanDirectories(config.AppConfig.ScanDirectories, author)
	if err != nil {
		log.Fatalf("Error scanning directories: %v", err)
	}

	if len(repos) == 0 {
		fmt.Println("No active repositories found!")
	}

	for _, repo := range repos {
		fmt.Printf("Repo: %s, Last Commit: %s, Commit Count: %d, Dormant: %v\n", 
			repo.Path, repo.LastCommit, repo.CommitCount, repo.Dormant)
	}
}
