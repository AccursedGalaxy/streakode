package cache

import (
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/AccursedGalaxy/streakode/scan"
)

func init() {
	// Register types for gob encoding/decoding
	gob.Register(CommitCache{})
	gob.Register(AuthorStats{})
	gob.Register(scan.CommitHistory{})
	gob.Register(scan.RepoMetadata{})
	gob.Register(time.Time{})
	gob.Register(map[string]bool{})
	gob.Register(map[string]int{})
}

// CommitCache represents the optimized cache structure
type CommitCache struct {
	// Core data
	Commits  map[string][]scan.CommitHistory // repo -> commits
	Authors  map[string]AuthorStats          // author -> stats
	LastSync time.Time
	Version  string

	// Performance optimizations
	CommitIndex map[string]map[string]bool // hash -> repo -> exists
	DateIndex   map[string][]string        // YYYY-MM-DD -> commit hashes
	AuthorIndex map[string][]string        // author -> commit hashes

	// Pre-calculated display data
	DisplayStats DisplayStats

	// Metadata
	Repositories map[string]scan.RepoMetadata
}

// AuthorStats holds aggregated statistics for an author
type AuthorStats struct {
	TotalCommits  int
	ActiveDays    map[string]bool
	CurrentStreak int
	LongestStreak int
	Languages     map[string]int
	PeakHours     map[int]int
	LastActivity  time.Time
}

// DisplayStats holds pre-calculated statistics for display
type DisplayStats struct {
	WeeklyTotal    int
	WeeklyDiff     int
	DailyAverage   float64
	TotalAdditions int
	TotalDeletions int
	PeakHour       int
	PeakCommits    int
	LanguageStats  map[string]int
	RepoStats      []RepoDisplayStats
	LastUpdate     time.Time
}

// RepoDisplayStats holds pre-calculated statistics for a repository
type RepoDisplayStats struct {
	Name           string
	WeeklyCommits  int
	CurrentStreak  int
	LongestStreak  int
	Additions      int
	Deletions      int
	LastCommitTime time.Time
}

// CacheManager handles all cache operations
type CacheManager struct {
	cache         *CommitCache
	mu            sync.RWMutex
	refreshTicker *time.Ticker
	updates       chan *CommitCache
	notifications chan CacheUpdate
	path          string
}

// CacheUpdate represents a cache update notification
type CacheUpdate struct {
	Type    string
	RepoID  string
	Changes int
}

// NewCacheManager creates a new cache manager instance
func NewCacheManager(cachePath string) *CacheManager {
	return &CacheManager{
		cache:         newCommitCache(),
		path:          cachePath,
		updates:       make(chan *CommitCache, 10),
		notifications: make(chan CacheUpdate, 100),
	}
}

func newCommitCache() *CommitCache {
	return &CommitCache{
		Commits:      make(map[string][]scan.CommitHistory),
		Authors:      make(map[string]AuthorStats),
		CommitIndex:  make(map[string]map[string]bool),
		DateIndex:    make(map[string][]string),
		AuthorIndex:  make(map[string][]string),
		Repositories: make(map[string]scan.RepoMetadata),
	}
}

// StartBackgroundRefresh initiates background refresh with specified interval
func (cm *CacheManager) StartBackgroundRefresh(interval time.Duration) {
	cm.refreshTicker = time.NewTicker(interval)
	go func() {
		for range cm.refreshTicker.C {
			cm.RefreshInBackground()
		}
	}()
}

// RefreshInBackground performs a non-blocking cache refresh
func (cm *CacheManager) RefreshInBackground() {
	go func() {
		if err := cm.Refresh(); err != nil {
			fmt.Printf("Background refresh failed: %v\n", err)
		}
	}()
}

// Refresh updates the cache with fresh data
func (cm *CacheManager) Refresh() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Create worker pool for parallel processing
	workerCount := runtime.NumCPU()
	jobs := make(chan string, len(cm.cache.Repositories))
	results := make(chan *scan.RepoMetadata, len(cm.cache.Repositories))

	// Start workers
	for i := 0; i < workerCount; i++ {
		go cm.repoWorker(jobs, results)
	}

	// Queue jobs
	for repoPath := range cm.cache.Repositories {
		jobs <- repoPath
	}
	close(jobs)

	// Collect results and update cache
	updatedRepos := make(map[string]scan.RepoMetadata)
	for i := 0; i < len(cm.cache.Repositories); i++ {
		if result := <-results; result != nil {
			updatedRepos[result.Path] = *result
		}
	}

	// Update cache with new data
	cm.updateCacheData(updatedRepos)

	return cm.Save()
}

