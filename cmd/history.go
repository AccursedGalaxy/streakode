package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
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
	Hash            string
	Date            time.Time
	Message         string
	FileCount       int
	Additions       int
	Deletions       int
	TotalLines      int
	FilesChanged    []string
	Branch          string
	Repository      string
	Author          string
	MatchingContent []string // Added to store matching content
}

// FileResult represents a file and its content
type FileResult struct {
	Path         string
	Content      string
	LastModified time.Time
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
	sem := make(chan struct{}, 5)

	// Skip cache for file searches
	if opts.Format != "files" {
		// Get commits from cache
		cachedCommits := getCachedCommits(opts, since)
		for _, commit := range cachedCommits {
			commitChan <- commit
		}
	}

	// Process repositories concurrently
	cache.Cache.Range(func(path string, repo scan.RepoMetadata) bool {
		// Skip if repository doesn't match filter
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

			// Filter commits based on command context
			filteredCommits := filterCommitsByOptions(localCommits, opts)
			for _, commit := range filteredCommits {
				commit.Repository = repoName
				select {
				case commitChan <- commit:
				default:
					return
				}
			}

			// Only fetch remote data if needed and not too many local commits
			if len(localCommits) < 100 && shouldFetchRemote(repoPath) {
				wg.Add(1)
				go func() {
					defer wg.Done()
					fetchRemoteData(repoPath)
					remoteCommits := getRemoteCommitsOptimized(repoPath, opts, since)

					// Filter remote commits based on command context
					filteredRemote := filterCommitsByOptions(remoteCommits, opts)

					// Only send remote commits that aren't in local
					seenHashes := make(map[string]bool)
					for _, c := range localCommits {
						seenHashes[c.Hash] = true
					}
					for _, commit := range filteredRemote {
						if !seenHashes[commit.Hash] {
							commit.Repository = repoName
							select {
							case commitChan <- commit:
							default:
								return
							}
						}
					}
				}()
			}
		}(path)
		return true
	})

	wg.Wait()
	doneChan <- true
	close(commitChan)
	close(doneChan)
}

// filterCommitsByOptions applies filtering based on command context
func filterCommitsByOptions(commits []CommitSummary, opts HistoryOptions) []CommitSummary {
	if len(commits) == 0 {
		return commits
	}

	filtered := make([]CommitSummary, 0, len(commits))
	for _, commit := range commits {
		// Apply author filter if specified
		if opts.Author != "" && !strings.Contains(commit.Author, opts.Author) {
			continue
		}

		// Apply repository filter if specified
		if opts.Repository != "" && !strings.EqualFold(commit.Repository, opts.Repository) {
			continue
		}

		// Apply file pattern filter if specified (for files command)
		if opts.Query != "" && opts.Format == "files" {
			hasMatchingFile := false
			pattern := opts.Query
			// Convert glob pattern to a more flexible matching pattern
			if strings.Contains(pattern, "*") {
				pattern = strings.ReplaceAll(pattern, ".", "\\.")
				pattern = strings.ReplaceAll(pattern, "*", ".*")
			}
			for _, file := range commit.FilesChanged {
				matched, err := filepath.Match(opts.Query, filepath.Base(file))
				if err == nil && matched {
					hasMatchingFile = true
					break
				}
				// Try regex matching if glob matching fails or for more complex patterns
				if !hasMatchingFile {
					if matched, _ := regexp.MatchString(pattern+"$", file); matched {
						hasMatchingFile = true
						break
					}
				}
			}
			if !hasMatchingFile {
				continue
			}
		}

		filtered = append(filtered, commit)
	}

	return filtered
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
	// For file searches, we want to show files and their contents
	if opts.Format == "files" {
		var commits []CommitSummary
		repoName := extractRepoName(repoPath)

		// Get all commits in time range
		args := []string{
			"-C", repoPath,
			"log",
			"--no-merges",
			"--format=%H", // Just get commit hashes
			"--after=" + since.Format("2006-01-02"),
		}

		cmd := exec.Command("git", args...)
		output, err := cmd.Output()
		if err != nil {
			return nil
		}

		// Process each commit
		commitHashes := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, hash := range commitHashes {
			if hash == "" {
				continue
			}

			// Get files changed in this commit
			filesArgs := []string{
				"-C", repoPath,
				"diff-tree",
				"--no-commit-id",
				"--name-only",
				"-r",
				hash,
			}
			filesCmd := exec.Command("git", filesArgs...)
			filesOutput, err := filesCmd.Output()
			if err != nil {
				continue
			}

			files := strings.Split(strings.TrimSpace(string(filesOutput)), "\n")
			for _, file := range files {
				if file == "" || !strings.HasSuffix(file, ".go") {
					continue
				}

				// Get file content at this commit
				contentArgs := []string{
					"-C", repoPath,
					"show",
					hash + ":" + file,
				}
				contentCmd := exec.Command("git", contentArgs...)
				content, err := contentCmd.Output()
				if err != nil {
					continue
				}

				// Get commit info
				infoArgs := []string{
					"-C", repoPath,
					"show",
					"--format=%H%n%aI%n%an%n%ae%n%s",
					"-s",
					hash,
				}
				infoCmd := exec.Command("git", infoArgs...)
				info, err := infoCmd.Output()
				if err != nil {
					continue
				}

				commit := parseFileCommit(string(info), file)
				if commit != nil {
					commit.Repository = repoName
					commit.MatchingContent = []string{string(content)}
					commits = append(commits, *commit)
				}
			}
		}
		return commits
	}

	// For other modes, use the existing commit history logic
	args := []string{
		"-C", repoPath,
		"log",
		"--no-merges",
		"--name-only",
		"--format=%H%n%aI%n%an%n%ae%n%s%n%x00",
		"--after=" + since.Format("2006-01-02"),
		"--max-count=1000",
	}

	if opts.Author != "" {
		args = append(args, "--author="+opts.Author)
	}

	if opts.Branch != "" {
		args = append(args, opts.Branch)
	} else {
		args = append(args, "--all")
	}

	cmd := exec.CommandContext(context.Background(), "git", args...)
	output, err := cmd.Output()
	if err != nil {
		if config.AppConfig.Debug {
			fmt.Printf("Error getting local commits from %s: %v\n", repoPath, err)
		}
		return nil
	}

	return parseGitLogWithPatch(string(output), opts)
}

