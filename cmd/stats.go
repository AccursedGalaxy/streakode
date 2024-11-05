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

// DisplayStats - Displays stats for all active repositories in a more compact format
func DisplayStats() {
	// Get table width from the rendered table first
	projectsSection := buildProjectsSection()
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
		insights := buildInsightsSection()
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

	// Add Table Header if Set in config
	if config.AppConfig.DisplayStats.TableStyle.UseTableHeader{
		t.AppendHeader(table.Row{
			"Repo",
			"Weekly",
			"Streak",
			"Changes",
			"Activity",
		})
	}

	// Simplify width calculations
	width, _, err := term.GetSize(0)
	if err != nil {
		width = 80
	}
	tableWidth := min(width-2, 120)

	// Configure table with simpler column ratios
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, WidthMax: int(float64(tableWidth) * 0.35)}, // Repository name
		{Number: 2, WidthMax: int(float64(tableWidth) * 0.15)}, // Weekly commits
		{Number: 3, WidthMax: int(float64(tableWidth) * 0.15)}, // Streak
		{Number: 4, WidthMax: int(float64(tableWidth) * 0.20)}, // Changes
		{Number: 5, WidthMax: int(float64(tableWidth) * 0.15)}, // Last activity
	})

	// Set overall table width
	t.SetAllowedRowLength(tableWidth)

	// Use simpler style configuration
	style := table.Style{
		Box: table.BoxStyle{
			BottomLeft:       "└",
			BottomRight:      "┘",
			BottomSeparator:  "┴",
			Left:            "│",
			LeftSeparator:    "├",
			MiddleHorizontal: "─",
			MiddleSeparator:  "┼",
			MiddleVertical:   "│",
			PaddingLeft:      " ",
			PaddingRight:     " ",
			Right:           "│",
			RightSeparator:   "┤",
			TopLeft:         "┌",
			TopRight:        "┐",
			TopSeparator:    "┬",
		},
		Options: table.Options{
			DrawBorder:      config.AppConfig.DisplayStats.TableStyle.Options.DrawBorder,
			SeparateColumns: config.AppConfig.DisplayStats.TableStyle.Options.SeparateColumns,
			SeparateHeader:  config.AppConfig.DisplayStats.TableStyle.Options.SeparateHeader,
			SeparateRows:    config.AppConfig.DisplayStats.TableStyle.Options.SeparateRows,
		},
	}
	t.SetStyle(style)

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

	// switch statement to check for user style setting in config
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

	// Render to buffer and return
	t.Render()
	return buf.String()
}

// TODO: Make Language Output Format More Appealing
func formatLanguages(stats map[string]int, topCount int) string {
	// Language icons mapping with more descriptive emojis
	languageIcons := map[string]string{
		"go":		config.AppConfig.LanguageSettings.LanguageDisplay.GoDisplay,
		"py":		config.AppConfig.LanguageSettings.LanguageDisplay.PythonDisplay,
		"lua":		config.AppConfig.LanguageSettings.LanguageDisplay.LuaDisplay,
		"js":		config.AppConfig.LanguageSettings.LanguageDisplay.JavaDisplay,
		"ts":		config.AppConfig.LanguageSettings.LanguageDisplay.TypeScriptDisplay,
		"rust":		config.AppConfig.LanguageSettings.LanguageDisplay.RustDisplay,
		"cpp":		config.AppConfig.LanguageSettings.LanguageDisplay.CppDisplay,
		"c":		config.AppConfig.LanguageSettings.LanguageDisplay.CDisplay,
		"java":		config.AppConfig.LanguageSettings.LanguageDisplay.JavaDisplay,
		"ruby":		config.AppConfig.LanguageSettings.LanguageDisplay.RubyDisplay,
		"php":		config.AppConfig.LanguageSettings.LanguageDisplay.PHPDisplay,
		"html":		config.AppConfig.LanguageSettings.LanguageDisplay.HTMLDisplay,
		"css":		config.AppConfig.LanguageSettings.LanguageDisplay.CSSDisplay,
		"shell":	config.AppConfig.LanguageSettings.LanguageDisplay.ShellDisplay,
		"default":	config.AppConfig.LanguageSettings.LanguageDisplay.DefaultDisplay,
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
	
	// Format languages with icons and better number formatting
	var formatted []string
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
				sizeStr = fmt.Sprintf("%.1fM LOC", float64(langs[i].lines)/1000000)
			case langs[i].lines >= 1000:
				sizeStr = fmt.Sprintf("%.1fK LOC", float64(langs[i].lines)/1000)
			default:
				sizeStr = fmt.Sprintf("%d LOC", langs[i].lines)
			}
			
			// Format with icon, language, and size
			formatted = append(formatted, fmt.Sprintf("%s (%s)", 
				icon, sizeStr))
		}
	}
	
	return strings.Join(formatted, "  ")
}

func buildInsightsSection() string {
	if !config.AppConfig.DisplayStats.ShowInsights {
		return ""
	}

	// Get the same terminal width as used elsewhere
	width, _, err := term.GetSize(0)
	if err != nil {
		width = 80
	}
	tableWidth := min(width-2, 120)

	insights := config.AppConfig.DisplayStats.InsightSettings
	
	if config.AppConfig.DetailedStats {
		t := table.NewWriter()
		t.SetStyle(table.Style{
			Options: table.Options{
				DrawBorder:      config.AppConfig.DisplayStats.TableStyle.Options.DrawBorder,
				SeparateColumns: config.AppConfig.DisplayStats.TableStyle.Options.SeparateColumns,
				SeparateHeader:  config.AppConfig.DisplayStats.TableStyle.Options.SeparateHeader,
				SeparateRows:    config.AppConfig.DisplayStats.TableStyle.Options.SeparateRows,
			},
			Box: table.BoxStyle{
				PaddingLeft:      "",
				PaddingRight:     " ",
				MiddleVertical:   "",
			},
		})

		// Set max width for the entire table
		t.SetAllowedRowLength(tableWidth-2)

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

		// TODO: Show Comparison to last week. (Up or Down & By How Much)
		if insights.ShowWeeklySummary {
			t.AppendRow(table.Row{"📈", "Weekly Summary:", 
				fmt.Sprintf("%d commits, +%d/-%d lines", 
					totalWeeklyCommits, totalAdditions, totalDeletions)})
		}
	
		// TODO: Show Comparison to last weeks daily average. (Up or Down & By How Much)
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
