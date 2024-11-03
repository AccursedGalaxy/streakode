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
	
	// First, let's get the configured Git user info for this repo
	configCmd := exec.Command("git", "-C", repoPath, "config", "--get-regexp", "^user\\.(name|email)$")
	configOutput, _ := configCmd.Output()
	
	// Build a list of possible author patterns
	authorPatterns := []string{
		author,                         // Exact match
		fmt.Sprintf("%s <.*>", author), // Name with any email
	}
	
	// Add configured git user if available
	if len(configOutput) > 0 {
		lines := strings.Split(string(configOutput), "\n")
		var userName, userEmail string
		for _, line := range lines {
			if strings.HasPrefix(line, "user.name ") {
				userName = strings.TrimPrefix(line, "user.name ")
			} else if strings.HasPrefix(line, "user.email ") {
				userEmail = strings.TrimPrefix(line, "user.email ")
			}
		}
		if userName != "" && userEmail != "" {
			authorPatterns = append(authorPatterns, 
				fmt.Sprintf("%s <%s>", userName, userEmail))
		}
	}
	
	// Try each author pattern
	for _, pattern := range authorPatterns {
		authorCmd := exec.Command("git", "-C", repoPath, "log", "--all", 
			"--author="+pattern, "--pretty=format:%ci")
		output, err := authorCmd.Output()
		if err == nil && len(output) > 0 {
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
			
			break // We found matching commits, no need to try other patterns
		}
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
	now := time.Now()
	cutoff := now.AddDate(0, 0, -days)
	
	// Use a map to track unique days with commits
	uniqueDays := make(map[string]bool)
	
	for _, dateStr := range dates {
		commitDate, err := time.Parse("2006-01-02 15:04:05 -0700", dateStr)
		if err != nil {
			continue
		}
		
		// Only count if the commit is after the cutoff and before now
		if commitDate.After(cutoff) && commitDate.Before(now) {
			// Use date only (no time) as the key to count unique days
			dateOnly := commitDate.Format("2006-01-02")
			uniqueDays[dateOnly] = true
		}
	}

	return len(uniqueDays)
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
