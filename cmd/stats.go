package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/AccursedGalaxy/streakode/cache"
	"github.com/AccursedGalaxy/streakode/config"
	"github.com/AccursedGalaxy/streakode/scan"
	"github.com/charmbracelet/lipgloss"
)

// DisplayStats - Displays stats for all active repositories in a more compact format
func DisplayStats() {
	// Create styles
	style := lipgloss.NewStyle()
	headerStyle := style.
		Bold(true).
		Foreground(lipgloss.Color(config.AppConfig.Colors.HeaderColor))
	
	sectionStyle := style.
		Foreground(lipgloss.Color(config.AppConfig.Colors.SectionColor)).
		PaddingLeft(2)
	
	dividerStyle := style.
		Foreground(lipgloss.Color(config.AppConfig.Colors.DividerColor))
	
	// Track which sections are active
	activeSections := 0

	// Build sections dynamically
	var sections []string

	// Header section (shown if welcome message enabled)
	if config.AppConfig.DisplayStats.ShowWelcomeMessage {
		header := headerStyle.Render(fmt.Sprintf("🚀 %s's Coding Activity", config.AppConfig.Author))
		sections = append(sections, header)
	}

	// Stats section
	if hasAnyStatsEnabled() {
		activeSections++
		stats := buildStatsSection()
		if stats != "" {
			sections = append(sections, sectionStyle.Render(stats))
		}
	}

	// Active projects section
	if config.AppConfig.DisplayStats.ShowActiveProjects {
		activeSections++
		projects := buildProjectsSection()
		if projects != "" {
			sections = append(sections, sectionStyle.Render(projects))
		}
	}

	// Calculate highest streak from cache
	highestStreak := 0
	for _, repo := range cache.Cache {
		if repo.CurrentStreak > highestStreak {
			highestStreak = repo.CurrentStreak
		}
	}

	// Insights section
	if config.AppConfig.DisplayStats.ShowInsights && highestStreak > 0 {
		activeSections++
		insights := buildInsightsSection()
		if insights != "" {
			sections = append(sections, sectionStyle.Render(insights))
		}
	}

	// Only show dividers if we have multiple sections
	divider := dividerStyle.Render("──────────────────────────────────────")
	
	// Join sections with dividers only if they're not empty
	output := ""
	for i, section := range sections {
		if section == "" {
			continue
		}
		output += section
		if i < len(sections)-1 && sections[i+1] != "" {
			output += "\n" + divider + "\n"
		}
	}

	fmt.Println(output)
}

// Helper functions to build each section
func hasAnyStatsEnabled() bool {
	return config.AppConfig.DisplayStats.ShowWeeklyCommits ||
		config.AppConfig.DisplayStats.ShowMonthlyCommits ||
		config.AppConfig.DisplayStats.ShowTotalCommits
}

func buildStatsSection() string {
	weeklyTotal := 0
	monthlyTotal := 0
	totalCommits := 0
	
	// Sum up the commits from each repo in the cache
	for _, repo := range cache.Cache {
		weeklyTotal += repo.WeeklyCommits
		monthlyTotal += repo.MonthlyCommits
		totalCommits += repo.CommitCount
	}
	
	stats := []string{}
	if config.AppConfig.DisplayStats.ShowWeeklyCommits {
		stats = append(stats, fmt.Sprintf("%d commits this week", weeklyTotal))
	}
	if config.AppConfig.DisplayStats.ShowMonthlyCommits {
		stats = append(stats, fmt.Sprintf("%d this month", monthlyTotal))
	}
	if config.AppConfig.DisplayStats.ShowTotalCommits {
		stats = append(stats, fmt.Sprintf("%d total", totalCommits))
	}
	if len(stats) > 0 {
		return fmt.Sprintf("📊 %s", strings.Join(stats, " • "))
	}
	return ""
}

func buildProjectsSection() string {
	if !config.AppConfig.DisplayStats.ShowActiveProjects {
		return ""
	}

	totalProjects := len(cache.Cache)
	maxDisplay := config.AppConfig.DisplayStats.MaxProjects
	if maxDisplay <= 0 {
		maxDisplay = 5
	}
	displayCount := min(totalProjects, maxDisplay)
	
	// Convert map to slice for sorting
	type repoInfo struct {
		name       string
		metadata   scan.RepoMetadata
		lastCommit time.Time
	}
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

	// Build summaries for each repo
	var summaries []string
	for i := 0; i < displayCount && i < len(repos); i++ {
		repo := repos[i]
		repoName := repo.name
		
		activity := "⚡"
		if repo.metadata.WeeklyCommits > 5 {
			activity = "🔥"
		} else if repo.metadata.WeeklyCommits == 0 {
			activity = "💤"
		}

		summary := fmt.Sprintf("%s %s: %d↑ this week",
			activity,
			repoName[:min(len(repoName), 15)],
			repo.metadata.WeeklyCommits)

		if repo.metadata.CurrentStreak > 0 {
			summary += fmt.Sprintf(" • 🔥 %d day streak", repo.metadata.CurrentStreak)
		}

		daysAgo := time.Since(repo.lastCommit).Hours() / 24
		if daysAgo < 1 {
			summary += " • today"
		} else if daysAgo < 2 {
			summary += " • yesterday"
		} else {
			summary += fmt.Sprintf(" • %d days ago", int(daysAgo))
		}
		
		summaries = append(summaries, summary)
	}

	return strings.Join(summaries, "\n")
}

func buildInsightsSection() string {
	highestStreak := 0
	var streakChampRepo string
	
	for path, repo := range cache.Cache {
		if repo.CurrentStreak > highestStreak {
			highestStreak = repo.CurrentStreak
			streakChampRepo = path
		}
	}
	
	if streakChampRepo == "" || highestStreak == 0 {
		return ""
	}
	
	return fmt.Sprintf("💫 %s is your most active project with a %d day streak!",
		streakChampRepo[strings.LastIndex(streakChampRepo, "/")+1:],
		highestStreak)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}