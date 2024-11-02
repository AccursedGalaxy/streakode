package git

import (
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// Repository represents a Git repository with its basic information
type RepositoryInfo struct {
	Path string
	Name string
}

type CommitInfo struct {
	Hash      string
	Message   string
	Author    string
	Timestamp time.Time
}

type RepositoryStats struct {
	TotalCommits      int
	TotalAdditions    int
	TotalDeletions    int
	FirstCommit       CommitInfo
	LastCommit        CommitInfo
	LastCommitTimestamp time.Time
	FirstCommitTimestamp time.Time
	ActiveDays        int
	CommitsByDay      map[string]int // key is YYYY-MM-DD
	CommitsByHour     map[int]int     // key is hour of the day (0-23)
	CommitsByAuthor   map[string]int   // key is author name
}

// AnalyzeRepository analyzes the repository and returns statistics
func (r *Repository) AnalyzeRepository() (*RepositoryStats, error) {
	repo, err := git.PlainOpen(r.Path)
	if err != nil {
		return nil, err
	}

	stats := &RepositoryStats{
		CommitsByDay:    make(map[string]int),
		CommitsByHour:   make(map[int]int),
		CommitsByAuthor: make(map[string]int),
	}

	ref, err := repo.Head()
	if err != nil {
		return nil, err
	}

	commits, err := repo.Log(&git.LogOptions{From: ref.Hash()})
	if err != nil {
		return nil, err
	}

	err = commits.ForEach(func(c *object.Commit) error {
		stats.TotalCommits++

		// Update first and last commit dates
		if stats.FirstCommitTimestamp.IsZero() || c.Author.When.Before(stats.FirstCommitTimestamp) {
			stats.FirstCommitTimestamp = c.Author.When
			stats.FirstCommit = CommitInfo{
				Hash:      c.Hash.String(),
				Message:   c.Message,
				Author:    c.Author.Name,
				Timestamp: c.Author.When,
			}
		}

		if stats.LastCommitTimestamp.IsZero() || c.Author.When.After(stats.LastCommitTimestamp) {
			stats.LastCommitTimestamp = c.Author.When
			stats.LastCommit = CommitInfo{
				Hash:      c.Hash.String(),
				Message:   c.Message,
				Author:    c.Author.Name,
				Timestamp: c.Author.When,
			}
		}

		// Update commit counts
		dateStr := c.Author.When.Format("2006-01-02")
		stats.CommitsByDay[dateStr]++
		stats.CommitsByHour[c.Author.When.Hour()]++
		stats.CommitsByAuthor[c.Author.Email]++

		return nil
	})

	stats.ActiveDays = len(stats.CommitsByDay)

	return stats, err
}
