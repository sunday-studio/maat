package maat

import (
	"os"
	"path/filepath"
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
		"New goals must include an outcome",
		"new tickets must include a description and acceptance criteria",
		"Store durable plans as ticket comments",
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
		ProjectKey:    "maat",
		StoragePath:   "/tmp/maat-state",
		BinaryVersion: "maat v1.2.3 (abc123, 2026-05-26)",
	})

	for _, want := range []string{
		"# Maat Agent Instructions",
		"Maat binary: maat v1.2.3 (abc123, 2026-05-26).",
		"maat setup --storage /tmp/maat-state",
		"Default storage rules live in `/tmp/maat-state/setup.md`.",
		"maat initialize --project maat --storage /tmp/maat-state",
		"Save the snippet below into `AGENTS.md`, `CLAUDE.md`, Cursor rules",
		"This repo is registered as `maat`.",
		"maat project show maat --storage /tmp/maat-state",
		"maat goal create maat",
		"--outcome",
		"--description",
		"--acceptance",
		"maat ticket claim <ticket-id> --project maat --agent \"<agent-id>\"",
		"## Next Steps",
		"maat ticket list --project maat --storage /tmp/maat-state",
		"maat ticket complete <ticket-id> --project maat --evidence \"<verification>\" --storage /tmp/maat-state",
		"maat validate --storage /tmp/maat-state",
		"Store durable plans as ticket comments",
	} {
		if !strings.Contains(document, want) {
			t.Fatalf("expected setup document to include %q, got %q", want, document)
		}
	}
}

func TestEnsureStorageSetupDocumentCreatesAndPreserves(t *testing.T) {
	store := t.TempDir()

	result, err := EnsureStorageSetupDocument(store)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Created || result.Updated || result.Existing {
		t.Fatalf("expected created setup document, got %#v", result)
	}
	if result.Path != filepath.Join(store, StorageSetupFilename) {
		t.Fatalf("unexpected setup path: %#v", result)
	}
	data, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "# Maat Setup") || !strings.Contains(string(data), "Store durable plans as ticket comments") {
		t.Fatalf("unexpected setup document: %q", data)
	}

	custom := "# Custom Rules\n"
	if err := os.WriteFile(result.Path, []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err = EnsureStorageSetupDocument(store)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Existing || result.Created || result.Updated {
		t.Fatalf("expected existing setup document, got %#v", result)
	}
	data, err = os.ReadFile(result.Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != custom {
		t.Fatalf("setup document should not be overwritten, got %q", data)
	}

	if err := os.WriteFile(result.Path, []byte(" \n\t"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err = EnsureStorageSetupDocument(store)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Updated || result.Created || result.Existing {
		t.Fatalf("expected empty setup document to be filled, got %#v", result)
	}
}
