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
	"sync"
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

	// Create channels for progressive loading
	commitChan := make(chan CommitSummary, 100)
	doneChan := make(chan bool)

	// Start loading commits in background
	go loadCommitsProgressively(opts, commitChan, doneChan)

	// Start interactive search immediately
	displayInteractiveHistoryProgressive(commitChan, doneChan, opts)
}

func loadCommitsProgressively(opts HistoryOptions, commitChan chan<- CommitSummary, doneChan chan<- bool) {
	var wg sync.WaitGroup
	since := time.Now().AddDate(0, 0, -opts.Days)

	// Use semaphore to limit concurrent git operations
	sem := make(chan struct{}, 5) // Limit to 5 concurrent git operations

	// First try to get commits from cache
	cachedCommits := getCachedCommits(opts, since)
	for _, commit := range cachedCommits {
		commitChan <- commit
	}

	// Then process repositories concurrently
	cache.Cache.Range(func(path string, repo scan.RepoMetadata) bool {
		if opts.Repository != "" && !matchesRepository(path, opts.Repository) {
			return true
		}

		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore
		go func(repoPath string) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			repoName := extractRepoName(repoPath)
			localCommits := getLocalCommitsOptimized(repoPath, opts, since)
			for _, commit := range localCommits {
				commit.Repository = repoName
				select {
				case commitChan <- commit:
				default:
					// Channel might be closed, skip
					return
				}
			}

			// Only fetch remote data if needed and not too many local commits
			if len(localCommits) < 100 && shouldFetchRemote(repoPath) {
				// Fetch remote data in background
				wg.Add(1)
				go func() {
					defer wg.Done()
					fetchRemoteData(repoPath)
					remoteCommits := getRemoteCommitsOptimized(repoPath, opts, since)
					// Only send remote commits that aren't in local
					seenHashes := make(map[string]bool)
					for _, c := range localCommits {
						seenHashes[c.Hash] = true
					}
					for _, commit := range remoteCommits {
						if !seenHashes[commit.Hash] {
							commit.Repository = repoName
							select {
							case commitChan <- commit:
							default:
								// Channel might be closed, skip
								return
							}
						}
					}
				}()
			}
		}(path)
		return true
	})

	// Wait for all goroutines to complete before closing channels
	wg.Wait()
	doneChan <- true
	close(commitChan)
	close(doneChan)
}

func getCachedCommits(opts HistoryOptions, since time.Time) []CommitSummary {
	var commits []CommitSummary
	cache.Cache.Range(func(path string, repo scan.RepoMetadata) bool {
		if opts.Repository != "" && !matchesRepository(path, opts.Repository) {
			return true
		}

		repoName := extractRepoName(path)
		for _, ch := range repo.CommitHistory {
			if ch.Date.After(since) {
				commits = append(commits, CommitSummary{
					Hash:       ch.Hash,
					Date:       ch.Date,
					Message:    ch.MessageHead,
					FileCount:  ch.FileCount,
					Additions:  ch.Additions,
					Deletions:  ch.Deletions,
					Repository: repoName,
				})
			}
		}
		return true
	})
	return commits
}

func getLocalCommitsOptimized(repoPath string, opts HistoryOptions, since time.Time) []CommitSummary {
	var commits []CommitSummary

	// Build optimized git log command
	args := []string{
		"-C", repoPath,
		"log",
		"--no-merges",
		"--date-order",
		"--pretty=format:%H%x00%aI%x00%an%x00%ae%x00%s%x00",
		"--numstat",
		"--after=" + since.Format("2006-01-02"),
		"--max-count=1000", // Limit to prevent excessive processing
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
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	output, err := cmd.Output()
	if err != nil {
		if config.AppConfig.Debug {
			fmt.Printf("Error getting local commits from %s: %v\n", repoPath, err)
		}
		return commits
	}

	return parseGitLogOptimized(string(output))
}

func parseGitLogOptimized(output string) []CommitSummary {
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

		// Parse commit info more efficiently
		parts := strings.Split(lines[0], "\x00")
		if len(parts) < 5 {
			continue
		}

		date, _ := time.Parse(time.RFC3339, parts[1])

		// Process stats in batches
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
			Hash:       parts[0],
			Date:       date,
			Message:    parts[4],
			FileCount:  fileCount,
			Additions:  additions,
			Deletions:  deletions,
			TotalLines: additions + deletions,
			Author:     fmt.Sprintf("%s <%s>", parts[2], parts[3]),
		})
	}

	return commits
}

func getRemoteCommitsOptimized(repoPath string, opts HistoryOptions, since time.Time) []CommitSummary {
	var commits []CommitSummary

	// Get remote branches efficiently
	cmd := exec.Command("git", "-C", repoPath, "for-each-ref", "--format=%(refname)", "refs/remotes/origin")
	output, err := cmd.Output()
	if err != nil {
		return commits
	}

	branches := strings.Split(string(output), "\n")

	// Process branches in parallel with a limit
	sem := make(chan struct{}, 3) // Limit concurrent branch processing
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, branch := range branches {
		if branch == "" {
			continue
		}

		wg.Add(1)
		go func(branchName string) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			args := []string{
				"-C", repoPath,
				"log",
				"--no-merges",
				"--date-order",
				branchName,
				"--pretty=format:%H%x00%aI%x00%an%x00%ae%x00%s%x00",
				"--numstat",
				"--after=" + since.Format("2006-01-02"),
				"--max-count=500", // Limit per branch
			}

			if opts.Author != "" {
				args = append(args, "--author="+opts.Author)
			}

			cmd := exec.CommandContext(ctx, "git", args...)
			output, err := cmd.Output()
			if err != nil {
				return
			}

			branchCommits := parseGitLogOptimized(string(output))

			mu.Lock()
			commits = append(commits, branchCommits...)
			mu.Unlock()
		}(branch)
	}

	wg.Wait()
	return commits
}

func displayInteractiveHistoryProgressive(commitChan <-chan CommitSummary, doneChan <-chan bool, opts HistoryOptions) {
	// Convert commits to search results as they arrive
	resultsChan := make(chan search.SearchResult, 100)
	go func() {
		defer close(resultsChan)

		seen := make(map[string]bool)
		for {
			select {
			case commit, ok := <-commitChan:
				if !ok {
					return
				}
				// Deduplicate commits
				if !seen[commit.Hash] {
					seen[commit.Hash] = true
					resultsChan <- search.SearchResult{
						Hash:       commit.Hash,
						Date:       commit.Date,
						Message:    commit.Message,
						FileCount:  commit.FileCount,
						Additions:  commit.Additions,
						Deletions:  commit.Deletions,
						Repository: commit.Repository,
						Branch:     commit.Branch,
						Author:     commit.Author,
					}
				}
			case <-doneChan:
				return
			}
		}
	}()

	// Configure search options
	searchOpts := search.SearchOptions{
		Preview:     opts.Preview,
		DetailLevel: boolToInt(opts.Detailed),
		Repository:  opts.Repository,
		Author:      opts.Author,
		Interactive: true,
		Format:      opts.Format,
		Progressive: true,
	}

	// Run interactive search with progressive loading
	selected, err := search.RunInteractiveSearchProgressive(resultsChan, searchOpts)
	if err != nil {
		if err.Error() == "fzf is not installed" {
			fmt.Println("Interactive search requires fzf. Falling back to table view.")
			// Collect remaining commits for table view
			var commits []CommitSummary
			for commit := range commitChan {
				commits = append(commits, commit)
			}
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