func (cm *CacheManager) repoWorker(jobs <-chan string, results chan<- *scan.RepoMetadata) {
	for repoPath := range jobs {
		// Get existing metadata
		existing := cm.cache.Repositories[repoPath]

		// Check if refresh is needed
		if !cm.needsRefresh(repoPath) {
			results <- &existing
			continue
		}

		// Fetch fresh metadata
		meta := scan.FetchRepoMetadata(repoPath)
		results <- &meta
	}
}

func (cm *CacheManager) needsRefresh(repoPath string) bool {
	existing, exists := cm.cache.Repositories[repoPath]
	if !exists {
		return true
	}

	// Check if last analysis is too old
	return time.Since(existing.LastAnalyzed) > 15*time.Minute
}

// updateCacheData updates the cache with new repository data and pre-calculates statistics
func (cm *CacheManager) updateCacheData(newRepos map[string]scan.RepoMetadata) {
	// Pre-allocate maps for better performance
	commitsByRepo := make(map[string][]scan.CommitHistory, len(newRepos))
	authorStats := make(map[string]AuthorStats)
	commitIndex := make(map[string]map[string]bool)
	dateIndex := make(map[string][]string)
	authorIndex := make(map[string][]string)

	// Pre-calculated display stats
	displayStats := DisplayStats{
		LanguageStats: make(map[string]int),
		LastUpdate:    time.Now(),
	}

	hourStats := make(map[int]int)
	var repoStats []RepoDisplayStats

	// Process repositories sequentially for better memory usage
	for path, repo := range newRepos {
		commitStats := make([]scan.CommitHistory, 0, len(repo.CommitHistory))
		repoAdditions := 0
		repoDeletions := 0

		for _, commit := range repo.CommitHistory {
			commitStats = append(commitStats, commit)

			// Update indexes
			if commitIndex[commit.Hash] == nil {
				commitIndex[commit.Hash] = make(map[string]bool)
			}
			commitIndex[commit.Hash][path] = true

			dateKey := commit.Date.Format("2006-01-02")
			dateIndex[dateKey] = append(dateIndex[dateKey], commit.Hash)

			// Update author stats
			stats := authorStats[commit.Author]
			stats.TotalCommits++
			if stats.ActiveDays == nil {
				stats.ActiveDays = make(map[string]bool)
			}
			stats.ActiveDays[dateKey] = true
			if stats.LastActivity.Before(commit.Date) {
				stats.LastActivity = commit.Date
			}
			if stats.PeakHours == nil {
				stats.PeakHours = make(map[int]int)
			}
			stats.PeakHours[commit.Date.Hour()]++
			authorStats[commit.Author] = stats

			// Update author index
			authorIndex[commit.Author] = append(authorIndex[commit.Author], commit.Hash)

			// Update display stats
			hourStats[commit.Date.Hour()]++
			repoAdditions += commit.Additions
			repoDeletions += commit.Deletions
			displayStats.TotalAdditions += commit.Additions
			displayStats.TotalDeletions += commit.Deletions
		}

		commitsByRepo[path] = commitStats

		// Update language stats
		for lang, lines := range repo.Languages {
			displayStats.LanguageStats[lang] += lines
		}

		// Create repo display stats
		repoStats = append(repoStats, RepoDisplayStats{
			Name:           path[strings.LastIndex(path, "/")+1:],
			WeeklyCommits:  repo.WeeklyCommits,
			CurrentStreak:  repo.CurrentStreak,
			LongestStreak:  repo.LongestStreak,
			Additions:      repoAdditions,
			Deletions:      repoDeletions,
			LastCommitTime: repo.LastCommit,
		})
	}

	// Find peak coding hour
	peakHour, peakCommits := 0, 0
	for hour, commits := range hourStats {
		if commits > peakCommits {
			peakHour = hour
			peakCommits = commits
		}
	}

	// Sort repo stats by last commit time
	sort.Slice(repoStats, func(i, j int) bool {
		return repoStats[i].LastCommitTime.After(repoStats[j].LastCommitTime)
	})

	// Calculate weekly stats
	weeklyTotal := 0
	lastWeekTotal := 0

	for _, repo := range newRepos {
		weeklyTotal += repo.WeeklyCommits
		lastWeekTotal += repo.LastWeeksCommits
	}

	// Update display stats
	displayStats.WeeklyTotal = weeklyTotal
	displayStats.WeeklyDiff = weeklyTotal - lastWeekTotal
	displayStats.DailyAverage = float64(weeklyTotal) / 7
	displayStats.PeakHour = peakHour
	displayStats.PeakCommits = peakCommits
	displayStats.RepoStats = repoStats

	// Update cache with all data
	cm.cache.Commits = commitsByRepo
	cm.cache.Authors = authorStats
	cm.cache.CommitIndex = commitIndex
	cm.cache.DateIndex = dateIndex
	cm.cache.AuthorIndex = authorIndex
	cm.cache.Repositories = newRepos
	cm.cache.DisplayStats = displayStats
	cm.cache.LastSync = time.Now()
}

