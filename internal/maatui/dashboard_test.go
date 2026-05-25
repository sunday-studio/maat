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
						{ID: "T-001", Title: "Open monitor ticket", Done: false},
						{ID: "T-002", Title: "Done monitor ticket", Done: true},
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
	if len(dashboard.Projects[0].TicketRows) != 2 || dashboard.Projects[0].TicketRows[0].Status != "active" || dashboard.Projects[0].TicketRows[1].Status != "done" {
		t.Fatalf("ticket rows = %+v", dashboard.Projects[0].TicketRows)
	}
}

func TestDashboardFromObjectProjectsIncludesTicketRows(t *testing.T) {
	dashboard := DashboardFromObjectProjects([]maat.ObjectProject{
		{
			Key:         "orion",
			DisplayName: "Orion",
			Status:      "active",
			Goals: []maat.ObjectGoal{
				{ID: "G-001", Status: "active", Title: "Improve monitor clarity"},
			},
			Tickets: []maat.ObjectTicket{
				{ID: "T-001", Title: "Add status table", Status: "active", GoalID: "G-001"},
				{ID: "T-002", Title: "Fix deploy note", Status: "done"},
			},
		},
	})

	project := dashboard.Projects[0]
	if project.OpenTickets != 1 || project.DoneTickets != 1 || project.Tickets != 2 {
		t.Fatalf("project ticket counts = %+v", project)
	}
	if len(project.TicketRows) != 2 || project.TicketRows[0].GoalID != "G-001" || project.TicketRows[1].GoalID != "" {
		t.Fatalf("ticket rows = %+v", project.TicketRows)
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

func TestRenderProjectTicketsShowsGoalAndStandaloneTickets(t *testing.T) {
	got := RenderProjectTickets(ProjectRow{
		Key:         "orion",
		DisplayName: "Orion",
		Tickets:     2,
		OpenTickets: 1,
		DoneTickets: 1,
		TicketRows: []TicketRow{
			{ID: "T-001", Title: "Add status table", Status: "active", GoalID: "G-001"},
			{ID: "T-002", Title: "Fix deploy note", Status: "done"},
		},
	})

	for _, want := range []string{"Tickets", "Orion", "1 open / 1 done / 2 total", "T-001", "Add status table", "G-001", "T-002", "standalone"} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderProjectTickets() missing %q in:\n%s", want, got)
		}
	}
}

func TestRenderDashboardCanShowTicketMode(t *testing.T) {
	got := RenderDashboardWithSelectionAndMode(Dashboard{Projects: []ProjectRow{
		{
			Key:         "orion",
			DisplayName: "Orion",
			Tickets:     1,
			OpenTickets: 1,
			TicketRows: []TicketRow{
				{ID: "T-001", Title: "Add status table", Status: "active"},
			},
		},
	}}, 0, DetailModeTickets)

	for _, want := range []string{"Tickets", "T-001", "tab to switch detail/tickets"} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderDashboardWithSelectionAndMode() missing %q in:\n%s", want, got)
		}
	}
}

func TestRenderDashboardShowsNavigationHelp(t *testing.T) {
	got := RenderDashboardWithSelection(Dashboard{Projects: []ProjectRow{
		{Key: "orion", DisplayName: "Orion", Status: "active"},
	}}, 0)

	for _, want := range []string{"Search and timeline views are planned", "up/down or k/j", "tab to switch detail/tickets", "q to quit"} {
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

func TestModelTogglesDetailMode(t *testing.T) {
	model := NewModel(Dashboard{Projects: []ProjectRow{{Key: "orion"}}}, nil)

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	if cmd != nil {
		t.Fatalf("tab command = %v, want nil", cmd)
	}
	got := updated.(Model)
	if got.mode != DetailModeTickets {
		t.Fatalf("mode after tab = %v, want tickets", got.mode)
	}

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyRight})
	got = updated.(Model)
	if got.mode != DetailModeProject {
		t.Fatalf("mode after right = %v, want project", got.mode)
	}

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyLeft})
	got = updated.(Model)
	if got.mode != DetailModeTickets {
		t.Fatalf("mode after left = %v, want tickets", got.mode)
	}
}

func TestModelQuitKeyReturnsCommand(t *testing.T) {
	model := NewModel(Dashboard{}, nil)
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("quit command is nil")
	}
}
