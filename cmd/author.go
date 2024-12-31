package cmd

import (
	"fmt"
	"os/exec"
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

type AuthorStats struct {
	Name            string
	Email           string
	TotalCommits    int
	CurrentStreak   int
	LongestStreak   int
	WeeklyCommits   int
	MonthlyCommits  int
	TotalAdditions  int
	TotalDeletions  int
	TopRepositories []RepoActivity
	PeakHour        int
	PeakCommits     int
	Languages       map[string]int
}

type RepoActivity struct {
	Name       string
	Commits    int
	LastCommit time.Time
	Additions  int
	Deletions  int
	IsStarred  bool
	StarCount  int
}

// DisplayAuthorInfo shows detailed information about the specified author or the configured author
func DisplayAuthorInfo(targetAuthor string) {
	// If no target author is specified, use the configured author
	if targetAuthor == "" {
		targetAuthor = config.AppConfig.Author
	}

	// Get git configuration
	globalName, _ := exec.Command("git", "config", "--global", "user.name").Output()
	globalEmail, _ := exec.Command("git", "config", "--global", "user.email").Output()

	// Calculate author statistics
	stats := calculateAuthorStats(targetAuthor)
	stats.Name = strings.TrimSpace(string(globalName))
	stats.Email = strings.TrimSpace(string(globalEmail))

	// Display the information
	displayAuthorStats(stats)
}

func calculateAuthorStats(author string) AuthorStats {
	stats := AuthorStats{
		Languages: make(map[string]int),
	}

	repoActivities := make(map[string]*RepoActivity)
	now := time.Now()
	lookbackTime := now.AddDate(0, 0, -config.AppConfig.AuthorSettings.LookbackDays)
	weekAgo := now.AddDate(0, 0, -7)
	monthAgo := now.AddDate(0, 0, -30)

	// Debug output
	if config.AppConfig.Debug {
		fmt.Printf("Current time: %s\n", now.Format("2006-01-02"))
		fmt.Printf("Looking back to: %s (-%d days)\n", lookbackTime.Format("2006-01-02"), config.AppConfig.AuthorSettings.LookbackDays)
	}

	// Collect all commits across repositories
	var allCommits []struct {
		date      time.Time
		additions int
		deletions int
		repo      string
	}

	// Process each repository
	cache.Cache.Range(func(path string, repo scan.RepoMetadata) bool {
		repoName := path[strings.LastIndex(path, "/")+1:]
		activity := &RepoActivity{
			Name:       repoName,
			LastCommit: repo.LastCommit,
		}

		// Process commit history
		for _, commit := range repo.CommitHistory {
			if !strings.Contains(commit.Author, author) {
				continue
			}

			// Only process commits within lookback period
			if commit.Date.Before(lookbackTime) || commit.Date.After(now) {
				if config.AppConfig.Debug {
					fmt.Printf("Skipping commit from %s (outside lookback period)\n", commit.Date.Format("2006-01-02"))
				}
				continue
			}

			allCommits = append(allCommits, struct {
				date      time.Time
				additions int
				deletions int
				repo      string
			}{
				date:      commit.Date,
				additions: commit.Additions,
				deletions: commit.Deletions,
				repo:      repoName,
			})

			activity.Commits++
			activity.Additions += commit.Additions
			activity.Deletions += commit.Deletions
			stats.TotalCommits++
			stats.TotalAdditions += commit.Additions
			stats.TotalDeletions += commit.Deletions

			// Calculate weekly and monthly stats
			// Only count if within the lookback period
			if !commit.Date.Before(weekAgo) && !commit.Date.After(now) {
				stats.WeeklyCommits++
			}
			if !commit.Date.Before(monthAgo) && !commit.Date.After(now) {
				stats.MonthlyCommits++
			}

			// Track peak coding hour
			hour := commit.Date.Hour()
			commitCount := 1
			for _, c := range allCommits {
				if c.date.Hour() == hour {
					commitCount++
				}
			}
			if commitCount > stats.PeakCommits {
				stats.PeakHour = hour
				stats.PeakCommits = commitCount
			}
		}

		// Process languages if there are commits in the lookback period
		if activity.Commits > 0 {
			for lang, lines := range repo.Languages {
				stats.Languages[lang] += lines
			}
		}

		if activity.Commits > 0 {
			repoActivities[repoName] = activity
		}

		return true
	})

	// Debug output
	if config.AppConfig.Debug {
		fmt.Printf("Found %d commits in lookback period\n", len(allCommits))
		fmt.Printf("Weekly commits: %d\n", stats.WeeklyCommits)
		fmt.Printf("Monthly commits: %d\n", stats.MonthlyCommits)
	}

	// Sort commits by date in descending order (most recent first)
	sort.Slice(allCommits, func(i, j int) bool {
		return allCommits[i].date.After(allCommits[j].date)
	})

	// Calculate streaks
	if len(allCommits) > 0 {
		currentStreak := 0
		longestStreak := 0
		currentStreakStart := time.Now()
		lastDate := time.Now()

		// Check if there's a commit today to start the streak
		if time.Since(allCommits[0].date) < 24*time.Hour {
			currentStreak = 1
			currentStreakStart = allCommits[0].date
			lastDate = allCommits[0].date
		}

		// Process all commits for streaks
		for i := 1; i < len(allCommits); i++ {
			commitDate := allCommits[i].date
			dayDiff := lastDate.Sub(commitDate).Hours() / 24

			if dayDiff <= 1 { // Same day or consecutive days
				if currentStreak == 0 {
					currentStreak = 2
					currentStreakStart = lastDate
				} else {
					currentStreak++
				}
			} else if dayDiff > 1 {
				// Break in streak
				if currentStreak > longestStreak {
					longestStreak = currentStreak
				}
				currentStreak = 0
			}
			lastDate = commitDate
		}

		// Update final streak counts
		if currentStreak > longestStreak {
			longestStreak = currentStreak
		}

		// Only count current streak if it's active (includes today)
		if time.Since(currentStreakStart) > 24*time.Hour {
			currentStreak = 0
		}

		stats.CurrentStreak = currentStreak
		stats.LongestStreak = longestStreak
	}

	// Convert map to slice and sort by activity
	for _, activity := range repoActivities {
		stats.TopRepositories = append(stats.TopRepositories, *activity)
	}

	// Sort repositories by commit count
	sort.Slice(stats.TopRepositories, func(i, j int) bool {
		return stats.TopRepositories[i].Commits > stats.TopRepositories[j].Commits
	})

	// Limit to configured number of top repositories
	maxRepos := config.AppConfig.AuthorSettings.MaxTopRepos
	if len(stats.TopRepositories) > maxRepos {
		stats.TopRepositories = stats.TopRepositories[:maxRepos]
	}

	return stats
}

func displayAuthorStats(stats AuthorStats) {
	// Get terminal width for table sizing
	width := getTerminalWidth()

	// Create header style
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(config.AppConfig.Colors.HeaderColor))

	// Create main info table
	t := table.NewWriter()
	t.SetStyle(getAuthorTableStyle())
	t.AppendRow(table.Row{"📧", "Email", stats.Email})
	t.AppendRow(table.Row{"📊", "Total Commits", fmt.Sprintf("%d (last %d days)", stats.TotalCommits, config.AppConfig.AuthorSettings.LookbackDays)})

	// Format streak with appropriate emoji
	streakEmoji := config.AppConfig.DisplayStats.ActivityIndicators.ActiveStreak
	if stats.CurrentStreak == 0 {
		streakEmoji = config.AppConfig.DisplayStats.ActivityIndicators.NoActivity
	} else if stats.CurrentStreak >= 7 {
		streakEmoji = config.AppConfig.DisplayStats.ActivityIndicators.HighActivity
	}
	t.AppendRow(table.Row{streakEmoji, "Current Streak", fmt.Sprintf("%d days", stats.CurrentStreak)})

	// Format longest streak with trophy if it's the current record
	streakSuffix := ""
	if stats.CurrentStreak == stats.LongestStreak && stats.CurrentStreak > 0 {
		streakSuffix = " " + config.AppConfig.DisplayStats.ActivityIndicators.StreakRecord
	}
	t.AppendRow(table.Row{"⭐", "Longest Streak", fmt.Sprintf("%d days%s", stats.LongestStreak, streakSuffix)})

	// Format activity with appropriate emoji based on commit count
	activityEmoji := config.AppConfig.DisplayStats.ActivityIndicators.NormalActivity
	if stats.WeeklyCommits >= config.AppConfig.DisplayStats.Thresholds.HighActivity {
		activityEmoji = config.AppConfig.DisplayStats.ActivityIndicators.HighActivity
	} else if stats.WeeklyCommits == 0 {
		activityEmoji = config.AppConfig.DisplayStats.ActivityIndicators.NoActivity
	}
	t.AppendRow(table.Row{activityEmoji, "Weekly Activity", fmt.Sprintf("%d commits", stats.WeeklyCommits)})
	t.AppendRow(table.Row{"📅", "Monthly Activity", fmt.Sprintf("%d commits", stats.MonthlyCommits)})
	t.AppendRow(table.Row{"⚡", "Code Changes", fmt.Sprintf("+%d/-%d lines", stats.TotalAdditions, stats.TotalDeletions)})
	t.AppendRow(table.Row{"⏰", "Peak Coding Hour", fmt.Sprintf("%02d:00-%02d:00 (%d commits)",
		stats.PeakHour, (stats.PeakHour+1)%24, stats.PeakCommits)})

	// Set table width and render
	t.SetAllowedRowLength(width - 4)
	tableStr := t.Render()
	tableWidth := getTableWidth(tableStr)

	// Print header centered above the table
	header := fmt.Sprintf("🧑‍💻 %s's Coding Activity", stats.Name)
	fmt.Println(centerText(header, tableWidth))
	fmt.Println(tableStr)
	fmt.Println()

	// Display top repositories
	if len(stats.TopRepositories) > 0 {
		t = table.NewWriter()
		t.SetStyle(getAuthorTableStyle())
		t.SetAllowedRowLength(width - 4)

		// Add Table Header if Set in config
		if config.AppConfig.DisplayStats.TableStyle.UseTableHeader {
			t.AppendHeader(table.Row{"Repository", "Commits", "Changes", "Last Activity"})
		}

		for _, repo := range stats.TopRepositories {
			t.AppendRow(table.Row{
				repo.Name,
				fmt.Sprintf("%d", repo.Commits),
				fmt.Sprintf("+%d/-%d", repo.Additions, repo.Deletions),
				formatAuthorLastActivity(repo.LastCommit),
			})
		}

		tableStr = t.Render()
		tableWidth = getTableWidth(tableStr)
		fmt.Println(headerStyle.Render(centerText("📚 Top Repositories", tableWidth)))
		fmt.Println(tableStr)
		fmt.Println()
	}

	// Display language statistics
	if len(stats.Languages) > 0 {
		langStr := formatLanguages(stats.Languages, config.AppConfig.DisplayStats.InsightSettings.TopLanguagesCount)
		langWidth := getTableWidth(langStr)
		fmt.Println(headerStyle.Render(centerText("💻 Language Distribution", langWidth)))
		fmt.Println(langStr)
	}
}

