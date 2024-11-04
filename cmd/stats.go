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
	"github.com/jedib0t/go-pretty/v6/text"
)

type repoInfo struct {
    name       string
    metadata   scan.RepoMetadata
    lastCommit time.Time
}

// TODO: emojis are casuing the table to get dispalyed wrongly on different devices/terminals - since emoji width might be different.
// -> Added user specified emoji width settings. -> Still causes rows where emojis are present to be lsightly wider than other rows or separators.

// DisplayStats - Displays stats for all active repositories in a more compact format
func DisplayStats() {
	// Create a test table to calculate width
	t := table.NewWriter()
	t.AppendHeader(table.Row{"Repository", "Weekly", "Streak", "Changes", "Activity"})
	
	// Use configured column widths
	cfg := config.AppConfig.DisplayStats.TableStyle.MinColumnWidths
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, WidthMin: cfg.Repository},
		{Number: 2, WidthMin: cfg.Weekly},
		{Number: 3, WidthMin: cfg.Streak},
		{Number: 4, WidthMin: cfg.Changes},
		{Number: 5, WidthMin: cfg.Activity},
	})
	
	tableWidth := len(strings.Split(t.Render(), "\n")[0])
	
	// Create styles with calculated width
	style := lipgloss.NewStyle()
	headerStyle := style.
		Bold(true).
		Foreground(lipgloss.Color(config.AppConfig.Colors.HeaderColor)).
		Width(tableWidth).
		Align(lipgloss.Center)
	
	dividerStyle := style.
		Foreground(lipgloss.Color(config.AppConfig.Colors.DividerColor)).
		Width(tableWidth)
	
	// Build sections dynamically
	var sections []string

	// Header section
	if config.AppConfig.DisplayStats.ShowWelcomeMessage {
		header := headerStyle.Render(fmt.Sprintf("🚀 %s's Coding Activity", config.AppConfig.Author))
		sections = append(sections, header)
	}

	// Weekly/Monthly stats (combine with header if enabled)
	if config.AppConfig.DisplayStats.ShowWeeklyCommits || config.AppConfig.DisplayStats.ShowMonthlyCommits {
		weeklyTotal := 0
		monthlyTotal := 0
		for _, repo := range cache.Cache {
			if !repo.Dormant {
				weeklyTotal += repo.WeeklyCommits
				monthlyTotal += repo.MonthlyCommits
			}
		}
		
		var stats []string
		if config.AppConfig.DisplayStats.ShowWeeklyCommits {
			stats = append(stats, fmt.Sprintf("%d commits this week", weeklyTotal))
		}
		if config.AppConfig.DisplayStats.ShowMonthlyCommits {
			stats = append(stats, fmt.Sprintf("%d this month", monthlyTotal))
		}
		if len(stats) > 0 {
			statLine := headerStyle.Render(fmt.Sprintf("📊 %s", strings.Join(stats, " • ")))
			sections = append(sections, statLine)
		}
	}

	// Active projects section (table)
	if config.AppConfig.DisplayStats.ShowActiveProjects {
		projects := buildProjectsSection()
		if projects != "" {
			sections = append(sections, projects)
		}
	}

	// Insights section
	if config.AppConfig.DisplayStats.ShowInsights {
		insights := buildInsightsSection()
		if insights != "" {
			sections = append(sections, insights)
		}
	}

	// Join sections with dynamically sized dividers
	divider := dividerStyle.Render(strings.Repeat("─", tableWidth))
	
	output := ""
	for i, section := range sections {
		if section == "" {
			continue
		}
		if i > 0 {
			output += "\n" + divider + "\n"
		}
		output += section
	}

	fmt.Println(output)
}