// Save persists the cache to disk
func (cm *CacheManager) Save() error {
	tempFile := cm.path + ".tmp"

	file, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	defer file.Close()

	// Use gob encoding for efficient binary serialization
	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(cm.cache); err != nil {
		return fmt.Errorf("failed to encode cache: %v", err)
	}

	// Atomic rename
	if err := os.Rename(tempFile, cm.path); err != nil {
		return fmt.Errorf("failed to save cache file: %v", err)
	}

	return nil
}

// Load reads the cache from disk
func (cm *CacheManager) Load() error {
	file, err := os.Open(cm.path)
	if err != nil {
		if os.IsNotExist(err) {
			cm.cache = newCommitCache()
			return nil
		}
		return fmt.Errorf("failed to open cache file: %v", err)
	}
	defer file.Close()

	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(cm.cache); err != nil {
		if err == io.EOF {
			cm.cache = newCommitCache()
			return nil
		}
		return fmt.Errorf("failed to decode cache: %v", err)
	}

	return nil
}

// GetCommits retrieves commits based on query options
func (cm *CacheManager) GetCommits(options QueryOptions) []scan.CommitHistory {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var commits []scan.CommitHistory

	if options.Author != "" {
		commits = cm.getCommitsByAuthor(options.Author, options.Since)
	} else if options.Repository != "" {
		commits = cm.getCommitsByRepo(options.Repository, options.Since)
	} else {
		commits = cm.getCommitsByDate(options.Since, options.Until)
	}

	return commits
}

// QueryOptions defines parameters for commit queries
type QueryOptions struct {
	Author     string
	Repository string
	Since      time.Time
	Until      time.Time
}

func (cm *CacheManager) getCommitsByAuthor(author string, since time.Time) []scan.CommitHistory {
	var commits []scan.CommitHistory
	hashes := cm.cache.AuthorIndex[author]

	for _, hash := range hashes {
		for repoPath := range cm.cache.CommitIndex[hash] {
			for _, commit := range cm.cache.Commits[repoPath] {
				if commit.Hash == hash && commit.Date.After(since) {
					commits = append(commits, commit)
				}
			}
		}
	}

	return commits
}

func (cm *CacheManager) getCommitsByRepo(repo string, since time.Time) []scan.CommitHistory {
	commits := cm.cache.Commits[repo]
	if since.IsZero() {
		return commits
	}

	var filtered []scan.CommitHistory
	for _, commit := range commits {
		if commit.Date.After(since) {
			filtered = append(filtered, commit)
		}
	}
	return filtered
}

func (cm *CacheManager) getCommitsByDate(since, until time.Time) []scan.CommitHistory {
	var commits []scan.CommitHistory
	current := since

	for !current.After(until) {
		dateKey := current.Format("2006-01-02")
		if hashes, exists := cm.cache.DateIndex[dateKey]; exists {
			for _, hash := range hashes {
				for repoPath := range cm.cache.CommitIndex[hash] {
					for _, commit := range cm.cache.Commits[repoPath] {
						if commit.Hash == hash {
							commits = append(commits, commit)
						}
					}
				}
			}
		}
		current = current.AddDate(0, 0, 1)
	}

	return commits
}
