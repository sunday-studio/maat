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

func TestAgentSetupDocument(t *testing.T) {
	document := AgentSetupDocument(AgentSetupOptions{
		ProjectKey:  "maat",
		StoragePath: "/tmp/maat-state",
	})

	for _, want := range []string{
		"# Maat Agent Setup",
		"Audience: any agent that can read files, run shell commands, and update Git",
		"matt init /tmp/maat-state",
		"matt storage link /tmp/maat-state",
		"Codex: add it to the repo's `AGENTS.md`",
		"Claude Code: add it to `CLAUDE.md`",
		"matt project show maat --storage /tmp/maat-state",
		"matt goal create maat",
		"matt ticket claim <ticket-id> --project maat --agent \"<agent-id>\"",
		"Treat index warnings as cache warnings",
	} {
		if !strings.Contains(document, want) {
			t.Fatalf("expected setup document to include %q, got %q", want, document)
		}
	}
}
