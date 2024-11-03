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
	// Use pre-computed values from cache
	weeklyTotal := 0
	monthlyTotal := 0
	totalCommits := 0
	
	for _, repo := range cache.Cache {
		if !repo.Dormant {
			weeklyTotal += repo.WeeklyCommits
			monthlyTotal += repo.MonthlyCommits
			totalCommits += repo.CommitCount
		}
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
		// fmt.Printf("Debug - Cache entry for %s:\n", path)
		// fmt.Printf("  - Weekly commits: %d\n", repo.WeeklyCommits)
		// fmt.Printf("  - Commit history length: %d\n", len(repo.CommitHistory))
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
		
		// fmt.Printf("Debug - Repository: %s\n", repoName)
		// fmt.Printf("  - Weekly commits: %d\n", repo.metadata.WeeklyCommits)
		// fmt.Printf("  - Commit history length: %d\n", len(repo.metadata.CommitHistory))
		// fmt.Printf("  - Detailed stats enabled: %v\n", config.AppConfig.DetailedStats)
		
		activity := "⚡"
		if repo.metadata.WeeklyCommits > 10 {
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

		// Add detailed stats if enabled
		if config.AppConfig.DetailedStats && len(repo.metadata.CommitHistory) > 0 {
			// fmt.Printf("Debug - %s commit history: %d commits\n", repoName, len(repo.metadata.CommitHistory))
			var additions, deletions int
			for _, commit := range repo.metadata.CommitHistory {
				if time.Since(commit.Date) <= 7*24*time.Hour {
					additions += commit.Additions
					deletions += commit.Deletions
				}
			}
			if additions > 0 || deletions > 0 {
				summary += fmt.Sprintf(" • +%d/-%d lines", additions, deletions)
			}
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
	if !config.AppConfig.DisplayStats.ShowInsights {
		return ""
	}

	var insights []string
	
	// Basic insights (always shown)
	highestStreak := 0
	var streakChampRepo string
	
	for path, repo := range cache.Cache {
		if !repo.Dormant && repo.CurrentStreak > highestStreak {
			highestStreak = repo.CurrentStreak
			streakChampRepo = path
		}
	}
	
	if streakChampRepo != "" && highestStreak > 0 {
		insights = append(insights, fmt.Sprintf("💫 %s is your most active project with a %d day streak!",
			streakChampRepo[strings.LastIndex(streakChampRepo, "/")+1:],
			highestStreak))
	}

	// Detailed insights (only when detailed_stats is enabled)
	if config.AppConfig.DetailedStats {
		var totalAdditions, totalDeletions int
		for _, repo := range cache.Cache {
			for _, commit := range repo.CommitHistory {
				if time.Since(commit.Date) <= 7*24*time.Hour {
					totalAdditions += commit.Additions
					totalDeletions += commit.Deletions
				}
			}
		}
		
		if totalAdditions > 0 || totalDeletions > 0 {
			insights = append(insights, fmt.Sprintf("📝 This week: %d lines added, %d removed", 
				totalAdditions, totalDeletions))
		}
	}
	
	return strings.Join(insights, "\n")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}