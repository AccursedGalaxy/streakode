package cmd

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"sync"
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
	// Get pre-calculated display stats from cache
	displayStats := cache.Cache.GetDisplayStats()
	if displayStats == nil {
		fmt.Println("No stats available. Try running 'cache reload' first.")
		return
	}

	// Filter repo stats if target repo is specified
	var repoStats []cache.RepoDisplayStats
	if targetRepo != "" {
		for _, rs := range displayStats.RepoStats {
			if rs.Name == targetRepo {
				repoStats = append(repoStats, rs)
				break
			}
		}
		if len(repoStats) == 0 {
			fmt.Printf("Repository '%s' not found.\n", targetRepo)
			return
		}
	} else {
		repoStats = displayStats.RepoStats
	}

	// Calculate table width
	tableWidth := calculator.CalculateTableWidth()

	// Build projects table first to get actual width
	var tableOutput string
	if len(repoStats) > 0 {
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

		// Apply table style based on config
		var style table.Style
		switch strings.ToLower(config.AppConfig.DisplayStats.TableStyle.Style) {
		case "rounded":
			style = table.StyleRounded
		case "bold":
			style = table.StyleBold
		case "light":
			style = table.StyleLight
		case "double":
			style = table.StyleDouble
		default:
			style = table.StyleDefault
		}

		// Customize style based on config options
		style.Options.DrawBorder = config.AppConfig.DisplayStats.TableStyle.Options.DrawBorder
		style.Options.SeparateColumns = config.AppConfig.DisplayStats.TableStyle.Options.SeparateColumns
		style.Options.SeparateHeader = config.AppConfig.DisplayStats.TableStyle.Options.SeparateHeader
		style.Options.SeparateRows = config.AppConfig.DisplayStats.TableStyle.Options.SeparateRows

		t.SetStyle(style)

		// Add rows
		for _, rs := range repoStats {
			activityText := formatActivityText(rs.LastCommitTime)
			t.AppendRow(table.Row{
				rs.Name,
				fmt.Sprintf("%d%s", rs.WeeklyCommits, formatActivityIndicator(rs.WeeklyCommits)),
				formatStreakString(rs.CurrentStreak, rs.LongestStreak),
				fmt.Sprintf("+%d/-%d", rs.Additions, rs.Deletions),
				activityText,
			})
		}
		tableOutput = t.Render()
	}

	// Get actual table width from first line
	var actualWidth int
	if tableOutput != "" {
		lines := strings.Split(tableOutput, "\n")
		if len(lines) > 0 {
			actualWidth = len([]rune(lines[0])) // Use runes to handle Unicode characters correctly
		}
	} else {
		actualWidth = tableWidth
	}

	// Build header with actual table width
	var sections []string
	if config.AppConfig.DisplayStats.ShowWelcomeMessage {
		headerText := fmt.Sprintf("🚀 %s's Coding Activity", config.AppConfig.Author)
		if targetRepo != "" {
			headerText = fmt.Sprintf("🚀 %s's Activity in %s", config.AppConfig.Author, targetRepo)
		}

		// Calculate padding manually for perfect centering
		textWidth := len([]rune(headerText))
		leftPadding := (actualWidth - textWidth) / 2
		rightPadding := actualWidth - textWidth - leftPadding

		// Build centered header with exact padding
		centeredHeader := fmt.Sprintf("%s%s%s",
			strings.Repeat(" ", leftPadding),
			headerText,
			strings.Repeat(" ", rightPadding))

		style := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(config.AppConfig.Colors.HeaderColor))

		sections = append(sections, style.Render(centeredHeader))
	}

	// Add table to sections
	if tableOutput != "" {
		sections = append(sections, tableOutput)
	}

	// Build insights section
	if config.AppConfig.DisplayStats.ShowInsights {
		t := table.NewWriter()
		t.SetStyle(getTableStyle())
		t.SetAllowedRowLength(tableWidth - 2)

		// Weekly summary
		trend := "↗️"
		if displayStats.WeeklyDiff < 0 {
			trend = "↘️"
		}
		weeklyText := fmt.Sprintf("📈 Weekly Summary: %d commits (%s %s), +%d/-%d lines",
			displayStats.WeeklyTotal,
			trend,
			formatDiff(displayStats.WeeklyDiff),
			displayStats.TotalAdditions,
			displayStats.TotalDeletions)
		sections = append(sections, weeklyText)

		// Daily average
		dailyText := fmt.Sprintf("📊 Daily Average:  %.1f commits", displayStats.DailyAverage)
		sections = append(sections, dailyText)

		// Language stats
		if len(displayStats.LanguageStats) > 0 {
			langText := "💻 Top Languages:  " + formatLanguageStats(displayStats.LanguageStats)
			sections = append(sections, langText)
		}

		// Peak coding hour
		peakText := fmt.Sprintf("⏰ Peak Coding:    %02d:00-%02d:00 (%d commits)",
			displayStats.PeakHour,
			(displayStats.PeakHour+1)%24,
			displayStats.PeakCommits)
		sections = append(sections, peakText)

		// Weekly goal (hardcoded for now, can be made configurable later)
		const weeklyGoal = 200 // commits per week
		progress := float64(displayStats.WeeklyTotal) / float64(weeklyGoal) * 100
		goalText := fmt.Sprintf("🎯 Weekly Goal:    %d%% (%d/%d commits)",
			int(progress),
			displayStats.WeeklyTotal,
			weeklyGoal)
		sections = append(sections, goalText)
	}

	// Join sections
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

