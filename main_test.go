package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCacheFilePath(t *testing.T) {
	// Setup temporary home directory
	tmpHome, err := os.MkdirTemp("", "streakode-test-home")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpHome)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", oldHome)

	tests := []struct {
		name     string
		profile  string
		expected string
	}{
		{
			name:     "Default Profile",
			profile:  "",
			expected: filepath.Join(tmpHome, ".streakode.cache"),
		},
		{
			name:     "Custom Profile",
			profile:  "test",
			expected: filepath.Join(tmpHome, ".streakode_test.cache"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getCacheFilePath(tt.profile)
			assert.Equal(t, tt.expected, result)
		})
	}
} 