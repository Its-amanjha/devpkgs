package pm

import (
	"strings"
	"testing"
)

func TestParseWingetList(t *testing.T) {
	output := `Name                           Id                                       Version          Available        Source
----------------------------------------------------------------------------------------------------------------
Git                            Git.Git                                  2.54.0           2.55.0           winget`
	names, versions := parseWingetList(output)
	if len(names) != 1 || versions["Git.Git"] != "2.54.0" {
		t.Fatalf("unexpected list parse: %v, %v", names, versions)
	}
}

func TestParseWingetShow(t *testing.T) {
	output := `Found Git [Git.Git]
Version: 2.55.0.3
Publisher: The Git Development Community
Description:
  Git is a free and open source distributed version control system.
  It is fast.
Homepage: https://gitforwindows.org/
License: GPL-2.0`
	data := ParseWingetShow("Git.Git", output)
	if data.Version != "2.55.0.3" || data.Publisher != "The Git Development Community" || data.Homepage != "https://gitforwindows.org/" || data.License != "GPL-2.0" || !strings.Contains(data.Description, "fast") {
		t.Fatalf("unexpected winget show parse: %+v", data)
	}
}

