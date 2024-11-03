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
	"github.com/olekukonko/tablewriter"
)

type repoInfo struct {
    name       string
    metadata   scan.RepoMetadata
    lastCommit time.Time
}

// DisplayStats - Displays stats for all active repositories in a more compact format
func DisplayStats() {
	// Create a test table to calculate width
	testBuf := new(bytes.Buffer)
	testTable := tablewriter.NewWriter(testBuf)
	testTable.SetHeader([]string{"Repository", "Weekly", "Streak", "Changes", "Activity"})
	
	// Use configured column widths
	cfg := config.AppConfig.DisplayStats.TableStyle.MinColumnWidths
	testTable.SetColMinWidth(0, cfg.Repository)
	testTable.SetColMinWidth(1, cfg.Weekly)
	testTable.SetColMinWidth(2, cfg.Streak)
	testTable.SetColMinWidth(3, cfg.Changes)
	testTable.SetColMinWidth(4, cfg.Activity)
	testTable.Render()
	
	// Get the width from the rendered test table
	tableWidth := len(strings.Split(testBuf.String(), "\n")[0])
	
	// Create styles with calculated width
	style := lipgloss.NewStyle()
	headerStyle := style.
		Bold(true).
		Foreground(lipgloss.Color(config.AppConfig.Colors.HeaderColor)).
		Width(tableWidth).
		Align(lipgloss.Center)
	
	dividerStyle := style.
		Foreground(lipgloss.Color(config.AppConfig.Colors.DividerColor))
	
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
	table := tablewriter.NewWriter(buf)

	// Configure table using config values
	cfg := config.AppConfig.DisplayStats.TableStyle
	table.SetHeader([]string{"Repository", "Weekly", "Streak", "Changes", "Activity"})
	table.SetBorder(cfg.ShowBorder)
	table.SetColumnSeparator(cfg.ColumnSeparator)
	table.SetCenterSeparator(cfg.CenterSeparator)
	table.SetHeaderAlignment(getAlignment(cfg.HeaderAlignment))
	table.SetHeaderLine(cfg.ShowHeaderLine)
	table.SetRowLine(cfg.ShowRowLines)
	
	// Set configured column widths
	table.SetColMinWidth(0, cfg.MinColumnWidths.Repository)
	table.SetColMinWidth(1, cfg.MinColumnWidths.Weekly)
	table.SetColMinWidth(2, cfg.MinColumnWidths.Streak)
	table.SetColMinWidth(3, cfg.MinColumnWidths.Changes)
	table.SetColMinWidth(4, cfg.MinColumnWidths.Activity)

	// All columns centered
	table.SetColumnAlignment([]int{
		tablewriter.ALIGN_CENTER,
		tablewriter.ALIGN_CENTER,
		tablewriter.ALIGN_CENTER,
			tablewriter.ALIGN_CENTER,
		tablewriter.ALIGN_CENTER,
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

		// Calculate weekly changes (always use detailed format)
		var weeklyAdditions, weeklyDeletions int
		weekStart := time.Now().AddDate(0, 0, -7)
		for _, commit := range meta.CommitHistory {
			if commit.Date.After(weekStart) {
				weeklyAdditions += commit.Additions
				weeklyDeletions += commit.Deletions
			}
		}
		changesStr := fmt.Sprintf("+%d/-%d", weeklyAdditions, weeklyDeletions)

		table.Append([]string{
			fmt.Sprintf("%s %s", activity, repo.name),
			fmt.Sprintf("%d↑", meta.WeeklyCommits),
				streakStr,
				changesStr,
				activityStr,
		})
	}

	table.Render()
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
		buf := new(bytes.Buffer)
		table := tablewriter.NewWriter(buf)
		table.SetBorder(false)
		table.SetColumnSeparator(" ")
		table.SetHeaderLine(false)
		table.SetRowLine(false)
		table.SetAutoWrapText(false)

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
			table.Append([]string{"📈", "Weekly Summary:", fmt.Sprintf("%d commits, +%d/-%d lines", 
				totalWeeklyCommits, totalAdditions, totalDeletions)})
		}
		
		if insights.ShowDailyAverage {
			table.Append([]string{"📊", "Daily Average:", 
				fmt.Sprintf("%.1f commits", float64(totalWeeklyCommits)/7.0)})
		}

		if insights.ShowTopLanguages && len(languageStats) > 0 {
			langs := formatLanguages(languageStats, insights.TopLanguagesCount)
			table.Append([]string{"💻", "Top Languages:", langs})
		}

		if insights.ShowPeakCoding {
			table.Append([]string{"⏰", "Peak Coding:", 
				fmt.Sprintf("%02d:00-%02d:00 (%d commits)", 
				peakHour, (peakHour+1)%24, peakCommits)})
		}

		if insights.ShowWeeklyGoal && config.AppConfig.GoalSettings.WeeklyCommitGoal > 0 {
			progress := float64(totalWeeklyCommits) / float64(config.AppConfig.GoalSettings.WeeklyCommitGoal) * 100
			table.Append([]string{"🎯", "Weekly Goal:", 
				fmt.Sprintf("%d%% (%d/%d commits)", 
				int(progress), totalWeeklyCommits, config.AppConfig.GoalSettings.WeeklyCommitGoal)})
		}

		table.Render()
		return buf.String()
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

// Helper function to convert string alignment to tablewriter constant
func getAlignment(alignment string) int {
	switch strings.ToLower(alignment) {
	case "left":
		return tablewriter.ALIGN_LEFT
	case "right":
		return tablewriter.ALIGN_RIGHT
	default:
		return tablewriter.ALIGN_CENTER
	}
}