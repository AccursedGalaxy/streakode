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
	
	// Get commits with date only
	authorCmd := exec.Command("git", "-C", repoPath, "log", "--all", "--pretty=format:%ci")
	output, err := authorCmd.Output()
	if err != nil {
		fmt.Printf("Error getting git log for %s: %v\n", repoPath, err)
		return meta
	}

	if len(output) > 0 {
		meta.AuthorVerified = true
		dates := strings.Split(string(output), "\n")
		meta.CommitCount = len(dates)
		
		// Get both current and longest streaks
		streakInfo := calculateStreakInfo(dates)
		meta.CurrentStreak = streakInfo.Current
		meta.LongestStreak = streakInfo.Longest
		
		// Parse first date for last commit
		if lastCommitTime, err := time.Parse("2006-01-02 15:04:05 -0700", dates[0]); err == nil {
			meta.LastCommit = lastCommitTime
		}
		
		meta.WeeklyCommits = countRecentCommits(dates, 7)
		meta.MonthlyCommits = countRecentCommits(dates, 30)
		meta.MostActiveDay = findMostActiveDay(dates)

		meta.Dormant = time.Since(meta.LastCommit) > time.Duration(config.AppConfig.DormantThreshold) * 24 * time.Hour
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

	// Initialize variables
	streak := 1
	maxStreak := 1
	lastDate, _ := time.Parse("2006-01-02 15:04:05 -0700", dates[0])
	currentDay := lastDate.Format("2006-01-02")

	// Create a map to store unique days
	dayMap := make(map[string]bool)
	dayMap[currentDay] = true

	for i := 1; i < len(dates); i++ {
		commitDate, err := time.Parse("2006-01-02 15:04:05 -0700", dates[i])
		if err != nil {
			continue
		}

		commitDay := commitDate.Format("2006-01-02")
		
		// Skip if we already counted this day
		if dayMap[commitDay] {
			continue
		}
		dayMap[commitDay] = true

		// Calculate days between commits
		dayDiff := lastDate.Sub(commitDate).Hours() / 24

		// If it's consecutive (1 day difference)
		if dayDiff <= 1.5 { // Allow for timezone differences
			streak++
			if streak > maxStreak {
				maxStreak = streak
			}
		} else {
			// Break in the streak
			streak = 1
		}

		lastDate = commitDate
		currentDay = commitDay
	}

	return maxStreak
}

// Add this new function to track both current and longest streaks
type StreakInfo struct {
	Current int
	Longest int
}

func calculateStreakInfo(dates []string) StreakInfo {
	if len(dates) == 0 {
		return StreakInfo{0, 0}
	}

	// Initialize variables
	currentStreak := 1
	longestStreak := 1
	lastDate, _ := time.Parse("2006-01-02 15:04:05 -0700", dates[0])
	
	// If last commit wasn't today or yesterday, current streak should be 0
	daysSinceLastCommit := time.Since(lastDate).Hours() / 24
	if daysSinceLastCommit > 1.5 { // Using 1.5 to account for timezone differences
		return StreakInfo{0, longestStreak} // Current streak is 0, but keep longest
	}

	// Rest of streak calculation for longest streak...
	dayMap := make(map[string]bool)
	dayMap[lastDate.Format("2006-01-02")] = true

	for i := 1; i < len(dates); i++ {
		commitDate, err := time.Parse("2006-01-02 15:04:05 -0700", dates[i])
		if err != nil {
			continue
		}

		commitDay := commitDate.Format("2006-01-02")
		if dayMap[commitDay] {
			continue
		}
		dayMap[commitDay] = true

		dayDiff := lastDate.Sub(commitDate).Hours() / 24

		if dayDiff <= 1.5 { // Allow for timezone differences
			currentStreak++
			if currentStreak > longestStreak {
				longestStreak = currentStreak
			}
		} else {
			break // Stop counting current streak at first gap
		}

		lastDate = commitDate
	}

	return StreakInfo{currentStreak, longestStreak}
}

// countRecentCommits - counts the number of commits in the last n days
func countRecentCommits(dates []string, days int) int {
	now := time.Now().UTC()
	cutoff := now.AddDate(0, 0, -days).Truncate(24 * time.Hour)
	
	count := 0
	for _, dateStr := range dates {
		commitDate, err := time.Parse("2006-01-02 15:04:05 -0700", dateStr)
		if err != nil {
			continue
		}
		
		commitDate = commitDate.UTC()
		if commitDate.After(cutoff) && commitDate.Before(now) {
			count++
		}
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