func getAuthorTableStyle() table.Style {
	// Use the configured table style
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

	// Apply configured options
	style.Options.DrawBorder = config.AppConfig.DisplayStats.TableStyle.Options.DrawBorder
	style.Options.SeparateColumns = config.AppConfig.DisplayStats.TableStyle.Options.SeparateColumns
	style.Options.SeparateHeader = config.AppConfig.DisplayStats.TableStyle.Options.SeparateHeader
	style.Options.SeparateRows = config.AppConfig.DisplayStats.TableStyle.Options.SeparateRows

	return style
}

func centerText(text string, width int) string {
	textWidth := len([]rune(text))
	if textWidth >= width {
		return text
	}
	leftPadding := (width - textWidth) / 2
	rightPadding := width - textWidth - leftPadding
	return strings.Repeat(" ", leftPadding) + text + strings.Repeat(" ", rightPadding)
}

func getTerminalWidth() int {
	width, _, err := term.GetSize(0)
	if err != nil {
		return 80
	}
	return width
}

func formatAuthorLastActivity(lastCommit time.Time) string {
	duration := time.Since(lastCommit)
	switch {
	case duration < 24*time.Hour:
		return "today"
	case duration < 48*time.Hour:
		return "yesterday"
	case duration < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(duration.Hours()/24))
	default:
		return lastCommit.Format("2006-01-02")
	}
}

// getTableWidth returns the width of the widest line in a rendered table
func getTableWidth(tableStr string) int {
	lines := strings.Split(tableStr, "\n")
	maxWidth := 0
	for _, line := range lines {
		width := len([]rune(line))
		if width > maxWidth {
			maxWidth = width
		}
	}
	return maxWidth
}
