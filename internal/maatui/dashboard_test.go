package maatui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sunday-studio/maat/internal/maat"
)

func TestDashboardFromObjectProjectsCountsStatus(t *testing.T) {
	dashboard := DashboardFromObjectProjects([]maat.ObjectProject{
		{
			Key:         "sample",
			DisplayName: "Sample",
			Status:      "active",
			Updated:     "2026-05-25",
			Summary:     "Current Sample summary.",
			Goals: []maat.ObjectGoal{
				{ID: "G-001", Title: "Ship", Status: "active"},
				{ID: "G-002", Title: "Done", Status: "done"},
			},
			Tickets: []maat.ObjectTicket{
				{ID: "T-001", Title: "Open monitor ticket", Status: "active", GoalID: "G-001"},
				{ID: "T-002", Title: "Done monitor ticket", Status: "done", GoalID: "G-001"},
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
	if dashboard.Projects[0].Summary != "Current Sample summary." {
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
			Key:         "sample",
			DisplayName: "Sample",
			Status:      "active",
			Goals: []maat.ObjectGoal{
				{ID: "G-001", Status: "active", Title: "Improve monitor clarity"},
			},
			Tickets: []maat.ObjectTicket{
				{
					ID:          "T-001",
					Title:       "Add status table",
					Status:      "active",
					GoalID:      "G-001",
					ProjectKey:  "sample",
					Created:     "2026-05-25T19:00:00+02:00",
					Tags:        []string{"tui"},
					Description: "Make status easier to scan.",
					Acceptance:  []string{"Status is visible."},
				},
				{ID: "T-002", Title: "Fix deploy note", Status: "done"},
			},
			Events: []maat.ObjectEvent{
				{
					ID:         "E-20260525-190700-codex-a111",
					Time:       "2026-05-25T19:07:00+02:00",
					Actor:      "codex",
					ProjectKey: "sample",
					Type:       "ticket.created",
					ObjectID:   "T-001",
					Summary:    "Created the status table ticket.",
				},
				{
					ID:         "E-20260525-190800-codex-b222",
					Time:       "2026-05-25T19:08:00+02:00",
					Actor:      "codex-worker",
					ProjectKey: "sample",
					Type:       "ticket.claimed",
					ObjectID:   "T-001",
					Expires:    "2026-05-25T21:08:00+02:00",
					Summary:    "Claimed ticket T-001.",
				},
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
	if project.TicketRows[0].GoalTitle != "Improve monitor clarity" || project.TicketRows[0].Description == "" || len(project.TicketRows[0].Acceptance) != 1 {
		t.Fatalf("ticket detail fields = %+v", project.TicketRows[0])
	}
	if project.TicketRows[0].Owner != "codex-worker" || project.TicketRows[0].ClaimUntil != "2026-05-25T21:08:00+02:00" {
		t.Fatalf("ticket claim fields = %+v", project.TicketRows[0])
	}
	if len(project.EventRows) != 2 || project.EventRows[0].Type != "ticket.claimed" {
		t.Fatalf("project event rows = %+v", project.EventRows)
	}
	if len(dashboard.Events) != 2 || dashboard.Events[0].ProjectName != "Sample" || dashboard.Events[1].Type != "ticket.created" {
		t.Fatalf("event rows = %+v", dashboard.Events)
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
		{Key: "sample", DisplayName: "Sample", Status: "active"},
		{Key: "second", DisplayName: "Second", Status: "waiting"},
	}, 1)

	for _, want := range []string{"> Second", "  Sample"} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderSelectableProjectTable() missing %q in:\n%s", want, got)
		}
	}
}

func TestRenderProjectDetailShowsSummaryGoalsAndTicketCounts(t *testing.T) {
	got := RenderProjectDetail(ProjectRow{
		Key:         "sample",
		DisplayName: "Sample",
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

	for _, want := range []string{"Project Detail", "Sample", "2 open / 1 done / 3 total", "Shipping the agent monitor.", "G-001", "Improve monitor clarity"} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderProjectDetail() missing %q in:\n%s", want, got)
		}
	}
}

func TestRenderProjectTicketsShowsGoalAndStandaloneTickets(t *testing.T) {
	got := RenderProjectTickets(ProjectRow{
		Status:      "active",
		Key:         "sample",
		DisplayName: "Sample",
		Tickets:     2,
		OpenTickets: 1,
		DoneTickets: 1,
		TicketRows: []TicketRow{
			{ID: "T-001", Title: "Add status table", Status: "active", GoalID: "G-001"},
			{ID: "T-002", Title: "Fix deploy note", Status: "done"},
		},
	})

	for _, want := range []string{"Tickets Board", "Sample", "active", "1 open / 1 done / 2 total", "Open (1)", "Done (1)", "T-001", "Add status table", "G-001", "T-002", "standalone"} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderProjectTickets() missing %q in:\n%s", want, got)
		}
	}
}

func TestRenderProjectTicketBoardGroupsUsefulStatusColumns(t *testing.T) {
	got := RenderProjectTicketBoard(ProjectRow{
		Key:         "sample",
		DisplayName: "Sample",
		Status:      "active",
		Tickets:     4,
		OpenTickets: 2,
		DoneTickets: 2,
		TicketRows: []TicketRow{
			{ID: "T-001", Title: "Build board shell", Status: "active", GoalID: "G-001"},
			{ID: "T-002", Title: "Await review", Status: "waiting", GoalID: "G-001", Owner: "codex-worker"},
			{ID: "T-003", Title: "Ship old work", Status: "done"},
			{ID: "T-004", Title: "Close release note", Status: "completed"},
		},
	}, 96)

	for _, want := range []string{"Open (1)", "Waiting (1)", "Done (2)", "T-001", "[Open]", "T-002", "[Waiting]", "@codex-worker", "T-003", "T-004", "completed Close"} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderProjectTicketBoard() missing %q in:\n%s", want, got)
		}
	}
}

