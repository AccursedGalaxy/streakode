package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/AccursedGalaxy/streakode/cache"
	"github.com/AccursedGalaxy/streakode/config"
	"github.com/charmbracelet/lipgloss"
	"github.com/jedib0t/go-pretty/v6/table"
	"golang.org/x/term"
)

type HistoryOptions struct {
	Author     string
	Repository string
	Days       int
	Detailed   bool
}

type CommitSummary struct {
	Hash        string
	Date        time.Time
	Message     string
	FileCount   int
	Additions   int
	Deletions   int
	TotalLines  int
	FilesChanged []string
}

func DisplayHistory(opts HistoryOptions) {
	// Get table width for formatting
	tableWidth := calculateTableWidth()
	style := lipgloss.NewStyle()
	headerStyle := style.
		Bold(true).
		Foreground(lipgloss.Color(config.AppConfig.Colors.HeaderColor)).
		Width(tableWidth).
		Align(lipgloss.Center)

	// Build header based on options
	var headerText string
	if opts.Author != "" {
		headerText = fmt.Sprintf("📚 Git History for %s", opts.Author)
	} else if opts.Repository != "" {
		headerText = fmt.Sprintf("📚 Git History for %s", opts.Repository)
	} else {
		headerText = "📚 Git History"
	}

	// Print header
	fmt.Println(headerStyle.Render(headerText))
	fmt.Println()

	// Get commit history
	commits := getCommitHistory(opts)
	if len(commits) == 0 {
		fmt.Println("No commits found for the specified criteria.")
		return
	}

	// Display commit history
	displayCommitHistory(commits, opts.Detailed, tableWidth)
}

func getCommitHistory(opts HistoryOptions) []CommitSummary {
	var commits []CommitSummary
	since := time.Now().AddDate(0, 0, -opts.Days)

	// Filter repositories based on options
	for path, repo := range cache.Cache {
		if opts.Repository != "" && !matchesRepository(path, opts.Repository) {
			continue
		}

		for _, commit := range repo.CommitHistory {
			if commit.Date.Before(since) {
				continue
			}

			if opts.Author != "" && !matchesAuthor(opts.Author) {
				continue
			}

			commits = append(commits, CommitSummary{
				Hash:        commit.Hash,
				Date:        commit.Date,
				Message:     commit.MessageHead,
				FileCount:   commit.FileCount,
				Additions:   commit.Additions,
				Deletions:   commit.Deletions,
				TotalLines:  commit.Additions + commit.Deletions,
			})
		}
	}

	// Sort commits by date (most recent first)
	sortCommitsByDate(commits)
	return commits
}

func displayCommitHistory(commits []CommitSummary, detailed bool, tableWidth int) {
	t := table.NewWriter()
	t.SetAllowedRowLength(tableWidth)

	// Configure table style based on config
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

	// Set up columns
	if detailed {
		t.AppendHeader(table.Row{
			"Date",
			"Hash",
			"Message",
			"Files",
			"Changes",
		})
	} else {
		t.AppendHeader(table.Row{
			"Date",
			"Message",
			"Changes",
		})
	}

	// Add commit rows
	for _, commit := range commits {
		if detailed {
			t.AppendRow(table.Row{
				commit.Date.Format("2006-01-02 15:04"),
				commit.Hash[:8],
				commit.Message,
				commit.FileCount,
				fmt.Sprintf("+%d/-%d", commit.Additions, commit.Deletions),
			})
		} else {
			t.AppendRow(table.Row{
				commit.Date.Format("2006-01-02"),
				commit.Message,
				fmt.Sprintf("+%d/-%d", commit.Additions, commit.Deletions),
			})
		}
	}

	fmt.Println(t.Render())
}

func calculateTableWidth() int {
	width, _, err := term.GetSize(0)
	if err != nil {
		return 80 // default width
	}
	return min(width-2, 120)
}

func matchesRepository(path, targetRepo string) bool {
	return strings.HasSuffix(path, "/"+targetRepo)
}

func matchesAuthor(author string) bool {
	return config.AppConfig.Author == author
}

func sortCommitsByDate(commits []CommitSummary) {
	sort.Slice(commits, func(i, j int) bool {
		return commits[i].Date.After(commits[j].Date)
	})
} 