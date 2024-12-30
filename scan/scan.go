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
	Date        time.Time `json:"date"`
	Hash        string    `json:"hash"`
	MessageHead string    `json:"message_head"`
	FileCount   int       `json:"file_count"`
	Additions   int       `json:"additions"`
	Deletions   int       `json:"deletions"`
}

type DailyStats struct {
	Date    time.Time `json:"date"`
	Commits int       `json:"commits"`
	Lines   int       `json:"lines"`
	Files   int       `json:"files"`
}

// TimeSlot represents a 24-hour time period divided into slots
type TimeSlot struct {
	Hour    int `json:"hour"`
	Commits int `json:"commits"`
	Lines   int `json:"lines"`
}

// VelocityMetrics represents coding velocity over different time periods
type VelocityMetrics struct {
	DailyAverage float64    `json:"daily_average"`
	WeeklyTrend  float64    `json:"weekly_trend"`  // Percentage change from previous week
	MonthlyTrend float64    `json:"monthly_trend"` // Percentage change from previous month
	PeakHours    []TimeSlot `json:"peak_hours"`
}

type RepoMetadata struct {
	Path             string    `json:"path"`
	LastCommit       time.Time `json:"last_commit"`
	CommitCount      int       `json:"commit_count"`
	CurrentStreak    int       `json:"current_streak"`
	LongestStreak    int       `json:"longest_streak"`
	WeeklyCommits    int       `json:"weekly_commits"`
	LastWeeksCommits int       `json:"last_weeks_commits"`
	MonthlyCommits   int       `json:"monthly_commits"`
	MostActiveDay    string    `json:"most_active_day"`
	LastActivity     string    `json:"last_activity"`
	AuthorVerified   bool      `json:"author_verified"`
	Dormant          bool      `json:"dormant"`

	CommitHistory []CommitHistory       `json:"commit_history"`
	DailyStats    map[string]DailyStats `json:"daily_stats"`
	LastAnalyzed  time.Time             `json:"last_analyzed"`
	TotalLines    int                   `json:"total_lines"`
	TotalFiles    int                   `json:"total_files"`
	Languages     map[string]int        `json:"languages"`
	Contributors  map[string]int        `json:"contributors"`
}

// DateRange represents a time period with start (inclusive) and end (exclusive) dates
type DateRange struct {
	Start time.Time
	End   time.Time
}

// IsInDateRange checks if a date falls within a date range
// The range is inclusive of the start date and exclusive of the end date
func IsInDateRange(date time.Time, dateRange DateRange) bool {
	dateYMD := date.Format("2006-01-02")
	startYMD := dateRange.Start.Format("2006-01-02")
	endYMD := dateRange.End.Format("2006-01-02")

	return dateYMD >= startYMD && dateYMD < endYMD
}