func TestRenderProjectTicketBoardShowsOwnershipWithoutOpeningFiles(t *testing.T) {
	got := RenderProjectTicketBoard(ProjectRow{
		Key:         "sample",
		DisplayName: "Sample",
		Tickets:     3,
		OpenTickets: 2,
		DoneTickets: 1,
		TicketRows: []TicketRow{
			{ID: "T-001", Title: "Unclaimed active work", Status: "active"},
			{ID: "T-002", Title: "Owned waiting work", Status: "blocked", Owner: "hilbert", ClaimUntil: "2026-05-25T21:08:00+02:00"},
			{ID: "T-003", Title: "Completed owned work", Status: "done", Owner: "boole"},
		},
	}, 96)

	for _, want := range []string{"T-001", "[Open]", "unowned", "T-002", "[Waiting]", "blocked Owned", "@hilbert", "T-003", "[Done]", "@boole"} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderProjectTicketBoard() missing %q in:\n%s", want, got)
		}
	}
}

func TestRenderProjectTicketBoardStacksAtNarrowWidth(t *testing.T) {
	got := RenderProjectTicketBoard(ProjectRow{
		Key:         "sample",
		DisplayName: "Sample",
		Tickets:     3,
		OpenTickets: 2,
		DoneTickets: 1,
		TicketRows: []TicketRow{
			{ID: "T-001", Title: "Keep the board readable on compact terminals", Status: "active"},
			{ID: "T-002", Title: "Wait for external deploy", Status: "blocked"},
			{ID: "T-003", Title: "Finish detail copy", Status: "done"},
		},
	}, 64)

	openIndex := strings.Index(got, "Open (1)")
	waitingIndex := strings.Index(got, "Waiting (1)")
	doneIndex := strings.Index(got, "Done (1)")
	if openIndex < 0 || waitingIndex < 0 || doneIndex < 0 {
		t.Fatalf("stacked board missing column headers:\n%s", got)
	}
	if !(openIndex < waitingIndex && waitingIndex < doneIndex) {
		t.Fatalf("stacked board headers out of order:\n%s", got)
	}
	if !strings.Contains(got, "Keep the board readable") || !strings.Contains(got, "Wait for external deploy") {
		t.Fatalf("stacked board missing ticket titles:\n%s", got)
	}
}

func TestRenderProjectTicketBoardMarksSelectedTicket(t *testing.T) {
	got := RenderProjectTicketBoardWithSelection(ProjectRow{
		Key:         "sample",
		DisplayName: "Sample",
		Tickets:     2,
		OpenTickets: 2,
		TicketRows: []TicketRow{
			{ID: "T-001", Title: "First ticket", Status: "active"},
			{ID: "T-002", Title: "Second ticket", Status: "active"},
		},
	}, 96, 1, true)

	for _, want := range []string{"> T-002", "- T-001"} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderProjectTicketBoardWithSelection() missing %q in:\n%s", want, got)
		}
	}
}

