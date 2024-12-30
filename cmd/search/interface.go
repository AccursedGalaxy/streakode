package search

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// SearchResult represents a single commit in the search results
type SearchResult struct {
	Hash        string    `json:"hash"`
	Date        time.Time `json:"date"`
	Author      string    `json:"author"`
	Message     string    `json:"message"`
	Repository  string    `json:"repository"`
	FileCount   int       `json:"file_count"`
	Additions   int       `json:"additions"`
	Deletions   int       `json:"deletions"`
	Branch      string    `json:"branch"`      // Added to show branch information
	DisplayText string    `json:"-"`           // Used for fzf display
}

// SearchOptions defines the configuration for interactive search
type SearchOptions struct {
	Preview      bool   // Whether to show preview window
	DetailLevel  int    // 0: basic, 1: detailed, 2: full
	Query        string // Initial search query
	Repository   string // Filter by repository
	Author       string // Filter by author
	Interactive  bool   // Whether to use interactive mode
	Format      string // Output format (oneline, detailed, full)
}

// RunInteractiveSearch starts an interactive search session using fzf
func RunInteractiveSearch(results []SearchResult, opts SearchOptions) ([]SearchResult, error) {
	if !isFzfAvailable() {
		return nil, fmt.Errorf("fzf is not installed. Please install fzf to use interactive search")
	}

	// Prepare the input for fzf
	input := prepareSearchInput(results)

	// Configure fzf command
	cmd := exec.Command("fzf", buildFzfArgs(opts)...)
	cmd.Stderr = os.Stderr

	// Set up pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	// Start fzf
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start fzf: %v", err)
	}

	// Write input to fzf
	go func() {
		defer stdin.Close()
		for _, line := range input {
			fmt.Fprintln(stdin, line)
		}
	}()

	// Read fzf output
	var output bytes.Buffer
	if _, err := output.ReadFrom(stdout); err != nil {
		return nil, fmt.Errorf("failed to read fzf output: %v", err)
	}

	// Wait for fzf to finish
	if err := cmd.Wait(); err != nil {
		// Check if user cancelled (exit code 130)
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
			return nil, nil
		}
		return nil, fmt.Errorf("fzf failed: %v", err)
	}

	// Process selected items
	return processSearchOutput(output.String(), results)
}

func buildFzfArgs(opts SearchOptions) []string {
	args := []string{
		"--ansi",
		"--multi",
		"--no-mouse",
		"--bind=ctrl-a:toggle-all",
		"--bind=ctrl-d:half-page-down",
		"--bind=ctrl-u:half-page-up",
		"--bind=ctrl-/:toggle-preview",
		"--bind=?:toggle-preview",
		"--header=Ctrl-a: toggle all, Ctrl-d/u: page down/up, Ctrl-/: toggle preview, Enter: select",
		"--height=80%",
		"--border=rounded",
		"--preview-window=right:60%:wrap", // Increased preview window size
	}

	if opts.Preview {
		previewCmd := buildPreviewCmd()
		args = append(args,
			"--preview", previewCmd,
		)
	}

	if opts.Query != "" {
		args = append(args, "--query", opts.Query)
	}

	return args
}

