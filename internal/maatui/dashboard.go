package maatui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sunday-studio/maat/internal/maat"
)

type ProjectRow struct {
	Key         string
	DisplayName string
	Status      string
	Summary     string
	Goals       int
	Tickets     int
	OpenTickets int
	DoneTickets int
	Updated     string
	GoalRows    []GoalRow
	TicketRows  []TicketRow
}

type GoalRow struct {
	ID      string
	Title   string
	Status  string
	Tickets int
}

type TicketRow struct {
	ID     string
	Title  string
	Status string
	GoalID string
}

type EventRow struct {
	ID          string
	Time        string
	Actor       string
	ProjectKey  string
	ProjectName string
	Type        string
	ObjectID    string
	Summary     string
}

type Dashboard struct {
	Projects []ProjectRow
	Summary  maat.StatusSummary
	Events   []EventRow
}

type DetailMode int

const (
	DetailModeProject DetailMode = iota
	DetailModeTickets
	DetailModeTimeline
)

type Model struct {
	dashboard  Dashboard
	err        error
	refreshErr error
	refreshing bool
	width      int
	selected   int
	mode       DetailMode
	storage    string
	load       dashboardLoader
}

type dashboardLoader func(string) dashboardLoadedMsg

type TUIOptions struct {
	AutoPullBeforeRefresh bool
}

type dashboardRefreshTickMsg struct{}

type dashboardLoadedMsg struct {
	dashboard Dashboard
	err       error
	warning   error
}

const dashboardRefreshInterval = 5 * time.Second

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99"))
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("63"))
	mutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))
)

func RunTUI(storage string) error {
	return RunTUIWithOptions(storage, TUIOptions{})
}

func RunTUIWithOptions(storage string, options TUIOptions) error {
	dashboard, err := LoadDashboard(storage)
	model := NewLiveModelWithOptions(storage, dashboard, err, options)
	_, runErr := tea.NewProgram(model, tea.WithAltScreen()).Run()
	if runErr != nil {
		return runErr
	}
	return nil
}

func NewModel(dashboard Dashboard, err error) Model {
	return Model{dashboard: dashboard, err: err, load: loadDashboardWithoutPull}
}

func NewLiveModel(storage string, dashboard Dashboard, err error) Model {
	return NewLiveModelWithOptions(storage, dashboard, err, TUIOptions{})
}

func NewLiveModelWithOptions(storage string, dashboard Dashboard, err error, options TUIOptions) Model {
	model := NewModel(dashboard, err)
	model.storage = storage
	if options.AutoPullBeforeRefresh {
		model.load = loadDashboardWithPull
	}
	return model
}

func (m Model) Init() tea.Cmd {
	if m.storage == "" {
		return nil
	}
	return refreshTickCmd()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "up", "k":
			m.selected = clampSelection(m.selected-1, len(m.dashboard.Projects))
		case "down", "j":
			m.selected = clampSelection(m.selected+1, len(m.dashboard.Projects))
		case "tab", "right", "l":
			m.mode = nextDetailMode(m.mode)
		case "left", "h":
			m.mode = previousDetailMode(m.mode)
		case "r":
			return m.startDashboardReload(false)
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case dashboardRefreshTickMsg:
		if m.storage == "" {
			return m, nil
		}
		return m.startDashboardReload(true)
	case dashboardLoadedMsg:
		m = m.withLoadedDashboard(msg)
	}
	return m, nil
}

func (m Model) View() string {
	if m.err != nil && len(m.dashboard.Projects) == 0 {
		return errorStyle.Render(fmt.Sprintf("Maat TUI failed: %v", m.err)) + "\n\n" + mutedStyle.Render("Press q to quit.") + "\n"
	}
	view := RenderDashboardWithSelectionAndMode(m.dashboard, m.selected, m.mode)
	if m.refreshing {
		view += mutedStyle.Render("Refreshing...")
		view += "\n"
	}
	if m.refreshErr != nil {
		view += mutedStyle.Render(fmt.Sprintf("Auto-refresh warning: %v", m.refreshErr))
		view += "\n"
	}
	return view
}

func refreshTickCmd() tea.Cmd {
	return tea.Tick(dashboardRefreshInterval, func(time.Time) tea.Msg {
		return dashboardRefreshTickMsg{}
	})
}

