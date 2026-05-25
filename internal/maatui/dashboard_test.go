package maatui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sunday-studio/maat/internal/maat"
)

func TestDashboardFromLegacyProjectsCountsStatus(t *testing.T) {
	dashboard := DashboardFromLegacyProjects([]maat.Project{
		{
			ID:      "orion",
			Title:   "Orion",
			Status:  "active",
			Updated: "2026-05-25",
			Current: "Current Orion summary.",
			Goals: []maat.Goal{
				{
					ID:     "G-001",
					Status: "active",
					Tickets: []maat.Ticket{
						{ID: "T-001", Done: false},
						{ID: "T-002", Done: true},
					},
				},
				{ID: "G-002", Status: "done"},
			},
		},
	})

	if dashboard.Summary.Projects != 1 {
		t.Fatalf("projects = %d, want 1", dashboard.Summary.Projects)
	}
	if dashboard.Summary.ActiveGoals != 1 || dashboard.Summary.DoneGoals != 1 || dashboard.Summary.Goals != 2 {
		t.Fatalf("goal counts = %+v", dashboard.Summary)
	}
	if dashboard.Summary.OpenTickets != 1 || dashboard.Summary.DoneTickets != 1 || dashboard.Summary.Tickets != 2 {
		t.Fatalf("ticket counts = %+v", dashboard.Summary)
	}
	if len(dashboard.Projects) != 1 || dashboard.Projects[0].Tickets != 2 {
		t.Fatalf("project rows = %+v", dashboard.Projects)
	}
	if dashboard.Projects[0].OpenTickets != 1 || dashboard.Projects[0].DoneTickets != 1 {
		t.Fatalf("project ticket counts = %+v", dashboard.Projects[0])
	}
	if dashboard.Projects[0].Summary != "Current Orion summary." {
		t.Fatalf("project summary = %q", dashboard.Projects[0].Summary)
	}
	if len(dashboard.Projects[0].GoalRows) != 2 || dashboard.Projects[0].GoalRows[0].Tickets != 2 {
		t.Fatalf("goal rows = %+v", dashboard.Projects[0].GoalRows)
	}
}

func TestRenderSummary(t *testing.T) {
	got := RenderSummary(maat.StatusSummary{
		Projects:    2,
		Goals:       5,
		ActiveGoals: 3,
		DoneGoals:   1,
		Tickets:     8,
		OpenTickets: 6,
		DoneTickets: 2,
	})

	want := "Projects: 2  Goals: 3 active / 1 done / 5 total  Tickets: 6 open / 2 done / 8 total"
	if got != want {
		t.Fatalf("RenderSummary() = %q, want %q", got, want)
	}
}

func TestRenderProjectTable(t *testing.T) {
	got := RenderProjectTable([]ProjectRow{
		{
			Key:         "a-very-long-project-key",
			DisplayName: "A Very Long Project Name",
			Status:      "active",
			Goals:       2,
			Tickets:     7,
			Updated:     "2026-05-25",
		},
	})

	for _, want := range []string{"Project", "Status", "A Very Long P...", "active", "2026-05-25"} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderProjectTable() missing %q in:\n%s", want, got)
		}
	}
}

func TestRenderSelectableProjectTableMarksSelectedProject(t *testing.T) {
	got := RenderSelectableProjectTable([]ProjectRow{
		{Key: "orion", DisplayName: "Orion", Status: "active"},
		{Key: "aether", DisplayName: "Aether", Status: "waiting"},
	}, 1)

	for _, want := range []string{"> Aether", "  Orion"} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderSelectableProjectTable() missing %q in:\n%s", want, got)
		}
	}
}

func TestRenderProjectDetailShowsSummaryGoalsAndTicketCounts(t *testing.T) {
	got := RenderProjectDetail(ProjectRow{
		Key:         "orion",
		DisplayName: "Orion",
		Status:      "active",
		Summary:     "Shipping the agent monitor.",
		Tickets:     3,
		OpenTickets: 2,
		DoneTickets: 1,
		Updated:     "2026-05-25",
		GoalRows: []GoalRow{
			{ID: "G-001", Title: "Improve monitor clarity", Status: "active", Tickets: 3},
		},
	})

	for _, want := range []string{"Project Detail", "Orion", "2 open / 1 done / 3 total", "Shipping the agent monitor.", "G-001", "Improve monitor clarity"} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderProjectDetail() missing %q in:\n%s", want, got)
		}
	}
}

func TestRenderDashboardShowsNavigationHelp(t *testing.T) {
	got := RenderDashboardWithSelection(Dashboard{Projects: []ProjectRow{
		{Key: "orion", DisplayName: "Orion", Status: "active"},
	}}, 0)

	for _, want := range []string{"Search and timeline views are planned", "up/down or k/j", "q to quit"} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderDashboardWithSelection() missing %q in:\n%s", want, got)
		}
	}
}

func TestModelSelectionMovesWithArrowKeys(t *testing.T) {
	model := NewModel(Dashboard{Projects: []ProjectRow{
		{Key: "orion"},
		{Key: "aether"},
	}}, nil)

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	if cmd != nil {
		t.Fatalf("down command = %v, want nil", cmd)
	}
	got := updated.(Model)
	if got.selected != 1 {
		t.Fatalf("selected after down = %d, want 1", got.selected)
	}

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyDown})
	got = updated.(Model)
	if got.selected != 1 {
		t.Fatalf("selected after second down = %d, want clamped 1", got.selected)
	}

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyUp})
	got = updated.(Model)
	if got.selected != 0 {
		t.Fatalf("selected after up = %d, want 0", got.selected)
	}
}

func TestModelQuitKeyReturnsCommand(t *testing.T) {
	model := NewModel(Dashboard{}, nil)
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("quit command is nil")
	}
}
