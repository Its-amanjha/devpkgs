package pm

import "testing"

func TestParseWingetExport(t *testing.T) {
	names, versions, err := parseWingetExport([]byte(`{"Sources":[{"Packages":[{"PackageIdentifier":"Git.Git","Version":"2.54.0"}]}]}`))
	if err != nil || len(names) != 1 || versions["Git.Git"] != "2.54.0" {
		t.Fatalf("unexpected export parse: %v, %v, %v", names, versions, err)
	}
}