func (m Model) loadDashboardCmd() tea.Cmd {
	load := m.load
	if load == nil {
		load = loadDashboardWithoutPull
	}
	storage := m.storage
	return func() tea.Msg {
		return load(storage)
	}
}

func (m Model) startDashboardReload(includeNextTick bool) (Model, tea.Cmd) {
	if m.storage == "" {
		if includeNextTick {
			return m, refreshTickCmd()
		}
		return m, nil
	}
	if m.refreshing {
		if includeNextTick {
			return m, refreshTickCmd()
		}
		return m, nil
	}
	m.refreshing = true
	cmd := m.loadDashboardCmd()
	if includeNextTick {
		cmd = tea.Batch(cmd, refreshTickCmd())
	}
	return m, cmd
}

func (m Model) withLoadedDashboard(msg dashboardLoadedMsg) Model {
	m.refreshing = false
	if msg.err != nil {
		m.refreshErr = msg.err
		return m
	}
	selectedKey := ""
	if len(m.dashboard.Projects) > 0 {
		selectedKey = selectedProject(m.dashboard.Projects, m.selected).Key
	}
	m.dashboard = msg.dashboard
	m.err = nil
	m.refreshErr = msg.warning
	m.selected = selectedIndexByKey(m.dashboard.Projects, selectedKey, m.selected)
	return m
}

func loadDashboardWithoutPull(storage string) dashboardLoadedMsg {
	dashboard, err := LoadDashboard(storage)
	return dashboardLoadedMsg{dashboard: dashboard, err: err}
}

func loadDashboardWithPull(storage string) dashboardLoadedMsg {
	var warning error
	git := maat.GitSync{Store: storage}
	isRepository, err := git.IsRepository(context.Background())
	if err != nil {
		warning = err
	} else if isRepository {
		if err := git.PullRebase(context.Background()); err != nil {
			warning = err
		}
	}
	dashboard, err := LoadDashboard(storage)
	return dashboardLoadedMsg{dashboard: dashboard, err: err, warning: warning}
}

func LoadDashboard(storage string) (Dashboard, error) {
	objectProjects, err := maat.LoadObjectProjects(storage)
	if err != nil {
		return Dashboard{}, err
	}
	return DashboardFromObjectProjects(objectProjects), nil
}

func DashboardFromObjectProjects(projects []maat.ObjectProject) Dashboard {
	rows := make([]ProjectRow, 0, len(projects))
	events := make([]EventRow, 0)
	var summary maat.StatusSummary
	summary.Projects = len(projects)
	for _, project := range projects {
		goalRows := make([]GoalRow, 0, len(project.Goals))
		ticketRows := make([]TicketRow, 0, len(project.Tickets))
		ticketsByGoal := countObjectTicketsByGoal(project.Tickets)
		openTickets, doneTickets := countObjectTickets(project.Tickets)
		for _, goal := range project.Goals {
			goalRows = append(goalRows, GoalRow{
				ID:      goal.ID,
				Title:   goal.Title,
				Status:  goal.Status,
				Tickets: ticketsByGoal[goal.ID],
			})
		}
		for _, ticket := range project.Tickets {
			ticketRows = append(ticketRows, TicketRow{
				ID:     ticket.ID,
				Title:  ticket.Title,
				Status: ticket.Status,
				GoalID: ticket.GoalID,
			})
		}
		row := ProjectRow{
			Key:         project.Key,
			DisplayName: project.DisplayName,
			Status:      project.Status,
			Summary:     project.Summary,
			Goals:       len(project.Goals),
			Tickets:     len(project.Tickets),
			OpenTickets: openTickets,
			DoneTickets: doneTickets,
			Updated:     project.Updated,
			GoalRows:    goalRows,
			TicketRows:  ticketRows,
		}
		rows = append(rows, row)
		summary.Goals += len(project.Goals)
		summary.Tickets += len(project.Tickets)
		for _, goal := range project.Goals {
			if goal.Status == "active" {
				summary.ActiveGoals++
			}
			if goal.Status == "done" {
				summary.DoneGoals++
			}
		}
		for _, ticket := range project.Tickets {
			if ticket.Status == "done" {
				summary.DoneTickets++
			} else {
				summary.OpenTickets++
			}
		}
		for _, event := range project.Events {
			events = append(events, EventRow{
				ID:          event.ID,
				Time:        event.Time,
				Actor:       event.Actor,
				ProjectKey:  event.ProjectKey,
				ProjectName: project.DisplayName,
				Type:        event.Type,
				ObjectID:    event.ObjectID,
				Summary:     event.Summary,
			})
		}
	}
	sort.Slice(events, func(i, j int) bool {
		if events[i].Time == events[j].Time {
			return events[i].ID > events[j].ID
		}
		return events[i].Time > events[j].Time
	})
	return Dashboard{Projects: rows, Summary: summary, Events: events}
}