func TestRenderProjectTicketBoardKeepsSelectedTicketVisible(t *testing.T) {
	tickets := []TicketRow{
		{ID: "T-001", Title: "Ticket 1", Status: "active"},
		{ID: "T-002", Title: "Ticket 2", Status: "active"},
		{ID: "T-003", Title: "Ticket 3", Status: "active"},
		{ID: "T-004", Title: "Ticket 4", Status: "active"},
		{ID: "T-005", Title: "Ticket 5", Status: "active"},
		{ID: "T-006", Title: "Ticket 6", Status: "active"},
		{ID: "T-007", Title: "Ticket 7", Status: "active"},
		{ID: "T-008", Title: "Ticket 8", Status: "active"},
	}
	got := RenderProjectTicketBoardWithSelection(ProjectRow{
		Key:        "sample",
		Tickets:    len(tickets),
		TicketRows: tickets,
	}, 96, 7, true)

	for _, want := range []string{"> T-008", "+ 2 earlier"} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderProjectTicketBoardWithSelection() missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "T-001") {
		t.Fatalf("selected ticket window did not advance:\n%s", got)
	}
}

func TestRenderProjectTicketBoardShowsFocusedTicketDetailPane(t *testing.T) {
	got := RenderProjectTicketBoardWithSelection(ProjectRow{
		Key:         "sample",
		DisplayName: "Sample",
		Tickets:     1,
		OpenTickets: 1,
		TicketRows: []TicketRow{
			{
				ID:          "T-001",
				Title:       "Add focused ticket pane",
				Status:      "active",
				GoalID:      "G-001",
				GoalTitle:   "Make TUI useful",
				Created:     "2026-05-25T19:00:00+02:00",
				Tags:        []string{"tui", "detail"},
				Description: "Show enough ticket context to keep an agent oriented inside the terminal.",
				Acceptance:  []string{"Ticket metadata is visible.", "Action affordances are visible."},
				Owner:       "codex-worker",
				ClaimUntil:  "2026-05-25T21:08:00+02:00",
			},
		},
		EventRows: []EventRow{
			{
				Time:     "2026-05-25T19:08:00+02:00",
				Actor:    "codex-worker",
				Type:     "ticket.claimed",
				ObjectID: "T-001",
				Summary:  "Claimed the focused pane work.",
			},
			{
				Time:     "2026-05-25T19:09:00+02:00",
				Actor:    "codex-worker",
				Type:     "ticket.commented",
				ObjectID: "T-001",
				Summary:  "Added the first detail pane draft.",
			},
		},
	}, 96, 0, true)

	for _, want := range []string{"Tickets Board", "> T-001", "Ticket Detail", "Add focused ticket pane", "state Open", "owner codex-worker", "claim until 2026-05-25 21:08", "G-001 - Make TUI useful", "Ticket metadata is visible.", "ticket.commented", "r refresh | backspace back | up/down move | enter inspect"} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderProjectTicketBoardWithSelection() missing %q in:\n%s", want, got)
		}
	}
}

func TestRenderFocusedTicketPaneStaysCompactAtNarrowWidth(t *testing.T) {
	got := RenderFocusedTicketPane(ProjectRow{
		Key:         "sample",
		DisplayName: "Sample",
	}, TicketRow{
		ID:          "T-001",
		Title:       "Keep the focused ticket detail pane readable on compact terminals",
		Status:      "active",
		Description: "This description is intentionally long so the focused ticket detail pane has to wrap content instead of spilling across compact terminal layouts.",
		Acceptance: []string{
			"The detail pane wraps long acceptance criteria without breaking the board layout.",
			"The action affordances remain visible.",
		},
	}, 48)

	for _, want := range []string{"Ticket Detail", "Description", "Acceptance", "Actions", "compact terminal", "enter inspect"} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderFocusedTicketPane() missing %q in:\n%s", want, got)
		}
	}
	for _, line := range strings.Split(got, "\n") {
		if len(line) > 52 {
			t.Fatalf("narrow detail pane line is too wide (%d): %q\n%s", len(line), line, got)
		}
	}
}

func TestRenderDashboardCanShowTicketMode(t *testing.T) {
	got := RenderDashboardWithSelectionAndMode(Dashboard{Projects: []ProjectRow{
		{
			Key:         "sample",
			DisplayName: "Sample",
			Tickets:     1,
			OpenTickets: 1,
			TicketRows: []TicketRow{
				{ID: "T-001", Title: "Add status table", Status: "active"},
			},
		},
	}}, 0, DetailModeTickets)

	for _, want := range []string{"Tickets Board", "T-001", "project/tickets/timeline"} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderDashboardWithSelectionAndMode() missing %q in:\n%s", want, got)
		}
	}
}

