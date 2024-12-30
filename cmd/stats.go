package cmd

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/AccursedGalaxy/streakode/cache"
	"github.com/AccursedGalaxy/streakode/config"
	"github.com/AccursedGalaxy/streakode/scan"
	"github.com/charmbracelet/lipgloss"
	"github.com/jedib0t/go-pretty/v6/table"
	"golang.org/x/term"
)

type repoInfo struct {
	name       string
	metadata   scan.RepoMetadata
	lastCommit time.Time
}

type CommitTrend struct {
	indicator string
	text      string
}

type LanguageStats map[string]int
type HourStats map[int]int

const (
	defaultTerminalWidth = 80
	maxTableWidth        = 120
	hoursInDay           = 24
	daysInWeek           = 7
)

var calculator = &DefaultStatsCalculator{}

func (c *DefaultStatsCalculator) CalculateCommitTrend(current int, previous int) CommitTrend {
	diff := current - previous
	switch {
	case diff > 0:
		return CommitTrend{"↗️", fmt.Sprintf("up %d", diff)}
	case diff < 0:
		return CommitTrend{"↘️", fmt.Sprintf("down %d", -diff)}
	default:
		return CommitTrend{"-", ""}
	}
}

// DisplayStats - Displays stats for all active repositories or a specific repository
func DisplayStats(targetRepo string) {
	// Get table width from the rendered table first
	projectsSection := buildProjectsSection(targetRepo)
	tableLines := strings.Split(projectsSection, "\n")
	if len(tableLines) == 0 {
		return
	}

	// Get the actual table width from the first line (including borders)
	tableWidth := len([]rune(tableLines[0])) // use runes to handle Unicode characters correctly

	// Create styles with calculated width - match table width exactly
	style := lipgloss.NewStyle()
	headerStyle := style.
		Bold(true).
		Foreground(lipgloss.Color(config.AppConfig.Colors.HeaderColor)).
		Width(tableWidth).
		Align(lipgloss.Center)

	// Build sections dynamically
	var sections []string

	// Header section
	if config.AppConfig.DisplayStats.ShowWelcomeMessage {
		headerText := fmt.Sprintf("🚀 %s's Coding Activity", config.AppConfig.Author)
		if targetRepo != "" {
			headerText = fmt.Sprintf("🚀 %s's Activity in %s", config.AppConfig.Author, targetRepo)
		}
		padding := (tableWidth - len([]rune(headerText))) / 2
		centeredHeader := fmt.Sprintf("%*s%s%*s", padding, "", headerText, padding, "")
		sections = append(sections, headerStyle.Render(centeredHeader))
	}

	// Active projects section (table)
	if config.AppConfig.DisplayStats.ShowActiveProjects && projectsSection != "" {
		sections = append(sections, projectsSection)
	}

	// Insights section
	if config.AppConfig.DisplayStats.ShowInsights {
		insights := buildInsightsSection(targetRepo)
		if insights != "" {
			sections = append(sections, insights)
		}
	}

	// Join sections with dividers only if configured
	output := ""
	if config.AppConfig.ShowDividers {
		divider := strings.Repeat("─", tableWidth)
		for i, section := range sections {
			if section == "" {
				continue
			}
			if i > 0 {
				output += "\n" + divider + "\n"
			}
			output += strings.TrimSpace(section)
		}
	} else {
		// Join sections directly without dividers
		for _, section := range sections {
			if section != "" {
				if output != "" {
					output += "\n"
				}
				output += strings.TrimSpace(section)
			}
		}
	}

	fmt.Println(output)
}

func (c *DefaultStatsCalculator) CalculateTableWidth() int {
	width, _, err := term.GetSize(0)
	if err != nil {
		width = defaultTerminalWidth
	}
	return min(width-2, maxTableWidth)
}

// prepareRepoData converts the cache map into a sorted slice of repository information
func prepareRepoData() []repoInfo {
	repos := make([]repoInfo, 0, len(cache.Cache))
	for path, repo := range cache.Cache {
		repoName := path[strings.LastIndex(path, "/")+1:]
		repos = append(repos, repoInfo{
			name:       repoName,
			metadata:   repo,
			lastCommit: repo.LastCommit,
		})
	}

	// Sort by most recent activity
	sort.Slice(repos, func(i, j int) bool {
		return repos[i].lastCommit.After(repos[j].lastCommit)
	})

	return repos
}

