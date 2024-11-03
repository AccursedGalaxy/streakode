package scan

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
  "sort"

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

	// Sort dates in ascending order (oldest to newest)
	sort.Slice(dates, func(i, j int) bool {
		date1, _ := time.Parse("2006-01-02 15:04:05 -0700", dates[i])
		date2, _ := time.Parse("2006-01-02 15:04:05 -0700", dates[j])
		return date1.Before(date2)
	})

	// Initialize streak variables
	currentStreak := 1
	longestStreak := 1
	lastDate, _ := time.Parse("2006-01-02 15:04:05 -0700", dates[0])

	// Calculate longest streak and current streak
	for i := 1; i < len(dates); i++ {
		commitDate, err := time.Parse("2006-01-02 15:04:05 -0700", dates[i])
		if err != nil {
			continue
		}

		// Check if the commit is the next day
		if commitDate.Sub(lastDate).Hours() < 48 && commitDate.Sub(lastDate).Hours() >= 24 {
			currentStreak++
		} else if commitDate.Sub(lastDate).Hours() >= 48 {
			// Reset current streak if more than a day gap is found
			if currentStreak > longestStreak {
				longestStreak = currentStreak
			}
			currentStreak = 1 // Start a new streak
		}

		// Update last date to the current commitDate
		lastDate = commitDate
	}

	// Final check to update longest streak
	if currentStreak > longestStreak {
		longestStreak = currentStreak
	}

	// Check if the last streak is ongoing (today or yesterday)
	if time.Since(lastDate).Hours() > 24 {
		currentStreak = 0
	}

	return StreakInfo{Current: currentStreak, Longest: longestStreak}
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