func TestFilterDashboardNarrowsProjectsTicketsAndOwnership(t *testing.T) {
	dashboard := Dashboard{Projects: []ProjectRow{
		{
			Key:         "sample",
			DisplayName: "Sample",
			TicketRows: []TicketRow{
				{ID: "T-001", Title: "Owned active work", Status: "active", Owner: "hilbert"},
				{ID: "T-002", Title: "Waiting review", Status: "blocked"},
			},
		},
		{
			Key:         "second",
			DisplayName: "Second",
			TicketRows: []TicketRow{
				{ID: "T-003", Title: "Completed work", Status: "done", Owner: "boole"},
			},
		},
	}}

	filtered := FilterDashboard(dashboard, DashboardFilters{ProjectKey: "sample", Query: "review", Status: "waiting", Owner: "unowned"})
	if len(filtered.Projects) != 1 || filtered.Projects[0].Key != "sample" {
		t.Fatalf("filtered projects = %+v", filtered.Projects)
	}
	if len(filtered.Projects[0].TicketRows) != 1 || filtered.Projects[0].TicketRows[0].ID != "T-002" {
		t.Fatalf("filtered tickets = %+v", filtered.Projects[0].TicketRows)
	}
	if filtered.Summary.OpenTickets != 1 || filtered.Summary.DoneTickets != 0 || filtered.Summary.Tickets != 1 {
		t.Fatalf("filtered summary = %+v", filtered.Summary)
	}
}

func TestRenderDashboardShowsActiveFilters(t *testing.T) {
	got := RenderDashboardWithFilters(Dashboard{Projects: []ProjectRow{
		{Key: "sample", DisplayName: "Sample", TicketRows: []TicketRow{{ID: "T-001", Title: "Owned active work", Status: "active", Owner: "hilbert"}}},
	}}, 0, DetailModeTickets, 96, 0, false, DashboardFilters{ProjectKey: "sample", Query: "active", Status: "open", Owner: "owned"}, true)

	for _, want := range []string{"Filters:", "project sample", "state open", "owner owned", "query \"active_\"", "c clear"} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderDashboardWithFilters() missing %q in:\n%s", want, got)
		}
	}
}

func TestRenderTimelineShowsRecentEvents(t *testing.T) {
	got := RenderTimeline([]EventRow{
		{
			ID:          "E-older",
			Time:        "2026-05-25T18:00:00+02:00",
			Actor:       "claude",
			ProjectName: "Second",
			Type:        "ticket.commented",
			ObjectID:    "T-010",
			Summary:     "Added implementation notes.",
		},
	})

	for _, want := range []string{"Timeline", "2026-05-25 18:00", "Second", "ticket.commented", "T-010", "claude", "Added implementation notes"} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderTimeline() missing %q in:\n%s", want, got)
		}
	}
}

func TestRenderDashboardCanShowTimelineMode(t *testing.T) {
	got := RenderDashboardWithSelectionAndMode(Dashboard{
		Projects: []ProjectRow{{Key: "sample", DisplayName: "Sample"}},
		Events: []EventRow{
			{
				Time:        "2026-05-25T19:07:00+02:00",
				Actor:       "codex",
				ProjectName: "Sample",
				Type:        "ticket.created",
				ObjectID:    "T-001",
			},
		},
	}, 0, DetailModeTimeline)

	for _, want := range []string{"Timeline", "ticket.created", "T-001", "project/tickets/timeline"} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderDashboardWithSelectionAndMode() missing %q in:\n%s", want, got)
		}
	}
}

func TestRenderDashboardShowsNavigationHelp(t *testing.T) {
	got := RenderDashboardWithSelection(Dashboard{Projects: []ProjectRow{
		{Key: "sample", DisplayName: "Sample", Status: "active"},
	}}, 0)

	for _, want := range []string{"up/down or k/j", "tab/right to switch project/tickets/timeline", "enter to select tickets", "/ query", "s state", "o owner", "p project", "c clear", "q quit"} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderDashboardWithSelection() missing %q in:\n%s", want, got)
		}
	}
}

func TestModelFiltersTicketsWithoutLosingSelectedProject(t *testing.T) {
	model := NewModel(Dashboard{Projects: []ProjectRow{
		{Key: "first", TicketRows: []TicketRow{{ID: "T-001", Title: "First active", Status: "active"}}},
		{Key: "second", TicketRows: []TicketRow{{ID: "T-002", Title: "Second waiting", Status: "blocked", Owner: "hilbert"}}},
	}}, nil)
	model.selected = 1
	model.mode = DetailModeTickets

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	got := updated.(Model)
	if got.filters.ProjectKey != "second" || got.selected != 1 {
		t.Fatalf("project filter did not preserve selected project: filters=%+v selected=%d", got.filters, got.selected)
	}

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	got = updated.(Model)
	if got.filters.Status != "open" {
		t.Fatalf("status filter = %q, want open", got.filters.Status)
	}
	view := got.View()
	if !strings.Contains(view, "No projects found.") || !strings.Contains(view, "Filters:") {
		t.Fatalf("filtered view should show empty filtered state:\n%s", view)
	}

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	got = updated.(Model)
	if got.filters.Status != "waiting" || got.selected != 1 {
		t.Fatalf("waiting filter did not preserve selected project: filters=%+v selected=%d", got.filters, got.selected)
	}
	if !strings.Contains(got.View(), "T-002") {
		t.Fatalf("waiting filter did not show selected project ticket:\n%s", got.View())
	}

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	got = updated.(Model)
	if filtersActive(got.filters) {
		t.Fatalf("clear did not reset filters: %+v", got.filters)
	}
}

