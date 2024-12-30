package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/AccursedGalaxy/streakode/cache"
	"github.com/AccursedGalaxy/streakode/cmd/search"
	"github.com/AccursedGalaxy/streakode/config"
	"github.com/AccursedGalaxy/streakode/scan"
	"github.com/charmbracelet/lipgloss"
	"github.com/jedib0t/go-pretty/v6/table"
	"golang.org/x/term"
)

type HistoryOptions struct {
	Author      string
	Repository  string
	Days        int
	Detailed    bool
	Interactive bool
	Preview     bool
	Format      string
	Branch      string
	Query       string // Search query for filtering commits
}

type CommitSummary struct {
	Hash         string
	Date         time.Time
	Message      string
	FileCount    int
	Additions    int
	Deletions    int
	TotalLines   int
	FilesChanged []string
	Branch       string
	Repository   string
	Author       string
}

// DisplayHistory is the main entry point for the history command
func DisplayHistory(opts HistoryOptions) {
	// Always use interactive mode with preview by default
	opts.Interactive = true
	if !opts.Preview {
		opts.Preview = true
	}

	// Get commit history
	commits := getCommitHistory(opts)
	if len(commits) == 0 {
		fmt.Println("No commits found for the specified criteria.")
		return
	}

	displayInteractiveHistory(commits, opts)
}

func displayInteractiveHistory(commits []CommitSummary, opts HistoryOptions) {
	// Convert CommitSummary to SearchResult
	var searchResults []search.SearchResult
	for _, commit := range commits {
		searchResults = append(searchResults, search.SearchResult{
			Hash:       commit.Hash,
			Date:       commit.Date,
			Message:    commit.Message,
			FileCount:  commit.FileCount,
			Additions:  commit.Additions,
			Deletions:  commit.Deletions,
			Repository: commit.Repository,
			Branch:     commit.Branch,
		})
	}

	// Configure search options
	searchOpts := search.SearchOptions{
		Preview:     opts.Preview,
		DetailLevel: boolToInt(opts.Detailed),
		Repository:  opts.Repository,
		Author:      opts.Author,
		Interactive: true,
		Format:      opts.Format,
	}

	// Run interactive search
	selected, err := search.RunInteractiveSearch(searchResults, searchOpts)
	if err != nil {
		if err.Error() == "fzf is not installed" {
			fmt.Println("Interactive search requires fzf. Falling back to table view.")
			displayTableHistory(commits, opts)
			return
		}
		fmt.Printf("Error during interactive search: %v\n", err)
		return
	}

	// Handle selected commits if any
	if len(selected) > 0 {
		handleSelectedCommits(selected)
	}
}

func handleSelectedCommits(commits []search.SearchResult) {
	fmt.Printf("\nSelected %d commits:\n\n", len(commits))

	t := table.NewWriter()
	t.SetStyle(table.StyleRounded)
	t.AppendHeader(table.Row{"Date", "Repository", "Message", "Changes"})

	for _, commit := range commits {
		t.AppendRow(table.Row{
			commit.Date.Format("2006-01-02 15:04"),
			commit.Repository,
			commit.Message,
			fmt.Sprintf("+%d/-%d", commit.Additions, commit.Deletions),
		})
	}

	fmt.Println(t.Render())
}

func getCommitHistory(opts HistoryOptions) []CommitSummary {
	var commits []CommitSummary
	since := time.Now().AddDate(0, 0, -opts.Days)

	// Use channels for concurrent processing
	type repoResult struct {
		commits []CommitSummary
		err     error
	}
	results := make(chan repoResult)

	// Process repositories concurrently
	activeRepos := 0
	cache.Cache.Range(func(path string, repo scan.RepoMetadata) bool {
		if opts.Repository != "" && !matchesRepository(path, opts.Repository) {
			return true
		}
		activeRepos++

		go func(repoPath string) {
			repoName := extractRepoName(repoPath)
			var result repoResult

			// Only fetch if we need remote data and haven't fetched recently
			if shouldFetchRemote(repoPath) {
				fetchRemoteData(repoPath)
			}

			// Get local commits first (faster)
			localCommits := getLocalCommits(repoPath, opts, since)
			result.commits = localCommits

			// Only fetch remote commits if local commits don't satisfy our needs
			if len(localCommits) < 100 { // Arbitrary limit to avoid fetching too much
				remoteCommits := getRemoteCommits(repoPath, opts, since)
				result.commits = mergeCommits(localCommits, remoteCommits, repoName)
			} else {
				// Just set repository name for local commits
				for i := range result.commits {
					result.commits[i].Repository = repoName
				}
			}

			results <- result
		}(path)

		return true
	})

	// Collect results
	for i := 0; i < activeRepos; i++ {
		result := <-results
		if result.err != nil {
			if config.AppConfig.Debug {
				fmt.Printf("Error processing repository: %v\n", result.err)
			}
			continue
		}
		commits = append(commits, result.commits...)
	}

	// Sort commits by date (most recent first)
	sortCommitsByDate(commits)
	return commits
}

func shouldFetchRemote(repoPath string) bool {
	// Check if repo has a remote
	cmd := exec.Command("git", "-C", repoPath, "remote")
	if output, err := cmd.Output(); err != nil || len(output) == 0 {
		return false
	}

	// Check last fetch time
	lastFetchFile := filepath.Join(repoPath, ".git", "FETCH_HEAD")
	info, err := os.Stat(lastFetchFile)
	if err != nil {
		return true // No fetch record, should fetch
	}

	// Fetch if last fetch was more than 15 minutes ago
	return time.Since(info.ModTime()) > 15*time.Minute
}

