package pm

import (
	"strings"
	"testing"
)

func TestParseWingetSearchOutput(t *testing.T) {
	// Simulated winget search output
	sampleOutput := `Name                            Id                              Version
-------------------------------------------------------------------------------------
Docker Desktop                  Docker.DockerDesktop             4.26.1
Visual Studio Code              Microsoft.VisualStudioCode       1.85.1
Git                             Git.Git                          2.43.0
`

	lines := strings.Split(sampleOutput, "\n")
	// Find the separator line (all dashes)
	sepIdx := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if len(trimmed) > 10 && strings.Count(trimmed, "-") > len(trimmed)/2 {
			sepIdx = i
			break
		}
	}
	if sepIdx < 1 {
		t.Fatal("separator line not found")
	}

	// Parse header to find column positions
	header := lines[sepIdx-1]
	nameIdx := 0
	idIdx := strings.Index(header, "Id")
	if idIdx < 0 {
		idIdx = strings.Index(header, "ID")
	}
	versionIdx := strings.Index(header, "Version")

	if idIdx < 0 {
		t.Fatal("could not find Id column in header")
	}
	if versionIdx < 0 {
		t.Fatal("could not find Version column in header")
	}

	// Parse data rows
	var results []SearchResult
	for i := sepIdx + 1; i < len(lines); i++ {
		line := lines[i]
		if len(line) < versionIdx {
			continue
		}
		id := strings.TrimSpace(line[idIdx:min(versionIdx, len(line))])
		version := ""
		if versionIdx < len(line) {
			version = strings.TrimSpace(line[versionIdx:])
		}
		name := strings.TrimSpace(line[nameIdx:min(idIdx, len(line))])
		if id == "" {
			continue
		}
		results = append(results, SearchResult{
			Name:        id,
			Manager:     "winget",
			Description: name,
			Version:     version,
		})
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	expected := []struct {
		name    string
		desc    string
		version string
	}{
		{"Docker.DockerDesktop", "Docker Desktop", "4.26.1"},
		{"Microsoft.VisualStudioCode", "Visual Studio Code", "1.85.1"},
		{"Git.Git", "Git", "2.43.0"},
	}

	for i, exp := range expected {
		if results[i].Name != exp.name {
			t.Errorf("result[%d].Name = %q, want %q", i, results[i].Name, exp.name)
		}
		if results[i].Description != exp.desc {
			t.Errorf("result[%d].Description = %q, want %q", i, results[i].Description, exp.desc)
		}
		if results[i].Version != exp.version {
			t.Errorf("result[%d].Version = %q, want %q", i, results[i].Version, exp.version)
		}
		if results[i].Manager != "winget" {
			t.Errorf("result[%d].Manager = %q, want %q", i, results[i].Manager, "winget")
		}
	}
}

func TestSearchBrewLocal(t *testing.T) {
	formulaeMap := map[string]FormulaData{
		"wget": {
			Name: "wget",
			Desc: "Internet file retriever",
			Versions: struct {
				Stable string `json:"stable"`
			}{Stable: "1.21.4"},
		},
		"curl": {
			Name: "curl",
			Desc: "Get a file from an HTTP, HTTPS or FTP server",
			Versions: struct {
				Stable string `json:"stable"`
			}{Stable: "8.5.0"},
		},
		"jq": {
			Name: "jq",
			Desc: "Lightweight and flexible command-line JSON processor",
			Versions: struct {
				Stable string `json:"stable"`
			}{Stable: "1.7.1"},
		},
	}

	cmd := SearchBrew("get", formulaeMap)
	msg := cmd()
	result, ok := msg.(RegistrySearchMsg)
	if !ok {
		t.Fatal("expected RegistrySearchMsg")
	}
	if result.Manager != "brew" {
		t.Errorf("Manager = %q, want %q", result.Manager, "brew")
	}
	if result.Err != nil {
		t.Errorf("unexpected error: %v", result.Err)
	}
	// Should match "wget" (name contains "get") and "curl" (desc contains "get")
	if len(result.Results) != 2 {
		t.Fatalf("expected 2 results, got %d: %+v", len(result.Results), result.Results)
	}
	// Results are sorted by name
	if result.Results[0].Name != "curl" {
		t.Errorf("result[0].Name = %q, want %q", result.Results[0].Name, "curl")
	}
	if result.Results[1].Name != "wget" {
		t.Errorf("result[1].Name = %q, want %q", result.Results[1].Name, "wget")
	}
}

func TestSearchBrewEmptyMap(t *testing.T) {
	cmd := SearchBrew("test", nil)
	msg := cmd()
	result, ok := msg.(RegistrySearchMsg)
	if !ok {
		t.Fatal("expected RegistrySearchMsg")
	}
	if result.Manager != "brew" {
		t.Errorf("Manager = %q, want %q", result.Manager, "brew")
	}
	if len(result.Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(result.Results))
	}
}