func TestModelSelectionMovesWithArrowKeys(t *testing.T) {
	model := NewModel(Dashboard{Projects: []ProjectRow{
		{Key: "sample"},
		{Key: "second"},
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
	model := NewModel(Dashboard{Projects: []ProjectRow{{Key: "sample"}}}, nil)

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
	if got.mode != DetailModeTimeline {
		t.Fatalf("mode after right = %v, want timeline", got.mode)
	}

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyRight})
	got = updated.(Model)
	if got.mode != DetailModeProject {
		t.Fatalf("mode after second right = %v, want project", got.mode)
	}

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyLeft})
	got = updated.(Model)
	if got.mode != DetailModeTimeline {
		t.Fatalf("mode after left = %v, want timeline", got.mode)
	}
}

func TestModelMovesTicketSelectionWhenTicketPaneFocused(t *testing.T) {
	model := NewModel(Dashboard{Projects: []ProjectRow{
		{
			Key: "sample",
			TicketRows: []TicketRow{
				{ID: "T-001", Title: "First"},
				{ID: "T-002", Title: "Second"},
			},
		},
		{
			Key: "second",
			TicketRows: []TicketRow{
				{ID: "T-010", Title: "Other"},
			},
		},
	}}, nil)
	model.mode = DetailModeTickets

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatalf("enter command = %v, want nil", cmd)
	}
	got := updated.(Model)
	if !got.ticketFocus {
		t.Fatal("enter should focus ticket selection")
	}

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyDown})
	got = updated.(Model)
	if got.selected != 0 || got.selectedTicket != 1 {
		t.Fatalf("down with ticket focus selected project=%d ticket=%d, want project 0 ticket 1", got.selected, got.selectedTicket)
	}

	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	got = updated.(Model)
	if got.ticketFocus {
		t.Fatal("backspace should return focus to projects")
	}
	updated, _ = got.Update(tea.KeyMsg{Type: tea.KeyDown})
	got = updated.(Model)
	if got.selected != 1 || got.selectedTicket != 0 {
		t.Fatalf("down with project focus selected project=%d ticket=%d, want project 1 ticket 0", got.selected, got.selectedTicket)
	}
}

func TestModelRefreshPreservesSelectedTicketByID(t *testing.T) {
	model := NewLiveModel("/tmp/maat-state", Dashboard{Projects: []ProjectRow{
		{
			Key: "sample",
			TicketRows: []TicketRow{
				{ID: "T-001", Title: "First"},
				{ID: "T-002", Title: "Second"},
			},
		},
	}}, nil)
	model.mode = DetailModeTickets
	model.ticketFocus = true
	model.selectedTicket = 1

	updated, _ := model.Update(dashboardLoadedMsg{dashboard: Dashboard{Projects: []ProjectRow{
		{
			Key: "sample",
			TicketRows: []TicketRow{
				{ID: "T-000", Title: "New"},
				{ID: "T-002", Title: "Second reloaded"},
				{ID: "T-003", Title: "Third"},
			},
		},
	}}})
	got := updated.(Model)
	if got.selectedTicket != 1 || got.dashboard.Projects[0].TicketRows[got.selectedTicket].Title != "Second reloaded" {
		t.Fatalf("refresh did not preserve selected ticket: selectedTicket=%d tickets=%+v", got.selectedTicket, got.dashboard.Projects[0].TicketRows)
	}
	if !got.ticketFocus {
		t.Fatal("refresh should keep ticket focus when selected project still has tickets")
	}
}

func TestModelQuitKeyReturnsCommand(t *testing.T) {
	model := NewModel(Dashboard{}, nil)
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("quit command is nil")
	}
}

func TestLiveModelStartsRefreshLoop(t *testing.T) {
	model := NewLiveModel("/tmp/maat-state", Dashboard{}, nil)
	if cmd := model.Init(); cmd == nil {
		t.Fatal("expected live model init to schedule refresh")
	}
}

