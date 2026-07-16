package pm

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestRunStream(t *testing.T) {
	ch := make(chan tea.Msg, 10)
	cmd := RunStream(ch, "test-pkg", Upgrade, "npm", "go", "version")
	msg := cmd()
	if msg != nil {
		t.Errorf("expected cmd to return nil tea.Msg immediately, got %v", msg)
	}

	// Read first message
	select {
	case m := <-ch:
		switch v := m.(type) {
		case LogLineMsg:
			t.Logf("Received log line: %s", v.Line)
		case LogFinishMsg:
			t.Logf("Received finish: %v", v.Err)
		default:
			t.Logf("Received other message: %T", v)
		}
	case <-time.After(2 * time.Second):
		t.Error("timed out waiting for log output")
	}
}
