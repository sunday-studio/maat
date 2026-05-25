package maatui

import (
	"strings"
	"testing"

	"github.com/sunday-studio/maat/internal/maat"
)

func TestDashboardFromLegacyProjectsCountsStatus(t *testing.T) {
	dashboard := DashboardFromLegacyProjects([]maat.Project{
		{
			ID:      "orion",
			Title:   "Orion",
			Status:  "active",
			Updated: "2026-05-25",
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
