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
				{ID: "T-001", Title: "Add status table", Status: "active", GoalID: "G-001"},
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
	if len(dashboard.Events) != 1 || dashboard.Events[0].ProjectName != "Sample" || dashboard.Events[0].Type != "ticket.created" {
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

	for _, want := range []string{"Tickets", "Sample", "1 open / 1 done / 2 total", "T-001", "Add status table", "G-001", "T-002", "standalone"} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderProjectTickets() missing %q in:\n%s", want, got)
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

	for _, want := range []string{"Tickets", "T-001", "project/tickets/timeline"} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderDashboardWithSelectionAndMode() missing %q in:\n%s", want, got)
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

	for _, want := range []string{"up/down or k/j", "tab/right to switch project/tickets/timeline", "left to go back", "q to quit"} {
		if !strings.Contains(got, want) {
			t.Fatalf("RenderDashboardWithSelection() missing %q in:\n%s", want, got)
		}
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

	msg := model.loadDashboardCmd()()
	loaded, ok := msg.(dashboardLoadedMsg)
	if !ok {
		t.Fatalf("unexpected load message: %#v", msg)
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