// initializeTable creates and configures a new table writer with proper settings
func initializeTable(tableWidth int) table.Writer {
	t := table.NewWriter()

	// Configure table column widths
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, WidthMax: int(float64(tableWidth) * 0.35)}, // Repository name
		{Number: 2, WidthMax: int(float64(tableWidth) * 0.15)}, // Weekly commits
		{Number: 3, WidthMax: int(float64(tableWidth) * 0.15)}, // Streak
		{Number: 4, WidthMax: int(float64(tableWidth) * 0.20)}, // Changes
		{Number: 5, WidthMax: int(float64(tableWidth) * 0.15)}, // Last activity
	})

	// Set overall table width
	t.SetAllowedRowLength(tableWidth)

	// Add Table Header if Set in config
	if config.AppConfig.DisplayStats.TableStyle.UseTableHeader {
		t.AppendHeader(table.Row{
			"Repo",
			"Weekly",
			"Streak",
			"Changes",
			"Activity",
		})
	}

	return t
}

// formatActivityIndicator determines the activity indicator based on commit count
func formatActivityIndicator(weeklyCommits int) string {
	indicators := config.AppConfig.DisplayStats.ActivityIndicators
	thresholds := config.AppConfig.DisplayStats.Thresholds

	if weeklyCommits > thresholds.HighActivity {
		return indicators.HighActivity
	} else if weeklyCommits == 0 {
		return indicators.NoActivity
	}
	return indicators.NormalActivity
}

// formatStreakString formats the streak display with appropriate indicators
func formatStreakString(currentStreak, longestStreak int) string {
	indicators := config.AppConfig.DisplayStats.ActivityIndicators
	streakStr := fmt.Sprintf("%dd", currentStreak)

	if currentStreak == longestStreak && currentStreak > 0 {
		streakStr += indicators.StreakRecord
	} else if currentStreak > 0 {
		streakStr += indicators.ActiveStreak
	}

	return streakStr
}

// calculateWeeklyChanges calculates total additions and deletions for the week
func calculateWeeklyChanges(commitHistory []scan.CommitHistory) (int, int) {
	var weeklyAdditions, weeklyDeletions int
	weekStart := time.Now().AddDate(0, 0, -daysInWeek)

	for _, commit := range commitHistory {
		if commit.Date.After(weekStart) {
			weeklyAdditions += commit.Additions
			weeklyDeletions += commit.Deletions
		}
	}

	return weeklyAdditions, weeklyDeletions
}

// formatLastActivity formats the time since last commit
func formatLastActivity(lastCommit time.Time) string {
	if hours := time.Since(lastCommit).Hours(); hours > hoursInDay {
		return fmt.Sprintf("%dd ago", int(hours/hoursInDay))
	}
	return "today"
}

