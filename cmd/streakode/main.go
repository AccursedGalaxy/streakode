package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/accursedgalaxy/streakode/internal/git"
	"github.com/spf13/cobra"
)

func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}

func displayRepoStats(repo *git.Repository, stats *git.RepositoryStats) {
	fmt.Printf("\n=== Repository: %s ===\n", repo.Name)
	fmt.Printf("Total Commits: %d\n", stats.TotalCommits)
	fmt.Printf("Active Days: %d\n", stats.ActiveDays)
	fmt.Printf("First Commit: %s (%s ago)\n", 
		stats.FirstCommit.Timestamp.Format("2006-01-02"),
		formatDuration(time.Since(stats.FirstCommit.Timestamp)))
	fmt.Printf("Last Commit: %s (%s ago)\n",
		stats.LastCommit.Timestamp.Format("2006-01-02"),
		formatDuration(time.Since(stats.LastCommit.Timestamp)))

	// Show top 5 most active days
	type dayCount struct {
		day   string
		count int
	}
	var days []dayCount
	for day, count := range stats.CommitsByDay {
		days = append(days, dayCount{day, count})
	}
	sort.Slice(days, func(i, j int) bool {
		return days[i].count > days[j].count
	})

	fmt.Printf("\nMost Active Days:\n")
	for i := 0; i < len(days) && i < 5; i++ {
		fmt.Printf("  %s: %d commits\n", days[i].day, days[i].count)
	}
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "streakode",
		Short: "Streakode is a developer motivation and insight tool",
		Long:  `Track your coding streaks, get insights, and stay motivated with Streakode`,
	}

	var scanCmd = &cobra.Command{
		Use:   "scan [path]",
		Short: "Scan for Git repositories",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				fmt.Println("Please provide a path to scan for Git repositories.")
				return
			}

			fmt.Println("Scanning for Git repositories in:", args[0])
			scanner := git.NewScanner(args)
			repos, err := scanner.ScanForRepositories()
			if err != nil {
				log.Fatal(err)
			}

			// Print the number of repositories found
			fmt.Printf("Found %d repositories\n", len(repos))
			for _, repo := range repos {
				fmt.Printf("Repository: %s\n", repo.Name)
			}
		},
	}

	var statsCmd = &cobra.Command{
		Use:   "stats",
		Short: "Show your coding stats and streaks",
		Run: func(cmd *cobra.Command, args []string) {
			scanner := git.NewScanner(args)
			repos, err := scanner.ScanForRepositories()
			if err != nil {
				log.Fatal(err)
			}

			// Print the number of repositories found
			fmt.Printf("Found %d repositories\n", len(repos))

			for _, repo := range repos {
				stats, err := repo.AnalyzeRepository()
				if err != nil {
					fmt.Printf("Error analyzing %s: %v\n", repo.Name, err)
					continue
				}
				displayRepoStats(&repo, stats)
			}
		},
	}

	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(statsCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
