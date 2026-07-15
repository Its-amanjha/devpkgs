package app

import (
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
