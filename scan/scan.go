package scan

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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

// TimeSlot represents a 24-hour time period divided into slots
type TimeSlot struct {
    Hour    int     `json:"hour"`
    Commits int     `json:"commits"`
    Lines   int     `json:"lines"`
}

// VelocityMetrics represents coding velocity over different time periods
type VelocityMetrics struct {
    DailyAverage   float64 `json:"daily_average"`
    WeeklyTrend    float64 `json:"weekly_trend"`    // Percentage change from previous week
    MonthlyTrend   float64 `json:"monthly_trend"`   // Percentage change from previous month
    PeakHours      []TimeSlot `json:"peak_hours"`
}

type RepoMetadata struct {
	Path           string    `json:"path"`
	LastCommit     time.Time `json:"last_commit"`
	CommitCount    int       `json:"commit_count"`
	CurrentStreak  int       `json:"current_streak"`
	LongestStreak  int       `json:"longest_streak"`
	WeeklyCommits  int       `json:"weekly_commits"`
	LastWeeksCommits	int	 `json:"last_weeks_commits"`
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

// DateRange represents a time period with start and end dates
type DateRange struct {
	Start time.Time
	End   time.Time
}

// GetCurrentWeekRange returns the date range for the current week (Monday to Sunday)
func GetCurrentWeekRange() DateRange {
	now := time.Now()
	
	// Find the most recent Monday
	weekday := now.Weekday()
	daysFromMonday := int(weekday)
	if weekday == time.Sunday {
		daysFromMonday = 7
	}
	
	// Calculate start of week (Monday at 00:00:00)
	startDate := now.AddDate(0, 0, -daysFromMonday+1)
	startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
	
	// Calculate end of week (next Monday at 00:00:00)
	endDate := startDate.AddDate(0, 0, 7)
	
	return DateRange{Start: startDate, End: endDate}
}

// GetPreviousWeekRange returns the date range for the previous week (Monday to Sunday)
func GetPreviousWeekRange() DateRange {
	currentWeek := GetCurrentWeekRange()
	return DateRange{
		Start: currentWeek.Start.AddDate(0, 0, -7),
		End:   currentWeek.Start,
	}
}

// GetMonthRange returns the date range for the specified number of months back
func GetMonthRange(monthsBack int) DateRange {
	now := time.Now()
	
	// Calculate start of the target month
	startDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	startDate = startDate.AddDate(0, -monthsBack, 0)
	
	// Calculate end of the month (23:59:59 of the last day)
	endDate := time.Date(startDate.Year(), startDate.Month()+1, 1, 0, 0, 0, 0, startDate.Location())
	endDate = endDate.Add(-time.Second) // Move to 23:59:59 of the previous day
	
	return DateRange{Start: startDate, End: endDate}
}

// IsInDateRange checks if a date falls within a date range (inclusive start, exclusive end)
func IsInDateRange(date time.Time, dateRange DateRange) bool {
	return (date.Equal(dateRange.Start) || date.After(dateRange.Start)) && date.Before(dateRange.End)
}

// countCommitsInRange counts commits within a specific date range
func countCommitsInRange(dates []string, dateRange DateRange) int {
	count := 0
	for _, dateStr := range dates {
		commitDate, err := time.Parse("2006-01-02 15:04:05 -0700", dateStr)
		if err != nil {
			continue
		}

		if IsInDateRange(commitDate, dateRange) {
			count++
		}
	}
	return count
}

// Refactored version of countLastWeeksCommits using date ranges
func countLastWeeksCommits(dates []string) int {
	previousWeek := GetPreviousWeekRange()
	if config.AppConfig.Debug {
		fmt.Printf("Debug: Previous week range: %s to %s\n", 
			previousWeek.Start.Format("2006-01-02 (Monday)"), 
			previousWeek.End.Format("2006-01-02 (Monday)"))
	}
	return countCommitsInRange(dates, previousWeek)
}

// Refactored version of countRecentCommits using date ranges
func countRecentCommits(dates []string, days int) int {
	now := time.Now()
	dateRange := DateRange{
		Start: now.AddDate(0, 0, -days),
		End:   now,
	}
	if config.AppConfig.Debug {
		fmt.Printf("Debug: Recent commits range: %s to %s\n",
			dateRange.Start.Format("2006-01-02"),
			dateRange.End.Format("2006-01-02"))
	}
	return countCommitsInRange(dates, dateRange)
}

// Refactored version of countCommitsInPeriod using DateRange
func countCommitsInPeriod(history []CommitHistory, start, end time.Time) int {
    dateRange := DateRange{Start: start, End: end}
    count := 0
    for _, commit := range history {
        if IsInDateRange(commit.Date, dateRange) {
            count++
        }
    }
    return count
}

// fetchRepoMeta - gets metadata for a single repository and verifies user
func fetchRepoMeta(repoPath, author string) RepoMetadata {
	meta := RepoMetadata{
		Path:         repoPath,
		LastAnalyzed: time.Now(),
	}

	// Check if directory exists and is accessible
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		fmt.Printf("Warning: Directory not found: %s\n", repoPath)
		return meta
	} else if err != nil {
		fmt.Printf("Warning: Cannot access directory: %s - %v\n", repoPath, err)
		return meta
	}

	// Get commit dates in a single git command
	authorCmd := exec.Command("git", "-C", repoPath, "log", "--all",
		"--author="+author, "--pretty=format:%ci")
	output, err := authorCmd.Output()
	if err == nil && len(output) > 0 {
		meta.AuthorVerified = true
		dates := strings.Split(string(output), "\n")
		meta.CommitCount = len(dates)

		// Parse first date for last commit
		if lastCommitTime, err := time.Parse("2006-01-02 15:04:05 -0700", dates[0]); err == nil {
			meta.LastCommit = lastCommitTime
			meta.Dormant = time.Since(meta.LastCommit) > time.Duration(config.AppConfig.DormantThreshold) * 24 * time.Hour
		}

		// Quick stats that we always need
		meta.WeeklyCommits = countRecentCommits(dates, 7)
		meta.MonthlyCommits = countRecentCommits(dates, 30)

		// Gather Commits From Last Week
		meta.LastWeeksCommits = countLastWeeksCommits(dates)

		// Only compute streak info if the repo is active
		if !meta.Dormant {
			streakInfo := calculateStreakInfo(dates)
			meta.CurrentStreak = streakInfo.Current
			meta.LongestStreak = streakInfo.Longest
			meta.MostActiveDay = findMostActiveDay(dates)
		}

		// Only compute detailed stats if explicitly configured
		if config.AppConfig.DetailedStats {
			// fmt.Printf("Debug - Detailed stats enabled for %s\n", repoPath)
			meta.initDetailedStats()
			meta.updateDetailedStats(repoPath, author)
			// fmt.Printf("Debug - After update: CommitHistory length = %d\n", len(meta.CommitHistory))
		}
	}

	return meta
}

