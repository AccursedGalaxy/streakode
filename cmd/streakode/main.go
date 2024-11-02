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

var (
	// These will be set during build time
	Version   = "dev"
	CommitSHA = "none"
	BuildTime = "unknown"
)

func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}

func displayRepoStats(repo *git.Repository, stats *git.RepositoryStats) {
	// Header with repo name and commit summary
	fmt.Printf("\n📁 %s\n", repo.Name)
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("📊 %d commits across %d active days\n", 
		stats.TotalCommits, stats.ActiveDays)

	// Timeline view
	fmt.Printf("🕒 %s (%s ago) → %s (%s ago)\n",
		stats.FirstCommit.Timestamp.Format("2006-01-02"),
		formatDuration(time.Since(stats.FirstCommit.Timestamp)),
		stats.LastCommit.Timestamp.Format("2006-01-02"),
		formatDuration(time.Since(stats.LastCommit.Timestamp)))

	// Most active days
	if stats.TotalCommits > 0 {
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

		fmt.Printf("🔥 Peak activity: ")
		max := 3
		if len(days) < max {
			max = len(days)
		}
		for i := 0; i < max; i++ {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Printf("%s (%d)", days[i].day, days[i].count)
		}
		fmt.Println()
	}
	fmt.Println() // Add extra newline for spacing between repos
}

func main() {
	var rootCmd = &cobra.Command{
		Use:     "streakode",
		Short:   "Streakode is a developer motivation and insight tool",
		Long:    `Track your coding streaks, get insights, and stay motivated with Streakode`,
		Version: Version,
	}

	// Add version template to show more build info
	rootCmd.SetVersionTemplate(`Streakode Version: {{.Version}}
Commit: {{.CommitSHA}}
Built: {{.BuildTime}}
`)

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

			fmt.Printf("\nFound %d repositories! 🎉\n", len(repos))
			for _, repo := range repos {
				fmt.Printf("• %s\n", repo.Name)
			}
		},
	}

	var statsCmd = &cobra.Command{
		Use:   "stats [path]",
		Short: "Show your coding stats and streaks",
		Run: func(cmd *cobra.Command, args []string) {
			// If no path provided, use current directory
			if len(args) == 0 {
				args = append(args, ".")
			}

			scanner := git.NewScanner(args)
			repos, err := scanner.ScanForRepositories()
			if err != nil {
				log.Fatal(err)
			}

			// Add a small delay to make the scanning feel more natural
			time.Sleep(500 * time.Millisecond)

			for _, repo := range repos {
				stats, err := repo.AnalyzeRepository()
				if err != nil {
					fmt.Printf("⚠️  Error analyzing %s: %v\n", repo.Name, err)
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
