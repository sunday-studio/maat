package maat

import (
	"strings"
	"testing"
)

func TestAgentInstructionsSnippet(t *testing.T) {
	snippet := AgentInstructionsSnippet()
	if strings.TrimSpace(snippet) != snippet {
		t.Fatalf("snippet should not include leading or trailing whitespace: %q", snippet)
	}

	required := []string{
		"Use Maat as the canonical project memory",
		"`matt sync`",
		"`matt status`",
		"`matt project show <project>`",
		"`matt search <query>`",
		"Create or claim a ticket",
		"complete the ticket with evidence",
		"Do not mark work done without evidence",
	}
	for _, phrase := range required {
		if !strings.Contains(snippet, phrase) {
			t.Fatalf("snippet missing %q in %q", phrase, snippet)
		}
	}
}
