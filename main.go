package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/AccursedGalaxy/streakode/cache"
	"github.com/AccursedGalaxy/streakode/cmd"
	"github.com/AccursedGalaxy/streakode/config"
	"github.com/spf13/cobra"
)

var Version = "dev" // This will be overwritten during build

type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func getCacheFilePath(profile string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	
	if profile == "" {
		return filepath.Join(home, ".streakode.cache")
	}
	return filepath.Join(home, fmt.Sprintf(".streakode_%s.cache", profile))
}

func downloadFile(filepath string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.CreateTemp("", "streakode-update-*")
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, resp.Body); err != nil {
		return err
	}

	if err := os.Chmod(out.Name(), 0755); err != nil {
		return err
	}

	if err := os.Rename(out.Name(), filepath); err != nil {
		// If simple rename fails (e.g., cross-device), try copy & remove
		if err := copyFileContents(out.Name(), filepath); err != nil {
			return err
		}
	}

	return nil
}

func copyFileContents(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	err = os.WriteFile(dst, input, 0755)
	if err != nil {
		return err
	}

	os.Remove(src) // Clean up temp file
	return nil
}

func getCurrentExecutablePath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

func main() {
	var profile string

	rootCmd := &cobra.Command{
		Use:   "streakode",
		Short: "A Git activity tracker for monitoring coding streaks",
			Version: Version,
			PersistentPreRun: func(cmd *cobra.Command, args []string) {
				cacheFilePath := getCacheFilePath(profile)
				config.LoadConfig(profile)
				cache.InitCache()
				cache.LoadCache(cacheFilePath)
				cache.RefreshCache(config.AppConfig.ScanDirectories, config.AppConfig.Author, cacheFilePath)
			},
	}

	// Add profile flag to root command
	rootCmd.PersistentFlags().StringVarP(&profile, "profile", "p", "", "Config profile to use (e.g., work, home)")

	statsCmd := &cobra.Command{
		Use: "stats",
		Short: "Display stats for all active repositories",
		Run: func(cobraCmd *cobra.Command, args []string) {
			cmd.DisplayStats()
		},
	}
	refreshCmd := &cobra.Command{
		Use: "refresh",
		Short: "Refresh the streakode cache",
		Run: func(cobraCmd *cobra.Command, args []string) {
			cacheFilePath := getCacheFilePath(profile)
			err := cache.RefreshCache(config.AppConfig.ScanDirectories, config.AppConfig.Author, cacheFilePath)
			if err == nil {
				fmt.Println("✨ Cache refreshed successfully!")
			}
		},
	}

	// Add profile command
	profileCmd := &cobra.Command{
		Use:   "profile [name]",
		Short: "Set or show current profile",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				if config.AppState.ActiveProfile == "" {
					fmt.Println("Using default profile")
				} else {
					fmt.Printf("Using profile: %s\n", config.AppState.ActiveProfile)
				}
				return
			}
			
			profile = args[0]
			if profile == "default" || profile == "-" {
				profile = ""
				fmt.Println("Switched to default profile")
			} else {
				fmt.Printf("Switched to profile: %s\n", profile)
			}
			
			config.AppState.ActiveProfile = profile
			if err := config.SaveState(); err != nil {
				fmt.Printf("Warning: Could not save profile state: %v\n", err)
			}
			
			// Reload configuration with new profile
			config.LoadConfig(profile)
			
			// Refresh cache for new profile
			cacheFilePath := getCacheFilePath(profile)
			cache.InitCache()
			cache.LoadCache(cacheFilePath)
			cache.RefreshCache(config.AppConfig.ScanDirectories, config.AppConfig.Author, cacheFilePath)
		},
	}

	// Add version command
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Show streakode version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Streakode version %s\n", Version)
		},
	}

	// Add update command
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update streakode to the latest version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Checking for updates...")
			
			resp, err := http.Get("https://api.github.com/repos/AccursedGalaxy/streakode/releases/latest")
			if err != nil {
				fmt.Printf("Error checking for updates: %v\n", err)
				return
			}
			defer resp.Body.Close()

			var release GitHubRelease
			if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
				fmt.Printf("Error parsing release info: %v\n", err)
				return
			}

			latestVersion := strings.TrimPrefix(release.TagName, "v")
			currentVersion := strings.TrimPrefix(Version, "v")
			
			if latestVersion == currentVersion {
				fmt.Println("✨ You are already running the latest version!")
				return
			}

			fmt.Printf("New version available: %s (current: %s)\n", latestVersion, currentVersion)
			
			// Find the correct asset for the current platform
			var downloadURL string
			osName := runtime.GOOS
			archName := runtime.GOARCH
			expectedName := fmt.Sprintf("streakode-%s-%s", osName, archName)
			if osName == "windows" {
				expectedName += ".exe"
			}

			for _, asset := range release.Assets {
				if strings.Contains(asset.Name, expectedName) {
					downloadURL = asset.BrowserDownloadURL
					break
				}
			}

			if downloadURL == "" {
				fmt.Printf("Error: No compatible binary found for %s-%s\n", osName, archName)
				return
			}

			fmt.Println("📦 Downloading update...")
			
			// Get current executable path
			execPath, err := getCurrentExecutablePath()
			if err != nil {
				fmt.Printf("Error getting current executable path: %v\n", err)
				return
			}

			// Create backup of current executable
			backupPath := execPath + ".backup"
			if err := os.Rename(execPath, backupPath); err != nil {
				fmt.Printf("Error creating backup: %v\n", err)
				return
			}

			// Download and replace the binary
			if err := downloadFile(execPath, downloadURL); err != nil {
				// Restore backup if update fails
				os.Rename(backupPath, execPath)
				fmt.Printf("Error downloading update: %v\n", err)
				return
			}

			// Remove backup on successful update
			os.Remove(backupPath)

			fmt.Printf("✨ Successfully updated to version %s!\n", latestVersion)
			fmt.Println("Please restart streakode to use the new version.")
		},
	}

	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(refreshCmd)
	rootCmd.AddCommand(profileCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.Execute()
}