// buildProjectsSection - Displays stats for all active repositories or a specific repository
func buildProjectsSection(targetRepo string) string {
	if !config.AppConfig.DisplayStats.ShowActiveProjects {
		return ""
	}

	// Create buffer for table
	buf := new(bytes.Buffer)

	// Get sorted repo data
	repos := prepareRepoData()

	// Filter for target repository if specified
	if targetRepo != "" {
		filteredRepos := make([]repoInfo, 0)
		for _, repo := range repos {
			if repo.name == targetRepo {
				filteredRepos = append(filteredRepos, repo)
				break
			}
		}
		if len(filteredRepos) == 0 {
			fmt.Printf("Repository '%s' not found in cache. Run 'streakode cache reload' to update cache.\n", targetRepo)
			return ""
		}
		repos = filteredRepos
	}

	// Initialize table with proper width
	tableWidth := calculator.CalculateTableWidth()
	t := initializeTable(tableWidth)
	t.SetOutputMirror(buf)

	// Process each repository
	displayCount := min(len(repos), config.AppConfig.DisplayStats.MaxProjects)
	for i := 0; i < displayCount; i++ {
		repo := repos[i]
		meta := repo.metadata

		activity := formatActivityIndicator(meta.WeeklyCommits)
		streakStr := formatStreakString(meta.CurrentStreak, meta.LongestStreak)
		weeklyAdd, weeklyDel := calculateWeeklyChanges(meta.CommitHistory)
		activityStr := formatLastActivity(repo.lastCommit)
		changesStr := fmt.Sprintf("+%d/-%d", weeklyAdd, weeklyDel)

		t.AppendRow(table.Row{
			repo.name,
			fmt.Sprintf("%d%s", meta.WeeklyCommits, activity),
			streakStr,
			changesStr,
			activityStr,
		})
	}

	// Apply table style
	switch strings.ToLower(config.AppConfig.DisplayStats.TableStyle.Style) {
	case "rounded":
		t.SetStyle(table.StyleRounded)
	case "bold":
		t.SetStyle(table.StyleBold)
	case "light":
		t.SetStyle(table.StyleLight)
	case "double":
		t.SetStyle(table.StyleDouble)
	default:
		t.SetStyle(table.StyleDefault)
	}

	t.Render()
	return buf.String()
}

func formatLanguages(stats map[string]int, topCount int) string {
	// Language icons mapping with more descriptive emojis
	languageIcons := map[string]string{
		"go":      config.AppConfig.LanguageSettings.LanguageDisplay.GoDisplay,
		"py":      config.AppConfig.LanguageSettings.LanguageDisplay.PythonDisplay,
		"lua":     config.AppConfig.LanguageSettings.LanguageDisplay.LuaDisplay,
		"js":      config.AppConfig.LanguageSettings.LanguageDisplay.JavaDisplay,
		"ts":      config.AppConfig.LanguageSettings.LanguageDisplay.TypeScriptDisplay,
		"rust":    config.AppConfig.LanguageSettings.LanguageDisplay.RustDisplay,
		"cpp":     config.AppConfig.LanguageSettings.LanguageDisplay.CppDisplay,
		"c":       config.AppConfig.LanguageSettings.LanguageDisplay.CDisplay,
		"java":    config.AppConfig.LanguageSettings.LanguageDisplay.JavaDisplay,
		"ruby":    config.AppConfig.LanguageSettings.LanguageDisplay.RubyDisplay,
		"php":     config.AppConfig.LanguageSettings.LanguageDisplay.PHPDisplay,
		"html":    config.AppConfig.LanguageSettings.LanguageDisplay.HTMLDisplay,
		"css":     config.AppConfig.LanguageSettings.LanguageDisplay.CSSDisplay,
		"shell":   config.AppConfig.LanguageSettings.LanguageDisplay.ShellDisplay,
		"default": config.AppConfig.LanguageSettings.LanguageDisplay.DefaultDisplay,
	}

	// Convert map to slice for sorting
	type langStat struct {
		lang  string
		lines int
	}

	langs := make([]langStat, 0, len(stats))
	for lang, lines := range stats {
		cleanLang := strings.ToLower(strings.TrimPrefix(lang, "."))
		langs = append(langs, langStat{cleanLang, lines})
	}

	// Sort by line count descending
	sort.Slice(langs, func(i, j int) bool {
		return langs[i].lines > langs[j].lines
	})

	// Calculate size needed for formatted slice
	size := 0
	for i := 0; i < min(len(langs), topCount); i++ {
		if langs[i].lines > 0 {
			size++
		}
	}

	// Format languages with icons and better number formatting
	formatted := make([]string, 0, size)
	for i := 0; i < min(len(langs), topCount); i++ {
		if langs[i].lines > 0 {
			// Retrieve icon or default if not found
			icon := languageIcons[langs[i].lang]
			if icon == "" {
				icon = languageIcons["default"]
			}

			// Format lines of code with appropriate unit
			var sizeStr string
			switch {
			case langs[i].lines >= 1000000:
				sizeStr = fmt.Sprintf("%.1fM", float64(langs[i].lines)/1000000)
			case langs[i].lines >= 1000:
				sizeStr = fmt.Sprintf("%.1fK", float64(langs[i].lines)/1000)
			default:
				sizeStr = fmt.Sprintf("%d", langs[i].lines)
			}

			// Format with icon, language, and size
			formatted = append(formatted, fmt.Sprintf("%s (%s)",
				icon, sizeStr))
		}
	}

	return strings.Join(formatted, "  ")
}