// Initialize maps only when needed
func (m *RepoMetadata) initDetailedStats() {
	m.DailyStats = make(map[string]DailyStats)
	m.Languages = make(map[string]int)
	m.Contributors = make(map[string]int)
}

func (m *RepoMetadata) updateDetailedStats(repoPath, author string) {
	since := time.Now().AddDate(0, 0, -30) // Only fetch last 30 days for detailed stats

	// Fetch commit history
	if history, err := fetchDetailedCommitInfo(repoPath, author, since); err == nil {
		m.CommitHistory = history
	} else {
		fmt.Printf("Error collecting detailed stats for %s: %v\n", repoPath, err)
	}

	// Fetch language statistics
	if languages, err := fetchLanguageStats(repoPath); err == nil {
		m.Languages = languages

		// Calculate total lines across all languages
		totalLines := 0
		for _, lines := range languages {
			totalLines += lines
		}
		m.TotalLines = totalLines
	} else {
		fmt.Printf("Error collecting language stats for %s: %v\n", repoPath, err)
	}
}

func fetchDetailedCommitInfo(repoPath string, author string, since time.Time) ([]CommitHistory, error) {
    var history []CommitHistory

    // Get detailed git log with stats
    gitCmd := exec.Command("git", "-C", repoPath, "log",
        "--all",
        "--author="+author,
        "--pretty=format:%H|%aI|%s",
        "--numstat",
        "--after="+since.Format("2006-01-02"))

    // fmt.Printf("Debug - Running git command: %v\n", gitCmd.String())

    output, err := gitCmd.Output()
    if err != nil {
        return nil, fmt.Errorf("git command failed: %v", err)
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
func ScanDirectories(dirs []string, author string, shouldExclude func(string) bool) ([]RepoMetadata, error) {
	var repos []RepoMetadata
	var skippedDirs []string

	for _, dir := range dirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			// Handle directory access errors gracefully
			if err != nil {
				skippedDirs = append(skippedDirs, dir)
				return filepath.SkipDir
			}
			if info == nil {
				return nil
			}
			if info.IsDir() && info.Name() == ".git" {
				repoPath := filepath.Dir(path)
				if shouldExclude(repoPath) {
					return nil
				}
				meta := fetchRepoMeta(repoPath, author)
				if meta.AuthorVerified {
					if !meta.Dormant {
						repos = append(repos, meta)
					}
				}
			}
			return nil
		})

		// Handle initial directory access error
		if err != nil {
			skippedDirs = append(skippedDirs, dir)
			continue // Skip to next directory instead of returning error
		}
	}

	// Print warnings for skipped directories
	if len(skippedDirs) > 0 {
		fmt.Println("\nWarning: The following directories were skipped due to access issues:")
		for _, dir := range skippedDirs {
			fmt.Printf("- %s\n", dir)
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
		return languages, fmt.Errorf("git ls-files failed: %v", err)
	}

	files := strings.Split(string(output), "\n")
	for _, file := range files {
		if file == "" {
			continue
		}

		if ext := filepath.Ext(file); ext != "" {
			// Skip excluded extensions
			if isExcludedExtension(ext) {
				continue
			}

			// Count lines in file
			fullPath := filepath.Join(repoPath, file)
			if lines, err := countFileLines(fullPath); err == nil {
				// Only add if it meets the minimum lines threshold
				if lines >= config.AppConfig.LanguageSettings.MinimumLines {
					languages[ext] += lines
				}
			}
		}
	}

	return languages, nil
}

