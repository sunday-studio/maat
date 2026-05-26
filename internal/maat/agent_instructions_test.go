package maat

import (
	"strings"
	"testing"
)

func TestAgentInstructionsSnippet(t *testing.T) {
	snippet := agentInstructionsSnippetText()
	if strings.TrimSpace(snippet) != snippet {
		t.Fatalf("snippet should not include leading or trailing whitespace: %q", snippet)
	}

	required := []string{
		"Use Maat as the canonical project memory",
		"`maat status`",
		"`maat project show <project>`",
		"Create or claim a ticket",
		"Complete tickets only with evidence",
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
		"# Maat Agent Instructions",
		"maat setup --storage /tmp/maat-state",
		"maat initialize --project maat --storage /tmp/maat-state",
		"Save the snippet below into `AGENTS.md`, `CLAUDE.md`, Cursor rules",
		"This repo is registered as `maat`.",
		"maat project show maat --storage /tmp/maat-state",
		"maat goal create maat",
		"maat ticket claim <ticket-id> --project maat --agent \"<agent-id>\"",
	} {
		if !strings.Contains(document, want) {
			t.Fatalf("expected setup document to include %q, got %q", want, document)
		}
	}
}