func TestInitialDashboardLoadUsesPullAwareLoaderWhenEnabled(t *testing.T) {
	calls := []string{}
	withPull := func(storage string) dashboardLoadedMsg {
		calls = append(calls, "with:"+storage)
		return dashboardLoadedMsg{dashboard: Dashboard{Projects: []ProjectRow{{Key: "with-pull"}}}}
	}
	withoutPull := func(storage string) dashboardLoadedMsg {
		calls = append(calls, "without:"+storage)
		return dashboardLoadedMsg{dashboard: Dashboard{Projects: []ProjectRow{{Key: "without-pull"}}}}
	}

	loaded := initialDashboardLoaderWithOptions(TUIOptions{AutoPullBeforeRefresh: true}, withPull, withoutPull)("/tmp/maat-state")
	if len(calls) != 1 || calls[0] != "with:/tmp/maat-state" {
		t.Fatalf("initial load calls = %+v, want pull-aware loader", calls)
	}
	if loaded.dashboard.Projects[0].Key != "with-pull" {
		t.Fatalf("initial dashboard = %+v, want pull-aware result", loaded.dashboard)
	}
}

func TestInitialDashboardLoadUsesPlainLoaderWhenPullDisabled(t *testing.T) {
	calls := []string{}
	withPull := func(storage string) dashboardLoadedMsg {
		calls = append(calls, "with:"+storage)
		return dashboardLoadedMsg{dashboard: Dashboard{Projects: []ProjectRow{{Key: "with-pull"}}}}
	}
	withoutPull := func(storage string) dashboardLoadedMsg {
		calls = append(calls, "without:"+storage)
		return dashboardLoadedMsg{dashboard: Dashboard{Projects: []ProjectRow{{Key: "without-pull"}}}}
	}

	loaded := initialDashboardLoaderWithOptions(TUIOptions{}, withPull, withoutPull)("/tmp/maat-state")
	if len(calls) != 1 || calls[0] != "without:/tmp/maat-state" {
		t.Fatalf("initial load calls = %+v, want plain loader", calls)
	}
	if loaded.dashboard.Projects[0].Key != "without-pull" {
		t.Fatalf("initial dashboard = %+v, want plain result", loaded.dashboard)
	}
}

func TestRefreshDashboardLoadUsesPullAwareLoaderWhenEnabled(t *testing.T) {
	model := NewLiveModelWithOptions("/tmp/maat-state", Dashboard{}, nil, TUIOptions{AutoPullBeforeRefresh: true})
	model.load = refreshDashboardLoaderWithOptions(TUIOptions{AutoPullBeforeRefresh: true}, func(storage string) dashboardLoadedMsg {
		return dashboardLoadedMsg{dashboard: Dashboard{Projects: []ProjectRow{{Key: "with-pull"}}}}
	}, func(storage string) dashboardLoadedMsg {
		return dashboardLoadedMsg{dashboard: Dashboard{Projects: []ProjectRow{{Key: "without-pull"}}}}
	})

	msg := model.loadDashboardCmd(3)()
	loaded, ok := msg.(dashboardLoadedMsg)
	if !ok {
		t.Fatalf("unexpected load message: %#v", msg)
	}
	if loaded.requestID != 3 || loaded.dashboard.Projects[0].Key != "with-pull" {
		t.Fatalf("refresh load = %#v, want pull-aware result with request id", loaded)
	}
}

func TestInitialPullWarningKeepsLoadedDashboardVisible(t *testing.T) {
	model := newLiveModelFromInitialLoad("/tmp/maat-state", dashboardLoadedMsg{
		dashboard: Dashboard{Projects: []ProjectRow{{Key: "sample", DisplayName: "Sample"}}},
		warning:   errors.New("git pull failed"),
	}, TUIOptions{AutoPullBeforeRefresh: true})

	if model.err != nil || model.refreshErr == nil {
		t.Fatalf("expected warning-only initial model, got err=%v refreshErr=%v", model.err, model.refreshErr)
	}
	view := model.View()
	for _, want := range []string{"Sample", "Auto-refresh warning", "git pull failed"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q after initial warning:\n%s", want, view)
		}
	}
}

func TestModelRefreshReloadsDashboardAndPreservesSelection(t *testing.T) {
	model := NewLiveModel("/tmp/maat-state", Dashboard{Projects: []ProjectRow{
		{Key: "first"},
		{Key: "second"},
	}}, nil)
	model.selected = 1

	updated, _ := model.Update(dashboardLoadedMsg{dashboard: Dashboard{Projects: []ProjectRow{
		{Key: "new"},
		{Key: "second", DisplayName: "Second Reloaded"},
		{Key: "third"},
	}}})
	got := updated.(Model)
	if got.selected != 1 || got.dashboard.Projects[got.selected].DisplayName != "Second Reloaded" {
		t.Fatalf("refresh did not preserve selected project: selected=%d projects=%+v", got.selected, got.dashboard.Projects)
	}
	if got.err != nil || got.refreshErr != nil {
		t.Fatalf("expected successful refresh to clear errors, got err=%v refreshErr=%v", got.err, got.refreshErr)
	}
}