func buildProjectsSection() string {
	if !config.AppConfig.DisplayStats.ShowActiveProjects {
		return ""
	}

	// Convert map to slice for sorting
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

	// Create buffer for table
	buf := new(bytes.Buffer)
	t := table.NewWriter()
	t.SetOutputMirror(buf)

	// Configure table style
	cfg := config.AppConfig.DisplayStats.TableStyle
	style := table.Style{
			Box: table.BoxStyle{
				BottomLeft:       "└",
				BottomRight:      "┘",
				BottomSeparator:  "┴",
				Left:            cfg.ColumnSeparator,
				LeftSeparator:    "├",
				MiddleHorizontal: "─",
				MiddleSeparator:  cfg.CenterSeparator,
				MiddleVertical:   cfg.ColumnSeparator,
				PaddingLeft:      " ",
				PaddingRight:     " ",
				Right:           cfg.ColumnSeparator,
				RightSeparator:   "┤",
				TopLeft:         "┌",
				TopRight:        "┐",
				TopSeparator:    "┬",
			},
			Options: table.Options{
				DrawBorder:      cfg.ShowBorder,
				SeparateColumns: true,
				SeparateHeader:  cfg.ShowHeaderLine,
				SeparateRows:    cfg.ShowRowLines,
			},
		}
	t.SetStyle(style)

	// Set column configs with proper alignment
	headerAlign := text.AlignCenter
	if cfg.HeaderAlignment == "left" {
		headerAlign = text.AlignLeft
	} else if cfg.HeaderAlignment == "right" {
		headerAlign = text.AlignRight
	}

	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, WidthMin: cfg.MinColumnWidths.Repository, Align: text.AlignLeft, AlignHeader: headerAlign},
		{Number: 2, WidthMin: cfg.MinColumnWidths.Weekly, Align: text.AlignCenter, AlignHeader: headerAlign},
		{Number: 3, WidthMin: cfg.MinColumnWidths.Streak, Align: text.AlignCenter, AlignHeader: headerAlign},
		{Number: 4, WidthMin: cfg.MinColumnWidths.Changes, Align: text.AlignCenter, AlignHeader: headerAlign},
		{Number: 5, WidthMin: cfg.MinColumnWidths.Activity, Align: text.AlignCenter, AlignHeader: headerAlign},
	})

	displayCount := min(len(repos), config.AppConfig.DisplayStats.MaxProjects)
	for i := 0; i < displayCount; i++ {
		repo := repos[i]
		meta := repo.metadata

		// Use configured activity indicators
		indicators := config.AppConfig.DisplayStats.ActivityIndicators
		thresholds := config.AppConfig.DisplayStats.Thresholds
		
		activity := indicators.NormalActivity
		if meta.WeeklyCommits > thresholds.HighActivity {
			activity = indicators.HighActivity
		} else if meta.WeeklyCommits == 0 {
			activity = indicators.NoActivity
		}

		// Format streak with configured indicators
		streakStr := fmt.Sprintf("%dd", meta.CurrentStreak)
		if meta.CurrentStreak == meta.LongestStreak && meta.CurrentStreak > 0 {
			streakStr += indicators.StreakRecord
		} else if meta.CurrentStreak > 0 {
			streakStr += indicators.ActiveStreak
		}

		// Format activity
		activityStr := "today"
		if hours := time.Since(repo.lastCommit).Hours(); hours > 24 {
			activityStr = fmt.Sprintf("%dd ago", int(hours/24))
		}

		// Calculate weekly changes
		var weeklyAdditions, weeklyDeletions int
		weekStart := time.Now().AddDate(0, 0, -7)
		for _, commit := range meta.CommitHistory {
			if commit.Date.After(weekStart) {
				weeklyAdditions += commit.Additions
				weeklyDeletions += commit.Deletions
			}
		}
		changesStr := fmt.Sprintf("+%d/-%d", weeklyAdditions, weeklyDeletions)

		// Append row with all formatted data
		t.AppendRow(table.Row{
			repo.name,
			fmt.Sprintf("%d%s", meta.WeeklyCommits, activity),
			streakStr,
			changesStr,
			activityStr,
		})
	}

	// Render to buffer and return
	t.Render()
	return buf.String()
}

func formatLanguages(stats map[string]int, topCount int) string {
	// Convert map to slice for sorting
	type langStat struct {
		lang  string
			lines int
	}
	
	langs := make([]langStat, 0, len(stats))
	for lang, lines := range stats {
		langs = append(langs, langStat{lang, lines})
	}
	
	// Sort by line count descending
	sort.Slice(langs, func(i, j int) bool {
		return langs[i].lines > langs[j].lines
	})
	
	// Format top 3 languages
	var formatted []string
	for i := 0; i < min(len(langs), topCount); i++ {
		if langs[i].lines > 0 {
			formatted = append(formatted, fmt.Sprintf("%s:%.1fk", 
				langs[i].lang, float64(langs[i].lines)/1000))
		}
	}
	
	return strings.Join(formatted, ", ")
}