func getTableStyle() table.Style {
	return table.Style{
		Options: table.Options{
			DrawBorder:      config.AppConfig.DisplayStats.TableStyle.Options.DrawBorder,
			SeparateColumns: config.AppConfig.DisplayStats.TableStyle.Options.SeparateColumns,
			SeparateHeader:  config.AppConfig.DisplayStats.TableStyle.Options.SeparateHeader,
			SeparateRows:    config.AppConfig.DisplayStats.TableStyle.Options.SeparateRows,
		},
		Box: table.BoxStyle{
			PaddingLeft:    "",
			PaddingRight:   " ",
			MiddleVertical: "",
		},
	}
}

func (c *DefaultStatsCalculator) ProcessLanguageStats(cache map[string]scan.RepoMetadata) map[string]int {
	languageStats := make(map[string]int)
	for _, repo := range cache {
		for lang, lines := range repo.Languages {
			languageStats[lang] += lines
		}
	}
	return languageStats
}

// calculateGlobalStats calculates overall statistics across all repositories
func calculateGlobalStats(repos map[string]scan.RepoMetadata) (int, int, int, int, int, map[int]int) {
	totalWeeklyCommits := 0
	lastWeeksCommits := 0
	totalMonthlyCommits := 0
	totalAdditions := 0
	totalDeletions := 0
	hourStats := make(map[int]int)

	weekStart := time.Now().AddDate(0, 0, -daysInWeek)
	for _, repo := range repos {
		if repo.Dormant {
			continue
		}

		totalWeeklyCommits += repo.WeeklyCommits
		lastWeeksCommits += repo.LastWeeksCommits
		totalMonthlyCommits += repo.MonthlyCommits

		for _, commit := range repo.CommitHistory {
			if commit.Date.After(weekStart) {
				totalAdditions += commit.Additions
				totalDeletions += commit.Deletions
				hourStats[commit.Date.Hour()]++
			}
		}
	}

	return totalWeeklyCommits, lastWeeksCommits, totalMonthlyCommits, totalAdditions, totalDeletions, hourStats
}

// findPeakCodingHour determines the hour with the most commits
func findPeakCodingHour(hourStats map[int]int) (int, int) {
	peakHour := 0
	peakCommits := 0

	for hour, commits := range hourStats {
		if commits > peakCommits {
			peakHour = hour
			peakCommits = commits
		}
	}

	return peakHour, peakCommits
}

// formatWeeklySummary creates a formatted weekly summary string
func formatWeeklySummary(totalWeeklyCommits int, commitTrend CommitTrend, totalAdditions, totalDeletions int) string {
	return fmt.Sprintf("%d commits (%s %s), +%d/-%d lines",
		totalWeeklyCommits,
		commitTrend.indicator,
		commitTrend.text,
		totalAdditions,
		totalDeletions)
}

type insightStats struct {
	weeklyCommits int
	additions     int
	deletions     int
	peakHour      int
	peakCommits   int
	commitTrend   CommitTrend
	languageStats map[string]int
}

