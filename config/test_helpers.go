package config

import (
	"os"
	"path/filepath"
	"testing"
)

type TestSetup struct {
	TempHome string
	OrigHome string
	T        *testing.T
}

// SetupTestEnvironment creates a temporary test environment
func SetupTestEnvironment(t *testing.T) *TestSetup {
	tmpHome, err := os.MkdirTemp("", "streakode-test-home")
	if err != nil {
		t.Fatal(err)
	}

	setup := &TestSetup{
		TempHome: tmpHome,
		OrigHome: os.Getenv("HOME"),
		T:        t,
	}

	os.Setenv("HOME", tmpHome)
	return setup
}

// Cleanup removes the temporary test environment
func (ts *TestSetup) Cleanup() {
	os.Setenv("HOME", ts.OrigHome)
	os.RemoveAll(ts.TempHome)
}

// CreateConfigFile creates a config file with the given content
func (ts *TestSetup) CreateConfigFile(name string, content []byte) {
	err := os.WriteFile(filepath.Join(ts.TempHome, name), content, 0644)
	if err != nil {
		ts.T.Fatal(err)
	}
}

// LoadTestConfig loads a config file from the testdata directory
func (ts *TestSetup) LoadTestConfig(filename string) []byte {
	content, err := os.ReadFile(filepath.Join("testdata", filename))
	if err != nil {
		ts.T.Fatal(err)
	}
	return content
} 