func buildInsightsSection() string {
	if !config.AppConfig.DisplayStats.ShowInsights {
		return ""
	}

	insights := config.AppConfig.DisplayStats.InsightSettings
	
	if config.AppConfig.DetailedStats {
		t := table.NewWriter()
		t.SetStyle(table.Style{
			Options: table.Options{
				DrawBorder:      false,
				SeparateColumns: true,  // Controls column separator
				SeparateHeader:  false, // Instead of SetHeaderLine
				SeparateRows:    false, // Instead of SetRowLine
			},
			Box: table.BoxStyle{
				PaddingLeft:      " ",
				PaddingRight:     " ",
				MiddleVertical:   " ", // Column separator character
			},
		})

		// Note: AutoWrap is controlled per column via ColumnConfig if needed
		t.SetColumnConfigs([]table.ColumnConfig{
			{Number: 1, WidthMax: 0}, // WidthMax: 0 disables auto-wrapping
			{Number: 2, WidthMax: 0},
			{Number: 3, WidthMax: 0},
			{Number: 4, WidthMax: 0},
			{Number: 5, WidthMax: 0},
		})

		// Calculate global stats
		totalWeeklyCommits := 0
		totalMonthlyCommits := 0
		totalAdditions := 0
		totalDeletions := 0
		languageStats := make(map[string]int)
		hourStats := make(map[int]int)
		
		// Find peak coding hour
		peakHour := 0
		peakCommits := 0
		
		for _, repo := range cache.Cache {
			if repo.Dormant {
				continue
			}
			
			totalWeeklyCommits += repo.WeeklyCommits
			totalMonthlyCommits += repo.MonthlyCommits
			
			// Aggregate language stats
			for lang, lines := range repo.Languages {
				languageStats[lang] += lines
			}
			
			// Calculate code changes and peak hours
			weekStart := time.Now().AddDate(0, 0, -7)
			for _, commit := range repo.CommitHistory {
				if commit.Date.After(weekStart) {
					totalAdditions += commit.Additions
					totalDeletions += commit.Deletions
					hour := commit.Date.Hour()
					hourStats[hour]++
					
					// Update peak hour
					if hourStats[hour] > peakCommits {
						peakHour = hour
						peakCommits = hourStats[hour]
					}
				}
			}
		}

		// Only add configured insight rows
		if insights.ShowWeeklySummary {
			t.AppendRow(table.Row{
				"📈",  // No padding needed for single-column emojis
				"Weekly Summary:", 
				fmt.Sprintf("%d commits, +%d/-%d lines", 
					totalWeeklyCommits, totalAdditions, totalDeletions)})
		}
		
		if insights.ShowDailyAverage {
			t.AppendRow(table.Row{"📊", "Daily Average:", 
				fmt.Sprintf("%.1f commits", float64(totalWeeklyCommits)/7.0)})
		}

		if insights.ShowTopLanguages && len(languageStats) > 0 {
			langs := formatLanguages(languageStats, insights.TopLanguagesCount)
			t.AppendRow(table.Row{"💻", "Top Languages:", langs})
		}

		if insights.ShowPeakCoding {
			t.AppendRow(table.Row{"⏰", "Peak Coding:", 
				fmt.Sprintf("%02d:00-%02d:00 (%d commits)", 
				peakHour, (peakHour+1)%24, peakCommits)})
		}

		if insights.ShowWeeklyGoal && config.AppConfig.GoalSettings.WeeklyCommitGoal > 0 {
			progress := float64(totalWeeklyCommits) / float64(config.AppConfig.GoalSettings.WeeklyCommitGoal) * 100
			t.AppendRow(table.Row{"🎯", "Weekly Goal:", 
				fmt.Sprintf("%d%% (%d/%d commits)", 
				int(progress), totalWeeklyCommits, config.AppConfig.GoalSettings.WeeklyCommitGoal)})
		}

		return t.Render()
	} else {
		// Simple insights for non-detailed view
		if insights.ShowMostActive {
			var mostProductiveRepo string
			maxActivity := 0
			for path, repo := range cache.Cache {
				if repo.WeeklyCommits > maxActivity {
					maxActivity = repo.WeeklyCommits
					mostProductiveRepo = path[strings.LastIndex(path, "/")+1:]
				}
			}
			if mostProductiveRepo != "" {
				return fmt.Sprintf("  🌟 Most active: %s", mostProductiveRepo)
			}
		}
	}

	return ""
}