func parseFileCommit(output string, file string) *CommitSummary {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 5 {
		return nil
	}

	hash := lines[0]
	date, _ := time.Parse(time.RFC3339, lines[1])
	authorName := lines[2]
	authorEmail := lines[3]
	message := lines[4]

	return &CommitSummary{
		Hash:         hash,
		Date:         date,
		Message:      message,
		FileCount:    1,
		Author:       fmt.Sprintf("%s <%s>", authorName, authorEmail),
		FilesChanged: []string{file},
	}
}

func isGrepNoMatchError(err error) bool {
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode() == 1
	}
	return false
}

func parseGitLogWithPatch(output string, opts HistoryOptions) []CommitSummary {
	var commits []CommitSummary
	entries := strings.Split(output, "\x00")

	for _, entry := range entries {
		if entry == "" {
			continue
		}

		// Split entry into lines
		lines := strings.Split(strings.TrimSpace(entry), "\n")
		if len(lines) < 5 { // Need at least hash, date, author name, email, and message
			continue
		}

		// Parse basic commit info
		hash := lines[0]
		date, _ := time.Parse(time.RFC3339, lines[1])
		authorName := lines[2]
		authorEmail := lines[3]
		message := lines[4]

		// Process the patch to extract changed files and matching content
		var filesChanged []string
		var matchingContent []string
		var additions, deletions int
		inPatch := false
		currentFile := ""

		for i := 5; i < len(lines); i++ {
			line := lines[i]
			if strings.HasPrefix(line, "diff --git") {
				inPatch = true
				// Extract filename from diff header
				parts := strings.Split(line, " b/")
				if len(parts) > 1 {
					currentFile = parts[1]
					filesChanged = append(filesChanged, currentFile)
				}
			} else if inPatch {
				if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
					additions++
					// If we're searching for content and this line contains it
					if opts.Format == "files" && opts.Query != "" && !strings.Contains(opts.Query, "*") {
						if strings.Contains(line, opts.Query) {
							matchingContent = append(matchingContent, fmt.Sprintf("%s: %s", currentFile, strings.TrimPrefix(line, "+")))
						}
					}
				} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
					deletions++
				}
			}
		}

		// Only include commits that have matching content
		if opts.Format == "files" && opts.Query != "" && !strings.Contains(opts.Query, "*") && len(matchingContent) == 0 {
			continue
		}

		commits = append(commits, CommitSummary{
			Hash:         hash,
			Date:         date,
			Message:      message,
			FileCount:    len(filesChanged),
			Additions:    additions,
			Deletions:    deletions,
			TotalLines:   additions + deletions,
			Author:       fmt.Sprintf("%s <%s>", authorName, authorEmail),
			FilesChanged: filesChanged,
			// Store matching content for display
			MatchingContent: matchingContent,
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
				"--patch",                              // Show the actual changes
				"--unified=3",                          // Show 3 lines of context
				"--format=%H%n%aI%n%an%n%ae%n%s%n%x00", // Use newlines and null byte as separators
				branchName,
				"--after=" + since.Format("2006-01-02"),
				"--max-count=500", // Limit per branch
			}

			if opts.Author != "" {
				args = append(args, "--author="+opts.Author)
			}

			// Add file filter if in files mode
			if opts.Format == "files" && opts.Query != "" {
				if strings.Contains(opts.Query, "*") {
					args = append(args, "--", fmt.Sprintf("*%s", strings.TrimPrefix(opts.Query, "*")))
				} else {
					args = append(args, "-G", opts.Query) // -G uses basic regex for matching
				}
			}

			cmd := exec.CommandContext(ctx, "git", args...)
			output, err := cmd.Output()
			if err != nil {
				return
			}

			branchCommits := parseGitLogWithPatch(string(output), opts)

			mu.Lock()
			commits = append(commits, branchCommits...)
			mu.Unlock()
		}(branch)
	}

	wg.Wait()
	return commits
}