// Helper function to check if an extension is excluded
func isExcludedExtension(ext string) bool {
	for _, excluded := range config.AppConfig.LanguageSettings.ExcludedExtensions {
		if strings.EqualFold(ext, excluded) {
			return true
		}
	}
	return false
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

    // Calculate total excluding unwanted languages
    for lang, lines := range m.Languages {
        if !isExcludedExtension(lang) {
            total += lines
        }
    }

    if total > 0 {
        for lang, lines := range m.Languages {
            if !isExcludedExtension(lang) {
                dist[lang] = float64(lines) / float64(total) * 100
            }
        }
    }

    return dist
}

func (m *RepoMetadata) CalculatePeakHours() []TimeSlot {
    hourStats := make(map[int]*TimeSlot)

    // Initialize all hours
    for i := 0; i < 24; i++ {
        hourStats[i] = &TimeSlot{Hour: i}
    }

    // Aggregate commit data by hour
    for _, commit := range m.CommitHistory {
        hour := commit.Date.Hour()
        slot := hourStats[hour]
        slot.Commits++
        slot.Lines += commit.Additions + commit.Deletions
    }

    // Convert to slice and sort by commit count
    peaks := make([]TimeSlot, 0, 24)
    for _, slot := range hourStats {
        peaks = append(peaks, *slot)
    }

    // Sort by commit count descending
    sort.Slice(peaks, func(i, j int) bool {
        return peaks[i].Commits > peaks[j].Commits
    })

    // Return top 5 peak hours
    if len(peaks) > 5 {
        return peaks[:5]
    }
    return peaks
}

func (m *RepoMetadata) CalculateVelocity() VelocityMetrics {
    now := time.Now()
    metrics := VelocityMetrics{}

    // Calculate daily average (last 30 days)
    thirtyDaysAgo := now.AddDate(0, 0, -30)
    recentCommits := 0
    for _, commit := range m.CommitHistory {
        if commit.Date.After(thirtyDaysAgo) {
            recentCommits++
        }
    }
    metrics.DailyAverage = float64(recentCommits) / 30.0

    // Calculate weekly trend
    currentWeek := countCommitsInPeriod(m.CommitHistory, now.AddDate(0, 0, -7), now)
    previousWeek := countCommitsInPeriod(m.CommitHistory, now.AddDate(0, 0, -14), now.AddDate(0, 0, -7))
    if previousWeek > 0 {
        metrics.WeeklyTrend = (float64(currentWeek) - float64(previousWeek)) / float64(previousWeek) * 100
    }

    // Calculate monthly trend
    currentMonth := countCommitsInPeriod(m.CommitHistory, now.AddDate(0, -1, 0), now)
    previousMonth := countCommitsInPeriod(m.CommitHistory, now.AddDate(0, -2, 0), now.AddDate(0, -1, 0))
    if previousMonth > 0 {
        metrics.MonthlyTrend = (float64(currentMonth) - float64(previousMonth)) / float64(previousMonth) * 100
    }

    // Calculate peak hours
    metrics.PeakHours = m.CalculatePeakHours()

    return metrics
}