func buildPreviewCmd() string {
	return `
# Extract JSON data from the hidden part of the line
JSON=$(echo {} | sed 's/.*\x1b\[0m\x1b\[30m\(.*\)\x1b\[0m/\1/')
HASH=$(echo "$JSON" | grep -o '"hash":"[^"]*"' | cut -d'"' -f4)
REPO=$(echo "$JSON" | grep -o '"repository":"[^"]*"' | cut -d'"' -f4)

if [ -n "$HASH" ] && [ -n "$REPO" ]; then
	# Try to find the repository in common parent directories
	REPO_PATH=""
	CURRENT_DIR="$PWD"
	while [ "$CURRENT_DIR" != "/" ]; do
		if [ -d "$CURRENT_DIR/$REPO" ]; then
			REPO_PATH="$CURRENT_DIR/$REPO"
			break
		elif [ -d "$CURRENT_DIR/$REPO/.git" ]; then
			REPO_PATH="$CURRENT_DIR/$REPO"
			break
		fi
		CURRENT_DIR=$(dirname "$CURRENT_DIR")
	done

	# If repo not found in parent dirs, try common paths
	if [ -z "$REPO_PATH" ]; then
		for DIR in "$HOME/github" "$HOME/git" "$HOME/code" "$HOME/projects" "$HOME/workspace" "$HOME/dev"; do
			if [ -d "$DIR/$REPO" ]; then
				REPO_PATH="$DIR/$REPO"
				break
			fi
		done
	fi

	if [ -n "$REPO_PATH" ]; then
		cd "$REPO_PATH"
		
		# Try to get commit info
		if git show --no-patch --format="%H" $HASH >/dev/null 2>&1; then
			# Commit exists locally
			echo -e "\033[1;36m# Commit Details\033[0m"
			git show --color=always --stat $HASH 2>/dev/null
			
			echo -e "\n\033[1;36m# Full Commit Message\033[0m"
			git show -s --format=%B $HASH 2>/dev/null
			
			echo -e "\n\033[1;36m# Changed Files\033[0m"
			git show --name-status --color=always $HASH 2>/dev/null
			
			echo -e "\n\033[1;36m# Diff Summary\033[0m"
			git show --color=always --stat $HASH 2>/dev/null
			
			echo -e "\n\033[1;36m# Repository Context\033[0m"
			echo "Repository: $REPO"
			echo "Location: $REPO_PATH"
			
			# Show branch information
			BRANCH=$(git branch --contains $HASH 2>/dev/null | grep '*' | cut -d' ' -f2)
			if [ -n "$BRANCH" ]; then
				echo "Branch: $BRANCH"
			fi
		else
			# Try to fetch the commit
			echo -e "\033[1;33mFetching commit data from $REPO...\033[0m"
			git fetch -f origin $HASH 2>/dev/null
			if git show --no-patch --format="%H" $HASH >/dev/null 2>&1; then
				echo -e "\033[1;36m# Commit Details\033[0m"
				git show --color=always --stat $HASH 2>/dev/null
				
				echo -e "\n\033[1;36m# Full Commit Message\033[0m"
				git show -s --format=%B $HASH 2>/dev/null
				
				echo -e "\n\033[1;36m# Changed Files\033[0m"
				git show --name-status --color=always $HASH 2>/dev/null
				
				echo -e "\n\033[1;36m# Diff Summary\033[0m"
				git show --color=always --stat $HASH 2>/dev/null
			else
				echo -e "\033[1;31mCommit not found in repository: $REPO\033[0m"
				echo "This might be because:"
				echo "1. The commit was squashed or rebased"
				echo "2. The repository needs to be fetched"
				echo "3. The commit exists in a different branch"
				echo -e "\nTry running: cd $REPO_PATH && git fetch --all"
			fi
		fi
		
		# Add GitHub link if available
		if git remote -v >/dev/null 2>&1; then
			remote_url=$(git remote get-url origin 2>/dev/null)
			if [[ $remote_url == *"github.com"* ]]; then
				echo -e "\n\033[1;36m# Remote Info\033[0m"
				echo "View on GitHub: https://github.com/$(echo $remote_url | sed 's/.*github.com[:/]//; s/\.git$//')/commit/$HASH"
			fi
		fi
	else
		echo -e "\033[1;31mRepository not found: $REPO\033[0m"
		echo "Please make sure the repository is cloned and in one of these locations:"
		echo "- Current directory or parent directories"
		echo "- ~/github"
		echo "- ~/git"
		echo "- ~/code"
		echo "- ~/projects"
		echo "- ~/workspace"
		echo "- ~/dev"
	fi
else
	echo "Could not extract commit information from selection"
fi`
}

func prepareSearchInput(results []SearchResult) []string {
	var input []string
	for _, result := range results {
		// Format the display text first
		displayText := formatDisplayText(result)
		
		// Create a hidden JSON block after the display text using ANSI escape codes
		jsonData, err := json.Marshal(result)
		if err != nil {
			continue
		}

		// Combine visible text with hidden data
		// Use only color to hide data, not background color
		input = append(input, fmt.Sprintf("%s\x1b[0m\x1b[30m%s\x1b[0m", displayText, string(jsonData)))
	}
	return input
}

func formatDisplayText(result SearchResult) string {
	date := result.Date.Format("2006-01-02 15:04")
	author := truncateString(result.Author, 20)
	repo := truncateString(result.Repository, 15)
	message := truncateMessage(result.Message, 50)
	changes := fmt.Sprintf("+%d/-%d", result.Additions, result.Deletions)
	
	// Color scheme
	dateColor := "\x1b[38;5;242m" // Gray
	authorColor := "\x1b[38;5;214m" // Orange
	repoColor := "\x1b[38;5;039m" // Blue
	msgColor := "\x1b[38;5;252m"  // Light gray
	statsColor := "\x1b[38;5;035m" // Green
	reset := "\x1b[0m"

	// Format: date author repository: message (+x/-y)
	return fmt.Sprintf("%s%s%s %s%s%s %s%s%s: %s%s%s %s(%s)%s",
		dateColor, date, reset,
		authorColor, author, reset,
		repoColor, repo, reset,
		msgColor, message, reset,
		statsColor, changes, reset)
}

// truncateString ensures any string fits nicely in the display
func truncateString(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// truncateMessage ensures the commit message fits nicely in the display
func truncateMessage(msg string, maxLen int) string {
	msg = strings.TrimSpace(msg)
	msg = strings.ReplaceAll(msg, "\n", " ")
	
	if len(msg) <= maxLen {
		return msg
	}
	
	return msg[:maxLen-3] + "..."
}

func processSearchOutput(output string, _ []SearchResult) ([]SearchResult, error) {
	if output == "" {
		return nil, nil
	}

	var selected []SearchResult
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		var result SearchResult
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			continue
		}
		selected = append(selected, result)
	}

	return selected, nil
}

func isFzfAvailable() bool {
	_, err := exec.LookPath("fzf")
	return err == nil
} 