package app

import (
	"strings"
	"testing"

	"devpkgs/internal/pm"
)

func TestOutdatedNpmPackage(t *testing.T) {
	m := New()
	m.states[1].versions = map[string]string{"devpkgs": "1.0.0"}
	m.states[1].NpmDetails = map[string]*pm.NpmDetailData{"devpkgs": {Version: "1.1.0"}}
	if !m.isOutdated(1, "devpkgs") {
		t.Fatal("expected a newer npm version to be marked outdated")
	}
}

func TestLogOverlay(t *testing.T) {
	m := New()
	for i := range m.states {
		m.states[i].loading = false
	}
	m.width = 80
	m.height = 24
	m.logOverlay = true
	m.logActive = true
	m.logLines = []string{"line 1", "line 2", "line 3", "line 4", "line 5"}
	
	// Test that rendering works and contains the title and lines
	viewStr := m.View()
	if !strings.Contains(viewStr, "Installation Logs") {
		t.Fatal("expected view to contain 'Installation Logs'")
	}
	if !strings.Contains(viewStr, "line 1") || !strings.Contains(viewStr, "line 5") {
		t.Fatal("expected view to contain the log lines")
	}

	// Test scrolling
	m.logScrollActive = true
	m.logScrollOffset = 1
	// Add many log lines to trigger actual scrolling logic
	for i := 0; i < 30; i++ {
		m.logLines = append(m.logLines, "log line")
	}
	viewStr = m.View()
	if !strings.Contains(viewStr, "Installation Logs") {
		t.Fatal("expected view to contain 'Installation Logs' when scrolling")
	}
}
