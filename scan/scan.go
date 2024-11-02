package scan

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/AccursedGalaxy/streakode/config"
)

type RepoMetadata struct {
	Path           string    `json:"path"`
	LastCommit     time.Time `json:"last_commit"`
	CommitCount    int       `json:"commit_count"`
	LastActivity   string    `json:"last_activity"`
	AuthorVerified bool      `json:"author_verified"`
	Dormant        bool      `json:"dormant"`
}

// fetchRepoMeta - gets metadata for a single repository and verifies user
func fetchRepoMeta(repoPath, author string) RepoMetadata {
	meta := RepoMetadata{Path: repoPath}
	
	authorCmd := exec.Command("git", "-C", repoPath, "log", "--author="+author, "--pretty=format:%ci")
	output, err := authorCmd.Output()
	if err != nil {
		fmt.Printf("Error getting git log for %s: %v\n", repoPath, err)
		return meta
	}
	
	if len(output) > 0 {
		meta.AuthorVerified = true

		// Get last commit date and count
		lines := strings.Split(string(output), "\n")
		lastCommitTime, err := time.Parse("2006-01-02 15:04:05 -0700", lines[0])
		if err == nil {
			meta.LastCommit = lastCommitTime
		}
		meta.CommitCount = len(lines)

		// Check if dormant
		meta.Dormant = time.Since(meta.LastCommit) > time.Duration(config.AppConfig.DormantThreshold) * 24 * time.Hour
	}

	return meta
}

// ScanDirectories - scans for Git repositories in the specified directories
func ScanDirectories(dirs []string, author string) ([]RepoMetadata, error) {
	var repos []RepoMetadata

	for _, dir := range dirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info == nil {
				return nil
			}
			if info.IsDir() && info.Name() == ".git" {
				repoPath := filepath.Dir(path)
				meta := fetchRepoMeta(repoPath, author)
				if meta.AuthorVerified && !meta.Dormant {
					repos = append(repos, meta)
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return repos, nil
}