func fetchRemoteData(repoPath string) {
	if config.AppConfig.Debug {
		fmt.Printf("Fetching remote data for %s\n", repoPath)
	}

	// Fetch all branches and tags
	cmd := exec.Command("git", "-C", repoPath, "fetch", "--all", "--tags", "--force", "--quiet")
	cmd.Run() // Ignore errors, we'll work with what we have
}

func getLocalCommits(repoPath string, opts HistoryOptions, since time.Time) []CommitSummary {
	var commits []CommitSummary

	// Build optimized git log command
	args := []string{
		"-C", repoPath,
		"log",
		"--no-merges", // Skip merge commits for cleaner history
		"--date-order",
		"--pretty=format:%H%x00%aI%x00%an%x00%ae%x00%s%x00%b%x00",
		"--numstat",
		"--after=" + since.Format("2006-01-02"),
	}

	if opts.Author != "" {
		args = append(args, "--author="+opts.Author)
	}

	// Add branch filtering if specified
	if opts.Branch != "" {
		args = append(args, opts.Branch)
	} else {
		args = append(args, "--all")
	}

	// Execute command with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	output, err := cmd.Output()
	if err != nil {
		if config.AppConfig.Debug {
			fmt.Printf("Error getting local commits from %s: %v\n", repoPath, err)
		}
		return commits
	}

	return parseGitLog(string(output))
}

func getRemoteCommits(repoPath string, opts HistoryOptions, since time.Time) []CommitSummary {
	var commits []CommitSummary

	// Get remote branches efficiently
	cmd := exec.Command("git", "-C", repoPath, "for-each-ref", "--format=%(refname)", "refs/remotes/origin")
	output, err := cmd.Output()
	if err != nil {
		return commits
	}

	branches := strings.Split(string(output), "\n")

	// Process each branch concurrently
	type branchResult struct {
		commits []CommitSummary
		err     error
	}
	results := make(chan branchResult, len(branches))

	for _, branch := range branches {
		if branch == "" {
			continue
		}

		go func(branchName string) {
			var result branchResult

			// Get commits from remote branch with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			args := []string{
				"-C", repoPath,
				"log",
				"--no-merges",
				"--date-order",
				branchName,
				"--pretty=format:%H%x00%aI%x00%an%x00%ae%x00%s%x00%b%x00",
				"--numstat",
				"--after=" + since.Format("2006-01-02"),
			}

			if opts.Author != "" {
				args = append(args, "--author="+opts.Author)
			}

			cmd := exec.CommandContext(ctx, "git", args...)
			output, err := cmd.Output()
			if err != nil {
				result.err = err
				results <- result
				return
			}

			result.commits = parseGitLog(string(output))
			results <- result
		}(branch)
	}

	// Collect results with timeout
	timeout := time.After(10 * time.Second)
	for range branches {
		select {
		case result := <-results:
			if result.err == nil {
				commits = append(commits, result.commits...)
			}
		case <-timeout:
			if config.AppConfig.Debug {
				fmt.Printf("Timeout while getting remote commits from %s\n", repoPath)
			}
			return commits
		}
	}

	return commits
}

func parseGitLog(output string) []CommitSummary {
	var commits []CommitSummary
	entries := strings.Split(output, "\n\n")

	for _, entry := range entries {
		if entry == "" {
			continue
		}

		lines := strings.Split(entry, "\n")
		if len(lines) == 0 {
			continue
		}

		// Parse commit info
		parts := strings.Split(lines[0], "\x00")
		if len(parts) < 6 {
			continue
		}

		hash := parts[0]
		date, _ := time.Parse(time.RFC3339, parts[1])
		author := parts[2]
		email := parts[3]
		subject := parts[4]
		body := parts[5]

		// Parse stats
		var additions, deletions, fileCount int
		for _, line := range lines[1:] {
			if line == "" {
				continue
			}
			statParts := strings.Fields(line)
			if len(statParts) < 3 {
				continue
			}
			add, _ := strconv.Atoi(statParts[0])
			del, _ := strconv.Atoi(statParts[1])
			additions += add
			deletions += del
			fileCount++
		}

		commits = append(commits, CommitSummary{
			Hash:       hash,
			Date:       date,
			Message:    subject + "\n\n" + body,
			FileCount:  fileCount,
			Additions:  additions,
			Deletions:  deletions,
			TotalLines: additions + deletions,
			Author:     fmt.Sprintf("%s <%s>", author, email),
		})
	}

	return commits
}

func mergeCommits(local, remote []CommitSummary, repoName string) []CommitSummary {
	seen := make(map[string]bool)
	var merged []CommitSummary

	// Add all local commits
	for _, commit := range local {
		if !seen[commit.Hash] {
			commit.Repository = repoName
			merged = append(merged, commit)
			seen[commit.Hash] = true
		}
	}

	// Add remote commits that aren't in local
	for _, commit := range remote {
		if !seen[commit.Hash] {
			commit.Repository = repoName
			merged = append(merged, commit)
			seen[commit.Hash] = true
		}
	}

	return merged
}

func extractRepoName(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return path
}

func displayTableHistory(commits []CommitSummary, opts HistoryOptions) {
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

	// Display commit history
	displayCommitHistory(commits, opts.Detailed, tableWidth)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
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