// GetCurrentWeekRange returns the date range for the current week (Monday to Sunday)
func GetCurrentWeekRange() DateRange {
	now := time.Now()

	if config.AppConfig.Debug {
		fmt.Printf("Debug: Calculating week range\n")
		fmt.Printf("Debug: Current time: %s\n", now.Format("2006-01-02 15:04:05 -0700"))
		fmt.Printf("Debug: Current weekday: %s\n", now.Weekday())
	}

	// Get the start of the current day
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Calculate days since last Monday
	daysFromMonday := int(now.Weekday())
	if daysFromMonday == 0 { // Sunday
		daysFromMonday = 7
	}

	// Calculate start of week
	startDate := startOfDay.AddDate(0, 0, -daysFromMonday+1)

	if config.AppConfig.Debug {
		fmt.Printf("Debug: Start of day: %s\n", startOfDay.Format("2006-01-02"))
		fmt.Printf("Debug: Days from Monday: %d\n", daysFromMonday)
		fmt.Printf("Debug: Week start date: %s\n", startDate.Format("2006-01-02"))
	}

	return DateRange{
		Start: startDate,
		End:   startDate.AddDate(0, 0, 7),
	}
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

// countCommitsInRange counts commits within a specific date range
func countCommitsInRange(dates []string, dateRange DateRange) int {
	count := 0
	uniqueDays := make(map[string]bool)
	parsedDates := make(map[string]time.Time) // Cache parsed dates

	startYMD := dateRange.Start.Format("2006-01-02")
	endYMD := dateRange.End.Format("2006-01-02")

	if config.AppConfig.Debug {
		fmt.Printf("Debug: Counting commits between %s and %s\n", startYMD, endYMD)
	}

	for _, dateStr := range dates {
		// Check cache first
		commitDate, ok := parsedDates[dateStr]
		if !ok {
			var err error
			commitDate, err = time.Parse("2006-01-02 15:04:05 -0700", dateStr)
			if err != nil {
				continue
			}
			parsedDates[dateStr] = commitDate
		}

		dayKey := commitDate.Format("2006-01-02")
		if dayKey >= startYMD && dayKey < endYMD {
			count++
			uniqueDays[dayKey] = true
			if config.AppConfig.Debug && count%10 == 0 {
				fmt.Printf("Debug: Found %d commits across %d unique days\n",
					count, len(uniqueDays))
			}
		}
	}

	if config.AppConfig.Debug {
		fmt.Printf("Debug: Final count: %d commits across %d unique days\n",
			count, len(uniqueDays))
	}
	return count
}

// Refactored version of countLastWeeksCommits using date ranges
func countLastWeeksCommits(dates []string) int {
	previousWeek := GetPreviousWeekRange()
	if config.AppConfig.Debug {
		fmt.Printf("Debug: Calculating last week's commits (%s to %s)\n",
			previousWeek.Start.Format("2006-01-02"),
			previousWeek.End.Format("2006-01-02"))
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

	if config.AppConfig.Debug {
		fmt.Printf("Debug: Counting commits between %s and %s\n",
			start.Format("2006-01-02 15:04:05"),
			end.Format("2006-01-02 15:04:05"))
	}

	for _, commit := range history {
		if IsInDateRange(commit.Date, dateRange) {
			count++
			if config.AppConfig.Debug {
				fmt.Printf("Debug: Found commit in range: %s\n",
					commit.Date.Format("2006-01-02 15:04:05"))
			}
		}
	}

	if config.AppConfig.Debug {
		fmt.Printf("Debug: Found %d commits in period\n", count)
	}

	return count
}

// fetchRepoMeta - gets metadata for a single repository and verifies user
func fetchRepoMeta(repoPath, author string) RepoMetadata {
	if config.AppConfig.Debug {
		fmt.Printf("\nDebug: Fetching metadata for repo: %s (author: %s)\n", repoPath, author)
	}

	meta := RepoMetadata{
		Path:         repoPath,
		LastAnalyzed: time.Now(),
	}

	// Check if directory exists and is accessible
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		if config.AppConfig.Debug {
			fmt.Printf("Debug: Directory not found: %s\n", repoPath)
		}
		return meta
	}

	// Get commit dates in a single git command
	authorCmd := exec.Command("git", "-C", repoPath, "log", "--all",
		"--author="+author, "--pretty=format:%ci")

	if config.AppConfig.Debug {
		fmt.Printf("Debug: Running git command: %v\n", authorCmd.String())
	}

	output, err := authorCmd.Output()
	if err != nil {
		if config.AppConfig.Debug {
			fmt.Printf("Debug: Git command failed: %v\n", err)
		}
		return meta
	}

	if len(output) > 0 {
		meta.AuthorVerified = true
		dates := strings.Split(string(output), "\n")
		meta.CommitCount = len(dates)

		if config.AppConfig.Debug {
			fmt.Printf("Debug: Found %d commits\n", meta.CommitCount)
		}

		// Parse first date for last commit
		if lastCommitTime, err := time.Parse("2006-01-02 15:04:05 -0700", dates[0]); err == nil {
			meta.LastCommit = lastCommitTime
			meta.Dormant = time.Since(meta.LastCommit) > time.Duration(config.AppConfig.DormantThreshold)*24*time.Hour

			if config.AppConfig.Debug {
				fmt.Printf("Debug: Last commit: %s (Dormant: %v)\n",
					meta.LastCommit.Format("2006-01-02 15:04:05"),
					meta.Dormant)
			}
		}

		// Quick stats
		meta.WeeklyCommits = countRecentCommits(dates, 7)
		monthlyTotal := countRecentCommits(dates, 30)
		meta.MonthlyCommits = monthlyTotal
		meta.LastWeeksCommits = countLastWeeksCommits(dates)

		if config.AppConfig.Debug {
			fmt.Printf("Debug: Weekly commits: %d\n", meta.WeeklyCommits)
			fmt.Printf("Debug: Monthly commits: %d\n", monthlyTotal)
			fmt.Printf("Debug: Last week's commits: %d\n", meta.LastWeeksCommits)
		}

		// Only compute streak info if the repo is active
		if !meta.Dormant {
			streakInfo := calculateStreakInfo(dates)
			meta.CurrentStreak = streakInfo.Current
			meta.LongestStreak = streakInfo.Longest
			meta.MostActiveDay = findMostActiveDay(dates)

			if config.AppConfig.Debug {
				fmt.Printf("Debug: Current streak: %d days\n", meta.CurrentStreak)
				fmt.Printf("Debug: Longest streak: %d days\n", meta.LongestStreak)
				fmt.Printf("Debug: Most active day: %s\n", meta.MostActiveDay)
			}
		}

		// Detailed stats if configured
		if config.AppConfig.DetailedStats {
			if config.AppConfig.Debug {
				fmt.Println("Debug: Collecting detailed stats...")
			}
			meta.initDetailedStats()
			meta.updateDetailedStats(repoPath, author)
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

	dateCache := NewDateCache()
	uniqueDates := make(map[string]bool)
	var sortedDates []string

	if config.AppConfig.Debug {
		fmt.Printf("Debug: Processing %d commit dates\n", len(dates))
	}

	// First pass - parse dates and get unique days
	for _, dateStr := range dates {
		date, err := time.Parse("2006-01-02 15:04:05 -0700", dateStr)
		if err != nil {
			continue
		}

		ymd := date.Format("2006-01-02")
		if !uniqueDates[ymd] {
			uniqueDates[ymd] = true
			sortedDates = append(sortedDates, ymd)
			dateCache.Add(dateStr, date)
		}
	}

	// Sort dates in reverse chronological order
	sort.Sort(sort.Reverse(sort.StringSlice(sortedDates)))

	if config.AppConfig.Debug {
		fmt.Printf("Debug: Found %d unique dates\n", len(sortedDates))
	}

	// Calculate current streak
	currentStreak := 0
	longestStreak := 0
	today := time.Now().Format("2006-01-02")

	// Check if there's a commit today
	if sortedDates[0] == today {
		currentStreak = 1
		longestStreak = 1
	}

	// Calculate streaks
	for i := 0; i < len(sortedDates)-1; i++ {
		current, _ := time.Parse("2006-01-02", sortedDates[i])
		next, _ := time.Parse("2006-01-02", sortedDates[i+1])

		dayDiff := current.Sub(next).Hours() / 24

		if config.AppConfig.Debug && i < 5 { // Limit debug output
			fmt.Printf("Debug: Comparing %s and %s (%.1f days)\n",
				sortedDates[i], sortedDates[i+1], dayDiff)
		}

		if dayDiff <= 1 {
			currentStreak++
			if currentStreak > longestStreak {
				longestStreak = currentStreak
			}
		} else {
			if config.AppConfig.Debug {
				fmt.Printf("Debug: Streak break found after %d days\n", currentStreak)
			}
			break
		}
	}

	if config.AppConfig.Debug {
		fmt.Printf("Debug: Final streaks - Current: %d, Longest: %d\n",
			currentStreak, longestStreak)
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
	if config.AppConfig.Debug {
		fmt.Printf("Debug: Fetching language stats for %s\n", repoPath)
	}

	languages := make(map[string]int)

	cmd := exec.Command("git", "-C", repoPath, "ls-files")
	output, err := cmd.Output()
	if err != nil {
		if config.AppConfig.Debug {
			fmt.Printf("Debug: Git ls-files failed: %v\n", err)
		}
		return languages, fmt.Errorf("git ls-files failed: %v", err)
	}

	files := strings.Split(string(output), "\n")
	if config.AppConfig.Debug {
		fmt.Printf("Debug: Found %d tracked files\n", len(files))
	}

	for _, file := range files {
		if file == "" {
			continue
		}

		if ext := filepath.Ext(file); ext != "" {
			if isExcludedExtension(ext) {
				if config.AppConfig.Debug {
					fmt.Printf("Debug: Skipping excluded extension: %s\n", ext)
				}
				continue
			}

			fullPath := filepath.Join(repoPath, file)
			if lines, err := countFileLines(fullPath); err == nil {
				if lines >= config.AppConfig.LanguageSettings.MinimumLines {
					languages[ext] += lines
					if config.AppConfig.Debug {
						fmt.Printf("Debug: Added %d lines for %s (%s)\n", lines, file, ext)
					}
				}
			} else if config.AppConfig.Debug {
				fmt.Printf("Debug: Error counting lines in %s: %v\n", file, err)
			}
		}
	}

	if config.AppConfig.Debug {
		fmt.Println("Debug: Language statistics:")
		for lang, lines := range languages {
			fmt.Printf("Debug: %s: %d lines\n", lang, lines)
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

// Add at the top of the file
type DateCache struct {
	parsed map[string]time.Time
	ymd    map[string]string
}

func NewDateCache() *DateCache {
	return &DateCache{
		parsed: make(map[string]time.Time),
		ymd:    make(map[string]string),
	}
}

func (dc *DateCache) GetParsedDate(dateStr string) (time.Time, bool) {
	date, ok := dc.parsed[dateStr]
	return date, ok
}

func (dc *DateCache) GetYMD(dateStr string) (string, bool) {
	ymd, ok := dc.ymd[dateStr]
	return ymd, ok
}

func (dc *DateCache) Add(dateStr string, date time.Time) {
	dc.parsed[dateStr] = date
	dc.ymd[dateStr] = date.Format("2006-01-02")
}

// ValidationResult holds the results of data validation
type ValidationResult struct {
	Valid  bool
	Issues []string
}

func (m *RepoMetadata) ValidateData() ValidationResult {
	result := ValidationResult{Valid: true}

	// Validate commit counts
	weeklyTotal := 0
	uniqueDaysThisWeek := make(map[string]bool)

	// Get this week's range (Monday to Sunday)
	now := time.Now()
	weekRange := GetCurrentWeekRange()

	if config.AppConfig.Debug {
		fmt.Printf("Debug: Weekly commit window: %s 00:00:00 to %s 00:00:00\n",
			weekRange.Start.Format("2006-01-02"),
			weekRange.End.Format("2006-01-02"))
		fmt.Printf("Debug: Validating weekly commits from %s to %s\n",
			weekRange.Start.Format("2006-01-02"),
			weekRange.End.Format("2006-01-02"))
	}

	// Get monthly range
	monthRange := DateRange{
		Start: time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()),
		End:   now.AddDate(0, 1, 0), // Add one month to include current month
	}

	monthlyTotal := 0
	var lastCommit time.Time
	var lastCommitDay string

	for _, commit := range m.CommitHistory {
		// Track last commit
		if lastCommit.IsZero() || commit.Date.After(lastCommit) {
			lastCommit = commit.Date
			lastCommitDay = commit.Date.Format("2006-01-02")
		}

		// Count weekly commits
		if IsInDateRange(commit.Date, weekRange) {
			weeklyTotal++
			uniqueDaysThisWeek[commit.Date.Format("2006-01-02")] = true
		}

		// Count monthly commits
		if IsInDateRange(commit.Date, monthRange) {
			monthlyTotal++
		}
	}

	// Update weekly commits in metadata
	m.WeeklyCommits = weeklyTotal

	if weeklyTotal != m.WeeklyCommits {
		result.Issues = append(result.Issues,
			fmt.Sprintf("Weekly commit mismatch: counted %d, stored %d",
				weeklyTotal, m.WeeklyCommits))
		result.Valid = false
	}

	// Validate streak if we have commits
	if !lastCommit.IsZero() {
		daysSinceLastCommit := int(now.Sub(lastCommit).Hours() / 24)

		if config.AppConfig.Debug {
			fmt.Printf("Debug: Days since last commit: %d (last commit: %s)\n",
				daysSinceLastCommit, lastCommitDay)
		}

		// Verify current streak with grace period
		if m.CurrentStreak > 0 {
			if daysSinceLastCommit > 2 {
				result.Issues = append(result.Issues,
					fmt.Sprintf("Invalid current streak: %d (more than 2 days since last commit)",
						m.CurrentStreak))
				result.Valid = false
			} else if daysSinceLastCommit == 2 && now.Hour() >= 23 {
				// Only fail if it's near the end of the grace period
				result.Issues = append(result.Issues,
					fmt.Sprintf("Invalid current streak: %d (grace period ending)",
						m.CurrentStreak))
				result.Valid = false
			}
		}
	}

	// Validate language statistics
	totalLines := 0
	for _, lines := range m.Languages {
		totalLines += lines
	}
	if totalLines != m.TotalLines {
		result.Issues = append(result.Issues,
			fmt.Sprintf("Language lines mismatch: sum %d, stored %d",
				totalLines, m.TotalLines))
		result.Valid = false
	}

	// Update monthly commits in metadata
	m.MonthlyCommits = monthlyTotal

	if monthlyTotal != m.MonthlyCommits {
		result.Issues = append(result.Issues,
			fmt.Sprintf("Monthly commit mismatch: counted %d, stored %d",
				monthlyTotal, m.MonthlyCommits))
		result.Valid = false
	}

	if config.AppConfig.Debug {
		fmt.Printf("\nDebug: Validation Summary for %s:\n", m.Path)
		fmt.Printf("- Weekly commits: counted=%d, stored=%d\n", weeklyTotal, m.WeeklyCommits)
		fmt.Printf("- Monthly commits: counted=%d, stored=%d\n", monthlyTotal, m.MonthlyCommits)
		fmt.Printf("- Current streak: %d days (last commit: %s)\n",
			m.CurrentStreak, lastCommitDay)
		fmt.Printf("- Language lines: sum=%d, stored=%d\n", totalLines, m.TotalLines)

		if result.Valid {
			fmt.Printf("Debug: Data validation passed for %s\n", m.Path)
		} else {
			fmt.Printf("Debug: Data validation failed for %s:\n", m.Path)
			for _, issue := range result.Issues {
				fmt.Printf("Debug: - %s\n", issue)
			}
		}
	}

	return result
}
