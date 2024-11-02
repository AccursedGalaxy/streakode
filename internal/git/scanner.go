package git

import (
	"os"
	"path/filepath"
)

// Repository represents a Git repository with its basic information
type Repository struct {
	Path string
	Name string
}

// Scanner handles Git repository scanning
type Scanner struct {
	RootPaths []string
}

// NewScanner creates a new Scanner instance
func NewScanner(paths []string) *Scanner {
	return &Scanner{
		RootPaths: paths,
	}
}

// ScanForRepositories looks for Git repositories in the specified paths
func (s *Scanner) ScanForRepositories() ([]Repository, error) {
	var repos []Repository

	for _, root := range s.RootPaths {
		// Convert relative paths to absolute
		absRoot, err := filepath.Abs(root)
		if err != nil {
			return nil, err
		}

		err = filepath.Walk(absRoot, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip if we can't read the directory
			if info == nil {
				return nil
			}

			if info.IsDir() {
				if info.Name() == ".git" {
					repoPath := filepath.Dir(path)
					repos = append(repos, Repository{
						Path: repoPath,
						Name: filepath.Base(repoPath),
					})
					return filepath.SkipDir
				}
				
				// Skip common directories we don't want to scan
				if info.Name() == "node_modules" || info.Name() == "vendor" {
					return filepath.SkipDir
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