func formatDiff(diff int) string {
	if diff < 0 {
		return fmt.Sprintf("down %d", -diff)
	}
	return fmt.Sprintf("up %d", diff)
}

func formatLanguageStats(stats map[string]int) string {
	type langStat struct {
		name  string
		lines int
	}
	var sorted []langStat
	for lang, lines := range stats {
		sorted = append(sorted, langStat{lang, lines})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].lines > sorted[j].lines
	})

	var result []string
	for i, ls := range sorted {
		if i >= 3 {
			break
		}
		icon := getLanguageIcon(ls.name)
		result = append(result, fmt.Sprintf("%s %s (%.1fK)", icon, ls.name, float64(ls.lines)/1000))
	}
	return strings.Join(result, "  ")
}

func getLanguageIcon(lang string) string {
	icons := map[string]string{
		"Go":         "🔵",
		"Java":       "☕",
		"Python":     "🐍",
		"JavaScript": "💛",
		"TypeScript": "💙",
		"Rust":       "🦀",
		"C++":        "⚡",
		"C":          "⚡",
		"Ruby":       "💎",
		"Shell":      "🐚",
		"File":       "📄",
	}
	if icon, ok := icons[lang]; ok {
		return icon
	}
	return "📄"
}

func formatActivityText(lastCommit time.Time) string {
	duration := time.Since(lastCommit)
	switch {
	case duration < 24*time.Hour:
		return "today"
	case duration < 48*time.Hour:
		return "1d ago"
	case duration < 72*time.Hour:
		return "2d ago"
	case duration < 96*time.Hour:
		return "3d ago"
	default:
		return fmt.Sprintf("%dd ago", int(duration.Hours()/24))
	}
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
	// Pre-allocate slice with capacity
	repos := make([]repoInfo, 0, cache.Cache.Len())

	// Use parallel processing for large datasets
	if cache.Cache.Len() > 10 {
		var mu sync.Mutex
		var wg sync.WaitGroup

		cache.Cache.Range(func(path string, repo scan.RepoMetadata) bool {
			wg.Add(1)
			go func(p string, r scan.RepoMetadata) {
				defer wg.Done()
				repoName := p[strings.LastIndex(p, "/")+1:]
				info := repoInfo{
					name:       repoName,
					metadata:   r,
					lastCommit: r.LastCommit,
				}
				mu.Lock()
				repos = append(repos, info)
				mu.Unlock()
			}(path, repo)
			return true
		})

		wg.Wait()
	} else {
		// Sequential processing for small datasets
		cache.Cache.Range(func(path string, repo scan.RepoMetadata) bool {
			repoName := path[strings.LastIndex(path, "/")+1:]
			repos = append(repos, repoInfo{
				name:       repoName,
				metadata:   repo,
				lastCommit: repo.LastCommit,
			})
			return true
		})
	}

	// Sort by most recent activity using direct index access
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

	// Use pre-calculated stats from cache when possible
	var repoCache = make(map[string]scan.RepoMetadata)
	if targetRepo != "" {
		cache.Cache.Range(func(path string, repo scan.RepoMetadata) bool {
			if strings.HasSuffix(path, "/"+targetRepo) {
				repoCache[path] = repo
				return false
			}
			return true
		})
		if len(repoCache) == 0 {
			return ""
		}
	} else {
		cache.Cache.Range(func(path string, repo scan.RepoMetadata) bool {
			repoCache[path] = repo
			return true
		})
	}

	if config.AppConfig.DetailedStats {
		t := table.NewWriter()
		t.SetStyle(getTableStyle())
		t.SetAllowedRowLength(tableWidth - 2)

		// Use cached stats when available
		weeklyCommits := 0
		lastWeeksCommits := 0
		additions := 0
		deletions := 0
		hourStats := make(map[int]int)
		languageStats := make(map[string]int)

		for _, repo := range repoCache {
			weeklyCommits += repo.WeeklyCommits
			lastWeeksCommits += repo.LastWeeksCommits

			// Process language stats in parallel for large repos
			if len(repo.Languages) > 10 {
				var wg sync.WaitGroup
				var mu sync.Mutex

				for lang, lines := range repo.Languages {
					wg.Add(1)
					go func(l string, count int) {
						defer wg.Done()
						mu.Lock()
						languageStats[l] += count
						mu.Unlock()
					}(lang, lines)
				}

				wg.Wait()
			} else {
				for lang, lines := range repo.Languages {
					languageStats[lang] += lines
				}
			}

			// Use pre-calculated commit stats
			for _, commit := range repo.CommitHistory {
				additions += commit.Additions
				deletions += commit.Deletions
				hour := commit.Date.Hour()
				hourStats[hour]++
			}
		}

		peakHour, peakCommits := findPeakCodingHour(hourStats)
		commitTrend := calculator.CalculateCommitTrend(weeklyCommits, lastWeeksCommits)

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
