package scan

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/AccursedGalaxy/streakode/config"
)

type RepoMetadata struct {
	Path           string    `json:"path"`
	LastCommit     time.Time `json:"last_commit"`
	CommitCount    int       `json:"commit_count"`
	CurrentStreak  int       `json:"current_streak"`
	LongestStreak  int       `json:"longest_streak"`
	WeeklyCommits  int       `json:"weekly_commits"`
	MonthlyCommits int       `json:"monthly_commits"`
	MostActiveDay  string    `json:"most_active_day"`
	LastActivity   string    `json:"last_activity"`
	AuthorVerified bool      `json:"author_verified"`
	Dormant        bool      `json:"dormant"`
}

// fetchRepoMeta - gets metadata for a single repository and verifies user
func fetchRepoMeta(repoPath, author string) RepoMetadata {
	meta := RepoMetadata{Path: repoPath}
	
	authorCmd := exec.Command("git", "-C", repoPath, "log", "--author="+author, "--pretty=format:%ci")
	output, err := authorCmd.Output()
	if err != nil {
		fmt.Printf("Error getting git log for %s: %v\n", repoPath, err)
		return meta
	}
	
	if len(output) > 0 {
		meta.AuthorVerified = true

		// Calculate streaks and commit frequencies
		dates := strings.Split(string(output), "\n")
		lastCommitTime, _ := time.Parse("2006-01-02 15:04:05 -0700", dates[0])
		meta.LastCommit = lastCommitTime
		meta.CommitCount = len(dates)
		
		meta.CurrentStreak = calculateStreak(dates)
		meta.WeeklyCommits = countRecentCommits(dates, 7)
		meta.MonthlyCommits = countRecentCommits(dates, 30)
		meta.MostActiveDay = findMostActiveDay(dates)

		meta.Dormant = time.Since(lastCommitTime) > time.Duration(config.AppConfig.DormantThreshold) * 24 * time.Hour
	}

	return meta
}

// ScanDirectories - scans for Git repositories in the specified directories
func ScanDirectories(dirs []string, author string) ([]RepoMetadata, error) {
	var repos []RepoMetadata

	for _, dir := range dirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info == nil {
				return nil
			}
			if info.IsDir() && info.Name() == ".git" {
				repoPath := filepath.Dir(path)
				meta := fetchRepoMeta(repoPath, author)
				if meta.AuthorVerified {
					if !meta.Dormant {
						repos = append(repos, meta)
					}
				}
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("error walking directory %s: %v", dir, err)
		}
	}

	return repos, nil
}

// calculateStreak - calculates the current streak of commits
func calculateStreak(dates []string) int {
	if len(dates) == 0 {
		return 0
	}

	streak := 1
	lastDate, _ := time.Parse("2006-01-02 15:04:05 -0700", dates[0])
	currentDay := lastDate.Format("2006-01-02")

	for i := 1; i < len(dates); i++ {
		commitDate, err := time.Parse("2006-01-02 15:04:05 -0700", dates[i])
		if err != nil {
			continue
		}

		commitDay := commitDate.Format("2006-01-02")
		dayDiff := lastDate.Sub(commitDate).Hours() / 24

		// If it's a different day and the difference is 1 day
		if commitDay != currentDay && dayDiff <= 1 {
			streak++
			lastDate = commitDate
			currentDay = commitDay
		}
	}

	return streak
}

// CountRecentCommits - counts the number of commits in the last n days
func countRecentCommits(dates []string, days int) int {
	cutoff := time.Now().AddDate(0, 0, -days)
	count := 0
	for _, dateStr := range dates {
		commitDate, err := time.Parse("2006-01-02 15:04:05 -0700", dateStr)
		if err != nil {
			continue
		}
		if commitDate.Before(cutoff) {
			break
		}
		count++
	}
	return count
}

// FindMostActiveDay - finds the most active day in the last n days
func findMostActiveDay(dates []string) string {
	dayCount := make(map[string]int)
	for _, dateStr := range dates {
		commitDate, err := time.Parse("2006-01-02 15:04:05 -0700", dateStr)
		if err != nil {
			continue
		}
		day := commitDate.Weekday().String()
		dayCount[day]++
	}

	maxDay := ""
	maxCount := 0
	for day, count := range dayCount {
		if count > maxCount {
			maxDay = day
			maxCount = count
		}
	}
	return maxDay
}