func TestModelManualReloadStartsImmediatelyAndKeepsDataVisible(t *testing.T) {
	model := NewLiveModel("/tmp/maat-state", Dashboard{Projects: []ProjectRow{
		{Key: "first", DisplayName: "First"},
		{Key: "second", DisplayName: "Second"},
	}}, nil)
	model.selected = 1
	model.mode = DetailModeTickets
	model.load = func(storage string) dashboardLoadedMsg {
		if storage != "/tmp/maat-state" {
			return dashboardLoadedMsg{err: errors.New("unexpected storage")}
		}
		return dashboardLoadedMsg{dashboard: Dashboard{Projects: []ProjectRow{
			{Key: "second", DisplayName: "Second Reloaded"},
			{Key: "third", DisplayName: "Third"},
		}}}
	}

	updated, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd == nil {
		t.Fatal("manual reload command is nil")
	}
	loading := updated.(Model)
	if !loading.refreshing {
		t.Fatal("manual reload did not mark model as refreshing")
	}
	loadingView := loading.View()
	for _, want := range []string{"Second", "Refreshing..."} {
		if !strings.Contains(loadingView, want) {
			t.Fatalf("loading view missing %q:\n%s", want, loadingView)
		}
	}

	msg := cmd()
	loaded, ok := msg.(dashboardLoadedMsg)
	if !ok {
		t.Fatalf("unexpected reload message: %#v", msg)
	}
	if loaded.requestID == 0 {
		t.Fatal("manual reload message is missing request id")
	}
	updated, _ = loading.Update(loaded)
	got := updated.(Model)
	if got.refreshing {
		t.Fatal("successful reload should clear refreshing state")
	}
	if got.mode != DetailModeTickets {
		t.Fatalf("mode after manual reload = %v, want tickets", got.mode)
	}
	if got.selected != 0 || got.dashboard.Projects[got.selected].DisplayName != "Second Reloaded" {
		t.Fatalf("manual reload did not preserve selected project: selected=%d projects=%+v", got.selected, got.dashboard.Projects)
	}
}

func TestModelAutoRefreshTickDoesNotStartOverlappingLoad(t *testing.T) {
	model := NewLiveModel("/tmp/maat-state", Dashboard{Projects: []ProjectRow{{Key: "sample"}}}, nil)
	model.refreshing = true
	model.nextRefreshID = 1
	model.activeRefreshID = 1
	model.load = func(string) dashboardLoadedMsg {
		t.Fatal("in-flight refresh tick started another load")
		return dashboardLoadedMsg{}
	}

	updated, cmd := model.Update(dashboardRefreshTickMsg{})
	got := updated.(Model)
	if !got.refreshing {
		t.Fatal("in-flight refresh tick should keep refreshing state")
	}
	if got.nextRefreshID != 1 || got.activeRefreshID != 1 {
		t.Fatalf("in-flight refresh tick changed request ids: next=%d active=%d", got.nextRefreshID, got.activeRefreshID)
	}
	if cmd == nil {
		t.Fatal("in-flight refresh tick should schedule the next tick")
	}
}

func TestModelIgnoresStaleRefreshResultWhileNewerReloadIsActive(t *testing.T) {
	model := NewLiveModel("/tmp/maat-state", Dashboard{Projects: []ProjectRow{
		{Key: "current", DisplayName: "Current"},
	}}, nil)
	model.refreshing = true
	model.nextRefreshID = 2
	model.activeRefreshID = 2

	updated, _ := model.Update(dashboardLoadedMsg{
		requestID: 1,
		dashboard: Dashboard{Projects: []ProjectRow{
			{Key: "stale", DisplayName: "Stale"},
		}},
	})
	got := updated.(Model)
	if !got.refreshing || got.activeRefreshID != 2 {
		t.Fatalf("stale result changed active reload state: refreshing=%v active=%d", got.refreshing, got.activeRefreshID)
	}
	if got.dashboard.Projects[0].Key != "current" {
		t.Fatalf("stale result clobbered dashboard: %+v", got.dashboard.Projects)
	}
}