func RenderDashboard(dashboard Dashboard) string {
	return RenderDashboardWithSelection(dashboard, 0)
}

func RenderDashboardWithSelection(dashboard Dashboard, selected int) string {
	return RenderDashboardWithSelectionAndMode(dashboard, selected, DetailModeProject)
}

func RenderDashboardWithSelectionAndMode(dashboard Dashboard, selected int, mode DetailMode) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Maat"))
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("Git-backed project memory"))
	b.WriteString("\n\n")
	b.WriteString(RenderSummary(dashboard.Summary))
	b.WriteString("\n\n")
	b.WriteString(RenderSelectableProjectTable(dashboard.Projects, selected))
	b.WriteString("\n\n")
	project := selectedProject(dashboard.Projects, selected)
	switch mode {
	case DetailModeTickets:
		b.WriteString(RenderProjectTickets(project))
	case DetailModeTimeline:
		b.WriteString(RenderTimeline(dashboard.Events))
	default:
		b.WriteString(RenderProjectDetail(project))
	}
	b.WriteString("\n\n")
	b.WriteString(mutedStyle.Render("Use up/down or k/j to select, tab/right to switch project/tickets/timeline, left to go back, r to reload, q to quit."))
	b.WriteString("\n")
	return b.String()
}

func RenderSummary(summary maat.StatusSummary) string {
	return fmt.Sprintf(
		"Projects: %d  Goals: %d active / %d done / %d total  Tickets: %d open / %d done / %d total",
		summary.Projects,
		summary.ActiveGoals,
		summary.DoneGoals,
		summary.Goals,
		summary.OpenTickets,
		summary.DoneTickets,
		summary.Tickets,
	)
}

func RenderProjectTable(projects []ProjectRow) string {
	return RenderSelectableProjectTable(projects, -1)
}

func RenderSelectableProjectTable(projects []ProjectRow, selected int) string {
	if len(projects) == 0 {
		return mutedStyle.Render("No projects found.")
	}

	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("  %-16s %-10s %7s %8s %s", "Project", "Status", "Goals", "Tickets", "Updated")))
	b.WriteString("\n")
	for index, project := range projects {
		name := project.DisplayName
		if name == "" {
			name = project.Key
		}
		prefix := " "
		if selected == index {
			prefix = ">"
		}
		b.WriteString(fmt.Sprintf("%s %-16s %-10s %7d %8d %s\n", prefix, truncate(name, 16), project.Status, project.Goals, project.Tickets, project.Updated))
	}
	return strings.TrimRight(b.String(), "\n")
}