func displayInteractiveHistoryProgressive(commitChan <-chan CommitSummary, doneChan <-chan bool, opts HistoryOptions) {
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
				if !seen[commit.Hash] {
					seen[commit.Hash] = true
					result := search.SearchResult{
						Hash:         commit.Hash,
						Date:         commit.Date,
						Author:       commit.Author,
						Message:      commit.Message,
						Repository:   commit.Repository,
						FileCount:    commit.FileCount,
						FilesChanged: commit.FilesChanged,
					}

					// For file mode, show file path as the main message
					if opts.Format == "files" && len(commit.FilesChanged) > 0 {
						result.Message = commit.FilesChanged[0] // Use the file path as message
					}

					resultsChan <- result
				}
			case <-doneChan:
				return
			}
		}
	}()

	// Configure search options
	searchOpts := search.SearchOptions{
		Preview:     true,
		DetailLevel: getDetailLevelForFormat(opts.Format),
		Repository:  opts.Repository,
		Author:      opts.Author,
		Interactive: true,
		Format:      opts.Format,
		Progressive: true,
	}

	// Run interactive search
	selected, err := search.RunInteractiveSearchProgressive(resultsChan, searchOpts)
	if err != nil {
		fmt.Printf("Error during interactive search: %v\n", err)
		return
	}

	if len(selected) > 0 {
		handleSelectedFiles(selected)
	}
}

func handleSelectedFiles(files []search.SearchResult) {
	// Handle file selection if needed
	// This could open files in editor, show diff history, etc.
}

// getDetailLevelForFormat returns appropriate detail level based on format
func getDetailLevelForFormat(format string) int {
	switch format {
	case "detailed", "stats", "files":
		return 2
	case "compact":
		return 0
	default:
		return 1
	}
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

// filterMatchingFiles returns files that match the given pattern
func filterMatchingFiles(files []string, pattern string) []string {
	if pattern == "" {
		return files
	}

	var matched []string
	for _, file := range files {
		// Try direct glob matching first
		if m, err := filepath.Match(pattern, filepath.Base(file)); err == nil && m {
			matched = append(matched, filepath.Base(file))
			continue
		}

		// If the pattern contains *, convert to regex for more complex matches
		if strings.Contains(pattern, "*") {
			regexPattern := "^" + strings.ReplaceAll(regexp.QuoteMeta(pattern), "\\*", ".*") + "$"
			if m, err := regexp.MatchString(regexPattern, filepath.Base(file)); err == nil && m {
				matched = append(matched, filepath.Base(file))
			}
		}
	}
	return matched
}
