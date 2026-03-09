package main

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestListProjects(t *testing.T) {
	projects, err := ListProjects()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range projects {
		fmt.Printf("  %s -> %s\n", p.Name, p.Path)
	}
	if len(projects) == 0 {
		t.Fatal("expected at least one project")
	}
}

func TestListSessions(t *testing.T) {
	sessions, err := ListSessions("-home-mbadyl-Projects-claude-webui")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range sessions {
		display := s.Display
		if len(display) > 80 {
			display = display[:80] + "..."
		}
		fmt.Printf("  [%d] %s: %s\n", s.Timestamp, s.ID[:8], display)
	}
	if len(sessions) == 0 {
		t.Fatal("expected at least one session")
	}
}

func TestLoadSession(t *testing.T) {
	sessions, err := ListSessions("-home-mbadyl-Projects-claude-webui")
	if err != nil || len(sessions) == 0 {
		t.Skip("no sessions to load")
	}

	messages, err := LoadSession("-home-mbadyl-Projects-claude-webui", sessions[0].ID)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("  Loaded %d messages from session %s\n", len(messages), sessions[0].ID[:8])
	for i, m := range messages {
		if i > 4 {
			fmt.Printf("  ... and %d more\n", len(messages)-5)
			break
		}
		preview := fmt.Sprintf("%v", m.Content)
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		fmt.Printf("  [%s] %s: %s\n", m.Type, m.Role, preview)
	}

	raw, _ := json.MarshalIndent(messages[0], "", "  ")
	fmt.Printf("\n  First message raw:\n%s\n", string(raw))
}