func TestModelIgnoresLateRefreshResultAfterNewerSuccess(t *testing.T) {
	model := NewLiveModel("/tmp/maat-state", Dashboard{Projects: []ProjectRow{
		{Key: "current", DisplayName: "Current"},
	}}, nil)
	model.refreshing = true
	model.nextRefreshID = 2
	model.activeRefreshID = 2

	updated, _ := model.Update(dashboardLoadedMsg{
		requestID: 2,
		dashboard: Dashboard{Projects: []ProjectRow{
			{Key: "newer", DisplayName: "Newer"},
		}},
	})
	got := updated.(Model)
	if got.refreshing || got.activeRefreshID != 0 {
		t.Fatalf("newer result did not complete active reload: refreshing=%v active=%d", got.refreshing, got.activeRefreshID)
	}

	updated, _ = got.Update(dashboardLoadedMsg{
		requestID: 1,
		dashboard: Dashboard{Projects: []ProjectRow{
			{Key: "older", DisplayName: "Older"},
		}},
	})
	got = updated.(Model)
	if got.dashboard.Projects[0].Key != "newer" {
		t.Fatalf("late stale result clobbered newer dashboard: %+v", got.dashboard.Projects)
	}
}

func TestModelRefreshErrorKeepsExistingDashboardVisible(t *testing.T) {
	model := NewLiveModel("/tmp/maat-state", Dashboard{Projects: []ProjectRow{
		{Key: "sample", DisplayName: "Sample"},
	}}, nil)

	updated, _ := model.Update(dashboardLoadedMsg{err: errors.New("storage unavailable")})
	got := updated.(Model)
	if len(got.dashboard.Projects) != 1 || got.dashboard.Projects[0].Key != "sample" {
		t.Fatalf("refresh error should keep existing dashboard, got %+v", got.dashboard)
	}
	view := got.View()
	for _, want := range []string{"Sample", "Auto-refresh warning", "storage unavailable"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q after refresh error:\n%s", want, view)
		}
	}
}

func TestModelFailedReloadPreservesSelectionAndMode(t *testing.T) {
	model := NewLiveModel("/tmp/maat-state", Dashboard{Projects: []ProjectRow{
		{Key: "first", DisplayName: "First"},
		{Key: "second", DisplayName: "Second"},
	}}, nil)
	model.selected = 1
	model.mode = DetailModeTimeline
	model.refreshing = true

	updated, _ := model.Update(dashboardLoadedMsg{err: errors.New("storage unavailable")})
	got := updated.(Model)
	if got.refreshing {
		t.Fatal("failed reload should clear refreshing state")
	}
	if got.selected != 1 {
		t.Fatalf("selected after failed reload = %d, want 1", got.selected)
	}
	if got.mode != DetailModeTimeline {
		t.Fatalf("mode after failed reload = %v, want timeline", got.mode)
	}
	if len(got.dashboard.Projects) != 2 || got.dashboard.Projects[got.selected].Key != "second" {
		t.Fatalf("failed reload should keep existing dashboard and selection, got %+v", got.dashboard.Projects)
	}
}

func TestModelLoadCommandUsesConfiguredLoader(t *testing.T) {
	model := NewLiveModel("/tmp/maat-state", Dashboard{}, nil)
	model.load = func(storage string) dashboardLoadedMsg {
		if storage != "/tmp/maat-state" {
			return dashboardLoadedMsg{err: errors.New("unexpected storage")}
		}
		return dashboardLoadedMsg{dashboard: Dashboard{Projects: []ProjectRow{{Key: "loaded"}}}}
	}

	msg := model.loadDashboardCmd(7)()
	loaded, ok := msg.(dashboardLoadedMsg)
	if !ok {
		t.Fatalf("unexpected load message: %#v", msg)
	}
	if loaded.requestID != 7 {
		t.Fatalf("request id = %d, want 7", loaded.requestID)
	}
	if loaded.err != nil || len(loaded.dashboard.Projects) != 1 || loaded.dashboard.Projects[0].Key != "loaded" {
		t.Fatalf("unexpected loaded dashboard: %#v", loaded)
	}
}

func TestModelRefreshWarningKeepsNewDashboardVisible(t *testing.T) {
	model := NewLiveModel("/tmp/maat-state", Dashboard{Projects: []ProjectRow{{Key: "old"}}}, nil)

	updated, _ := model.Update(dashboardLoadedMsg{
		dashboard: Dashboard{Projects: []ProjectRow{{Key: "new", DisplayName: "New"}}},
		warning:   errors.New("git pull failed"),
	})
	got := updated.(Model)
	if got.refreshErr == nil || got.dashboard.Projects[0].Key != "new" {
		t.Fatalf("expected warning with refreshed dashboard, got refreshErr=%v dashboard=%+v", got.refreshErr, got.dashboard)
	}
	view := got.View()
	for _, want := range []string{"New", "Auto-refresh warning", "git pull failed"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view missing %q after warning refresh:\n%s", want, view)
		}
	}
}