func RenderProjectDetail(project ProjectRow) string {
	if project.Key == "" && project.DisplayName == "" {
		return mutedStyle.Render("Select a project to see details.")
	}

	name := project.DisplayName
	if name == "" {
		name = project.Key
	}

	var b strings.Builder
	b.WriteString(headerStyle.Render("Project Detail"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("%s  %s  updated %s\n", titleStyle.Render(name), project.Status, emptyFallback(project.Updated, "unknown")))
	b.WriteString(fmt.Sprintf("Tickets: %d open / %d done / %d total\n", project.OpenTickets, project.DoneTickets, project.Tickets))
	b.WriteString("\n")
	b.WriteString(headerStyle.Render("Summary"))
	b.WriteString("\n")
	b.WriteString(emptyFallback(project.Summary, "No summary recorded."))
	b.WriteString("\n\n")
	b.WriteString(headerStyle.Render("Goals"))
	b.WriteString("\n")
	if len(project.GoalRows) == 0 {
		b.WriteString(mutedStyle.Render("No goals recorded."))
		return b.String()
	}
	for index, goal := range project.GoalRows {
		if index >= 6 {
			b.WriteString(mutedStyle.Render(fmt.Sprintf("+ %d more goals", len(project.GoalRows)-index)))
			break
		}
		label := goal.ID
		if label == "" {
			label = "goal"
		}
		b.WriteString(fmt.Sprintf("- %s [%s] %s (%d tickets)\n", label, goal.Status, truncate(goal.Title, 48), goal.Tickets))
	}
	return strings.TrimRight(b.String(), "\n")
}

func RenderProjectTickets(project ProjectRow) string {
	if project.Key == "" && project.DisplayName == "" {
		return mutedStyle.Render("Select a project to see tickets.")
	}

	name := project.DisplayName
	if name == "" {
		name = project.Key
	}

	var b strings.Builder
	b.WriteString(headerStyle.Render("Tickets"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("%s  %d open / %d done / %d total\n", titleStyle.Render(name), project.OpenTickets, project.DoneTickets, project.Tickets))
	b.WriteString("\n")
	if len(project.TicketRows) == 0 {
		b.WriteString(mutedStyle.Render("No tickets recorded."))
		return b.String()
	}
	for index, ticket := range project.TicketRows {
		if index >= 10 {
			b.WriteString(mutedStyle.Render(fmt.Sprintf("+ %d more tickets", len(project.TicketRows)-index)))
			break
		}
		label := ticket.ID
		if label == "" {
			label = "ticket"
		}
		goal := "standalone"
		if ticket.GoalID != "" {
			goal = ticket.GoalID
		}
		b.WriteString(fmt.Sprintf("- %s [%s] %s (%s)\n", label, ticket.Status, truncate(ticket.Title, 52), goal))
	}
	return strings.TrimRight(b.String(), "\n")
}

func RenderTimeline(events []EventRow) string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("Timeline"))
	b.WriteString("\n")
	if len(events) == 0 {
		b.WriteString(mutedStyle.Render("No events recorded."))
		return b.String()
	}
	for index, event := range events {
		if index >= 12 {
			b.WriteString(mutedStyle.Render(fmt.Sprintf("+ %d more events", len(events)-index)))
			break
		}
		project := event.ProjectName
		if project == "" {
			project = event.ProjectKey
		}
		if project == "" {
			project = "project"
		}
		object := event.ObjectID
		if object == "" {
			object = "object"
		}
		summary := event.Summary
		if summary == "" {
			summary = event.Type
		}
		b.WriteString(fmt.Sprintf("- %s %s %s %s by %s: %s\n", compactTime(event.Time), truncate(project, 16), event.Type, object, emptyFallback(event.Actor, "unknown"), truncate(summary, 56)))
	}
	return strings.TrimRight(b.String(), "\n")
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	if limit <= 3 {
		return value[:limit]
	}
	return value[:limit-3] + "..."
}

func clampSelection(selected, count int) int {
	if count <= 0 {
		return 0
	}
	if selected < 0 {
		return 0
	}
	if selected >= count {
		return count - 1
	}
	return selected
}

func selectedProject(projects []ProjectRow, selected int) ProjectRow {
	if len(projects) == 0 {
		return ProjectRow{}
	}
	return projects[clampSelection(selected, len(projects))]
}

func selectedIndexByKey(projects []ProjectRow, key string, fallback int) int {
	if key != "" {
		for index, project := range projects {
			if project.Key == key {
				return index
			}
		}
	}
	return clampSelection(fallback, len(projects))
}

func nextDetailMode(mode DetailMode) DetailMode {
	if mode == DetailModeTimeline {
		return DetailModeProject
	}
	return mode + 1
}

func previousDetailMode(mode DetailMode) DetailMode {
	if mode == DetailModeProject {
		return DetailModeTimeline
	}
	return mode - 1
}

func countObjectTickets(tickets []maat.ObjectTicket) (int, int) {
	openTickets := 0
	doneTickets := 0
	for _, ticket := range tickets {
		if ticket.Status == "done" {
			doneTickets++
		} else {
			openTickets++
		}
	}
	return openTickets, doneTickets
}

func countObjectTicketsByGoal(tickets []maat.ObjectTicket) map[string]int {
	counts := map[string]int{}
	for _, ticket := range tickets {
		if ticket.GoalID == "" {
			continue
		}
		counts[ticket.GoalID]++
	}
	return counts
}

func emptyFallback(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func compactTime(value string) string {
	if len(value) >= len("2006-01-02T15:04") {
		return strings.Replace(value[:len("2006-01-02T15:04")], "T", " ", 1)
	}
	return emptyFallback(value, "unknown time")
}
