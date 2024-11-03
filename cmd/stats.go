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
	testTable.SetColMinWidth(0, 20)  // Repository
	testTable.SetColMinWidth(1, 8)   // Weekly
	testTable.SetColMinWidth(2, 8)   // Streak
	testTable.SetColMinWidth(3, 13)  // Changes
	testTable.SetColMinWidth(4, 10)  // Activity
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

	// Configure table style
	table.SetHeader([]string{"Repository", "Weekly", "Streak", "Changes", "Activity"})
	table.SetBorder(false)
	table.SetColumnSeparator("│")
	table.SetCenterSeparator("┼")
	table.SetHeaderAlignment(tablewriter.ALIGN_CENTER)
	table.SetAlignment(tablewriter.ALIGN_CENTER)
	table.SetHeaderLine(true)
	table.SetRowLine(false)
	
	// Set minimum column widths
	table.SetColMinWidth(0, 20)  // Repository
	table.SetColMinWidth(1, 8)   // Weekly
	table.SetColMinWidth(2, 8)   // Streak
	table.SetColMinWidth(3, 13)  // Changes
	table.SetColMinWidth(4, 10)  // Activity
	
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

		// Determine activity icon
		activity := "⚡"
		if meta.WeeklyCommits > 10 {
			activity = "🔥"
		} else if meta.WeeklyCommits == 0 {
			activity = "💤"
		}

		// Format streak
		streakStr := fmt.Sprintf("%dd", meta.CurrentStreak)
		if meta.CurrentStreak == meta.LongestStreak && meta.CurrentStreak > 0 {
			streakStr += "🏆"
		} else if meta.CurrentStreak > 0 {
			streakStr += "🔥"
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

func formatLanguages(stats map[string]int) string {
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
	for i := 0; i < min(len(langs), 3); i++ {
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

		// Add rows to table
		table.Append([]string{"📈", "Weekly Summary:", fmt.Sprintf("%d commits, +%d/-%d lines", 
			totalWeeklyCommits, totalAdditions, totalDeletions)})
		
		table.Append([]string{"📊", "Daily Average:", 
			fmt.Sprintf("%.1f commits", float64(totalWeeklyCommits)/7.0)})

		if len(languageStats) > 0 {
			langs := formatLanguages(languageStats)
			table.Append([]string{"💻", "Top Languages:", langs})
		}

		table.Append([]string{"⏰", "Peak Coding:", 
			fmt.Sprintf("%02d:00-%02d:00 (%d commits)", 
			peakHour, (peakHour+1)%24, peakCommits)})

		if config.AppConfig.GoalSettings.WeeklyCommitGoal > 0 {
			progress := float64(totalWeeklyCommits) / float64(config.AppConfig.GoalSettings.WeeklyCommitGoal) * 100
			table.Append([]string{"🎯", "Weekly Goal:", 
				fmt.Sprintf("%d%% (%d/%d commits)", 
				int(progress), totalWeeklyCommits, config.AppConfig.GoalSettings.WeeklyCommitGoal)})
		}

		table.Render()
		return buf.String()
	} else {
		// Simple insights for non-detailed view
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

	return ""
}