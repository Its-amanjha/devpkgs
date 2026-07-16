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

	// Read messages until we receive ActionMsg to avoid leaking the goroutine.
	done := false
	for !done {
		select {
		case m := <-ch:
			switch v := m.(type) {
			case LogLineMsg:
				t.Logf("Received log line: %s", v.Line)
			case LogFinishMsg:
				t.Logf("Received finish: %v", v.Err)
			case ActionMsg:
				t.Logf("Received action completion: %v", v.Err)
				done = true
			default:
				t.Logf("Received other message: %T", v)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for stream messages")
		}
	}
}

func TestRunStream_StartFailure(t *testing.T) {
	ch := make(chan tea.Msg, 10)
	cmd := RunStream(ch, "test-pkg", Upgrade, "npm", "non-existent-command-xyz")
	cmd()

	var finishReceived, actionReceived bool
	done := false
	for !done {
		select {
		case m := <-ch:
			switch v := m.(type) {
			case LogFinishMsg:
				finishReceived = true
				if v.Err == nil {
					t.Error("expected error in LogFinishMsg, got nil")
				}
			case ActionMsg:
				actionReceived = true
				if v.Err == nil {
					t.Error("expected error in ActionMsg, got nil")
				}
				done = true
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for stream messages")
		}
	}

	if !finishReceived {
		t.Error("expected LogFinishMsg to be received")
	}
	if !actionReceived {
		t.Error("expected ActionMsg to be received")
	}
}