// appendInsightRows adds insight rows to the table based on configuration
func appendInsightRows(t table.Writer, insights struct {
	TopLanguagesCount int  `mapstructure:"top_languages_count"`
	ShowDailyAverage  bool `mapstructure:"show_daily_average"`
	ShowTopLanguages  bool `mapstructure:"show_top_languages"`
	ShowPeakCoding    bool `mapstructure:"show_peak_coding"`
	ShowWeeklySummary bool `mapstructure:"show_weekly_summary"`
	ShowWeeklyGoal    bool `mapstructure:"show_weekly_goal"`
	ShowMostActive    bool `mapstructure:"show_most_active"`
}, stats insightStats) {
	if insights.ShowWeeklySummary {
		summary := formatWeeklySummary(stats.weeklyCommits, stats.commitTrend, stats.additions, stats.deletions)
		t.AppendRow(table.Row{"📈", "Weekly Summary:", summary})
	}

	if insights.ShowDailyAverage {
		t.AppendRow(table.Row{"📊", "Daily Average:",
			fmt.Sprintf("%.1f commits", float64(stats.weeklyCommits)/daysInWeek)})
	}

	if insights.ShowTopLanguages && len(stats.languageStats) > 0 {
		langs := formatLanguages(stats.languageStats, insights.TopLanguagesCount)
		t.AppendRow(table.Row{"💻", "Top Languages:", langs})
	}

	if insights.ShowPeakCoding {
		t.AppendRow(table.Row{"⏰", "Peak Coding:",
			fmt.Sprintf("%02d:00-%02d:00 (%d commits)",
				stats.peakHour, (stats.peakHour+1)%hoursInDay, stats.peakCommits)})
	}

	if insights.ShowWeeklyGoal && config.AppConfig.GoalSettings.WeeklyCommitGoal > 0 {
		progress := float64(stats.weeklyCommits) / float64(config.AppConfig.GoalSettings.WeeklyCommitGoal) * 100
		t.AppendRow(table.Row{"🎯", "Weekly Goal:",
			fmt.Sprintf("%d%% (%d/%d commits)",
				int(progress), stats.weeklyCommits, config.AppConfig.GoalSettings.WeeklyCommitGoal)})
	}
}

// buildSimpleInsights creates a simple insight string for non-detailed view
func buildSimpleInsights(repos map[string]scan.RepoMetadata) string {
	if !config.AppConfig.DisplayStats.InsightSettings.ShowMostActive {
		return ""
	}

	var mostProductiveRepo string
	maxActivity := 0
	for path, repo := range repos {
		if repo.WeeklyCommits > maxActivity {
			maxActivity = repo.WeeklyCommits
			mostProductiveRepo = path[strings.LastIndex(path, "/")+1:]
		}
	}

	if mostProductiveRepo != "" {
		return fmt.Sprintf("  🌟 Most active: %s", mostProductiveRepo)
	}
	return ""
}

// buildInsightsSection - Displays insights about coding activity
func buildInsightsSection(targetRepo string) string {
	if !config.AppConfig.DisplayStats.ShowInsights {
		return ""
	}

	// Get the same terminal width as used elsewhere
	tableWidth := calculator.CalculateTableWidth()
	insights := config.AppConfig.DisplayStats.InsightSettings

	// Filter cache for target repository if specified
	var repoCache map[string]scan.RepoMetadata
	repoCache = cache.Cache
	if targetRepo != "" {
		filteredCache := make(map[string]scan.RepoMetadata)
		for path, repo := range cache.Cache {
			if strings.HasSuffix(path, "/"+targetRepo) {
				filteredCache[path] = repo
				break
			}
		}
		if len(filteredCache) == 0 {
			return ""
		}
		repoCache = filteredCache
	}

	if config.AppConfig.DetailedStats {
		t := table.NewWriter()
		t.SetStyle(getTableStyle())
		t.SetAllowedRowLength(tableWidth - 2)

		// Calculate all stats
		weeklyCommits, lastWeeksCommits, _, additions, deletions, hourStats := calculateGlobalStats(repoCache)
		peakHour, peakCommits := findPeakCodingHour(hourStats)
		commitTrend := calculator.CalculateCommitTrend(weeklyCommits, lastWeeksCommits)
		languageStats := calculator.ProcessLanguageStats(repoCache)

		// Append rows based on configuration
		appendInsightRows(t, insights, insightStats{
			weeklyCommits: weeklyCommits,
			additions:     additions,
			deletions:     deletions,
			peakHour:      peakHour,
			peakCommits:   peakCommits,
			commitTrend:   commitTrend,
			languageStats: languageStats,
		})

		return t.Render()
	}

	return buildSimpleInsights(repoCache)
}

func (rc *DefaultRepoCache) GetRepos() map[string]scan.RepoMetadata {
	return rc.cache
}
