package scan

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/AccursedGalaxy/streakode/config"
)

type CommitHistory struct {
	Date			time.Time	`json:"date"`
	Hash			string		`json:"hash"`
	MessageHead		string		`json:"message_head"`
	FileCount		int			`json:"file_count"`
	Additions		int			`json:"additions"`
	Deletions		int			`json:"deletions"`
}

type DailyStats struct {
	Date			time.Time	`json:"date"`
	Commits			int			`json:"commits"`
	Lines			int			`json:"lines"`
	Files			int			`json:"files"`
}

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

	CommitHistory	[]CommitHistory 		`json:"commit_history"`
	DailyStats		map[string]DailyStats  	`json:"daily_stats"`
	LastAnalyzed	time.Time  				`json:"last_analyzed"`
	TotalLines		int 					`json:"total_lines"`
	TotalFiles 		int 					`json:"total_files"`
	Languages  		map[string]int 			`json:"languages"`
	Contributors	map[string]int 			`json:"contributors"`

}

// fetchRepoMeta - gets metadata for a single repository and verifies user
func fetchRepoMeta(repoPath, author string) RepoMetadata {
	meta := RepoMetadata{
		Path:         repoPath,
		DailyStats:   make(map[string]DailyStats),
		Languages:    make(map[string]int),
		Contributors: make(map[string]int),
		LastAnalyzed: time.Now(),
	}
	
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
			
			// Fetch detailed commit history (last 365 days by default)
			since := time.Now().AddDate(-1, 0, 0)
			if history, err := fetchDetailedCommitInfo(repoPath, author, since); err == nil {
				meta.CommitHistory = history
				
				// Aggregate daily stats
				for _, commit := range history {
					dateStr := commit.Date.Format("2006-01-02")
					stats := meta.DailyStats[dateStr]
					stats.Date, _ = time.Parse("2006-01-02", dateStr)
					stats.Commits++
					stats.Lines += commit.Additions - commit.Deletions
					stats.Files += commit.FileCount
					meta.DailyStats[dateStr] = stats
				}
			}
			
			// Fetch language statistics
			if langs, err := fetchLanguageStats(repoPath); err == nil {
				meta.Languages = langs
			}

			// Debug Printing for entire infrmatoin fetched for testing
			/*
			fmt.Printf("\n=== Debug Info for Repository: %s ===\n", repoPath)
			fmt.Printf("Author Verified: %v\n", meta.AuthorVerified)
			fmt.Printf("Commit Count: %d\n", meta.CommitCount)
			fmt.Printf("Current Streak: %d days\n", meta.CurrentStreak)
			fmt.Printf("Longest Streak: %d days\n", meta.LongestStreak)
			fmt.Printf("Last Commit: %v\n", meta.LastCommit)
			fmt.Printf("Weekly Commits: %d\n", meta.WeeklyCommits)
			fmt.Printf("Monthly Commits: %d\n", meta.MonthlyCommits)
			fmt.Printf("Most Active Day: %s\n", meta.MostActiveDay)
			fmt.Printf("Dormant: %v\n", meta.Dormant)

			fmt.Printf("\nLanguage Distribution:\n")
			for lang, lines := range meta.Languages {
				fmt.Printf("  %s: %d lines\n", lang, lines)
			}
			
			fmt.Printf("\nDaily Stats (Last 7 Days):\n")
			now := time.Now()
			for i := 0; i < 7; i++ {
				date := now.AddDate(0, 0, -i).Format("2006-01-02")
				if stats, ok := meta.DailyStats[date]; ok {
					fmt.Printf("  %s: %d commits, %d lines, %d files\n", 
						date, stats.Commits, stats.Lines, stats.Files)
				}
			}
			fmt.Printf("\n" */)
			
			break // We found matching commits, no need to try other patterns
		}
	}

	return meta
}

func fetchDetailedCommitInfo(repoPath string, author string, since time.Time) ([]CommitHistory, error) {
    var history []CommitHistory
    
    // Get detailed git log with stats
    cmd := exec.Command("git", "-C", repoPath, "log",
        "--all",
        "--author="+author,
        "--pretty=format:%H|%aI|%s",
        "--numstat",
        "--after="+since.Format("2006-01-02"))
    
    output, err := cmd.Output()
    if err != nil {
        return nil, err
    }

    // Parse the git log output
    lines := strings.Split(string(output), "\n")
    var currentCommit *CommitHistory
    
    for _, line := range lines {
        if strings.Contains(line, "|") {
            // This is a commit header line
            parts := strings.Split(line, "|")
            if len(parts) == 3 {
                if currentCommit != nil {
                    history = append(history, *currentCommit)
                }
                
                commitTime, _ := time.Parse(time.RFC3339, parts[1])
                currentCommit = &CommitHistory{
                    Hash:        parts[0],
                    Date:        commitTime,
                    MessageHead: parts[2],
                }
            }
        } else if line != "" && currentCommit != nil {
            // This is a stats line
            parts := strings.Fields(line)
            if len(parts) == 3 {
                additions, _ := strconv.Atoi(parts[0])
                deletions, _ := strconv.Atoi(parts[1])
                currentCommit.Additions += additions
                currentCommit.Deletions += deletions
                currentCommit.FileCount++
            }
        }
    }
    
    if currentCommit != nil {
        history = append(history, *currentCommit)
    }
    
    return history, nil
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
	
	count := 0
	for _, dateStr := range dates {
		commitDate, err := time.Parse("2006-01-02 15:04:05 -0700", dateStr)
		if err != nil {
			continue
		}
		
		// Count all commits within the time range
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

func fetchLanguageStats(repoPath string) (map[string]int, error) {
	languages := make(map[string]int)
	
	// Use git ls-files to get all tracked files
	cmd := exec.Command("git", "-C", repoPath, "ls-files")
	output, err := cmd.Output()
	if err != nil {
		return languages, err
	}
	
	files := strings.Split(string(output), "\n")
	for _, file := range files {
		if ext := filepath.Ext(file); ext != "" {
			// Count lines in file
			if lines, err := countFileLines(filepath.Join(repoPath, file)); err == nil {
				languages[ext] += lines
			}
		}
	}
	
	return languages, nil
}

// Helper function to count lines in a file
func countFileLines(filePath string) (int, error) {
    content, err := os.ReadFile(filePath)
    if err != nil {
        return 0, err
    }
    return len(strings.Split(string(content), "\n")), nil
}

// Add these utility functions to analyze the enhanced data

func (m *RepoMetadata) GetCommitTrend(days int) map[string]int {
    trend := make(map[string]int)
    since := time.Now().AddDate(0, 0, -days)
    
    for _, commit := range m.CommitHistory {
        if commit.Date.After(since) {
            dateStr := commit.Date.Format("2006-01-02")
            trend[dateStr]++
        }
    }
    return trend
}

func (m *RepoMetadata) GetLanguageDistribution() map[string]float64 {
    total := 0
    dist := make(map[string]float64)
    
    for _, lines := range m.Languages {
        total += lines
    }
    
    if total > 0 {
        for lang, lines := range m.Languages {
            dist[lang] = float64(lines) / float64(total) * 100
        }
    }
    
    return dist
}