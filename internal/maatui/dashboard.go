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
	EventRows   []EventRow
}

type GoalRow struct {
	ID      string
	Title   string
	Status  string
	Tickets int
}

type TicketRow struct {
	ID          string
	Title       string
	Status      string
	GoalID      string
	GoalTitle   string
	ProjectKey  string
	Created     string
	Tags        []string
	Description string
	Acceptance  []string
	Owner       string
	ClaimUntil  string
}

type EventRow struct {
	ID          string
	Time        string
	Actor       string
	ProjectKey  string
	ProjectName string
	Type        string
	ObjectID    string
	Expires     string
	Summary     string
}

type Dashboard struct {
	Projects []ProjectRow
	Summary  maat.StatusSummary
	Events   []EventRow
}

type DashboardFilters struct {
	ProjectKey string
	Query      string
	Status     string
	Owner      string
}

type DetailMode int

const (
	DetailModeProject DetailMode = iota
	DetailModeTickets
	DetailModeTimeline
)

type Model struct {
	dashboard       Dashboard
	err             error
	refreshErr      error
	refreshing      bool
	nextRefreshID   int
	activeRefreshID int
	width           int
	selected        int
	selectedTicket  int
	ticketFocus     bool
	filters         DashboardFilters
	filterEditing   bool
	mode            DetailMode
	storage         string
	load            dashboardLoader
}

type dashboardLoader func(string) dashboardLoadedMsg

type TUIOptions struct {
	AutoPullBeforeRefresh bool
}

type dashboardRefreshTickMsg struct{}

type dashboardLoadedMsg struct {
	requestID int
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
	openTicketStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))
	waitingTicketStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("214"))
	doneTicketStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("42"))
	mutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))
)

func RunTUI(storage string) error {
	return RunTUIWithOptions(storage, TUIOptions{})
}

func RunTUIWithOptions(storage string, options TUIOptions) error {
	loaded := loadInitialDashboard(storage, options)
	model := newLiveModelFromInitialLoad(storage, loaded, options)
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
	model.load = refreshDashboardLoader(options)
	return model
}

func newLiveModelFromInitialLoad(storage string, loaded dashboardLoadedMsg, options TUIOptions) Model {
	model := NewLiveModelWithOptions(storage, loaded.dashboard, loaded.err, options)
	model.refreshErr = loaded.warning
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
		if m.filterEditing && msg.String() != "ctrl+c" {
			m = m.updateFilterQuery(msg)
			return m, nil
		}
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "up", "k":
			m = m.moveSelection(-1)
		case "down", "j":
			m = m.moveSelection(1)
		case "tab", "right", "l":
			m.mode = nextDetailMode(m.mode)
			m.ticketFocus = false
		case "left", "h":
			m.mode = previousDetailMode(m.mode)
			m.ticketFocus = false
		case "enter":
			filtered := FilterDashboard(m.dashboard, m.filters)
			selected := selectedIndexByKey(filtered.Projects, selectedProject(m.dashboard.Projects, m.selected).Key, m.selected)
			if m.mode == DetailModeProject && len(filtered.Projects) > 0 {
				m.mode = DetailModeTickets
				m.ticketFocus = false
				m.selectedTicket = clampSelection(m.selectedTicket, len(selectedProject(filtered.Projects, selected).TicketRows))
			} else if m.mode == DetailModeTickets && len(selectedProject(filtered.Projects, selected).TicketRows) > 0 {
				m.ticketFocus = true
				m.selectedTicket = clampSelection(m.selectedTicket, len(selectedProject(filtered.Projects, selected).TicketRows))
			}
		case "backspace":
			if m.mode == DetailModeTickets && m.ticketFocus {
				m.ticketFocus = false
			} else if m.mode == DetailModeTickets {
				m.mode = DetailModeProject
			} else {
				m.ticketFocus = false
			}
		case "r":
			return m.startDashboardReload(false)
		case "/":
			m.filterEditing = true
		case "c":
			m.filters = DashboardFilters{}
			m.filterEditing = false
			m = m.withFilterSelection()
		case "s":
			m.filters.Status = nextStatusFilter(m.filters.Status)
			m = m.withFilterSelection()
		case "o":
			m.filters.Owner = nextOwnerFilter(m.filters.Owner)
			m = m.withFilterSelection()
		case "p":
			if m.filters.ProjectKey != "" {
				m.filters.ProjectKey = ""
			} else {
				m.filters.ProjectKey = selectedProject(m.dashboard.Projects, m.selected).Key
			}
			m = m.withFilterSelection()
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
		return RenderDashboardError(m.err, m.storage)
	}
	filtered := FilterDashboard(m.dashboard, m.filters)
	selected := selectedIndexByKey(filtered.Projects, selectedProject(m.dashboard.Projects, m.selected).Key, m.selected)
	view := RenderDashboardWithFilters(filtered, selected, m.mode, m.width, m.selectedTicket, m.ticketFocus, m.filters, m.filterEditing)
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

func (m Model) loadDashboardCmd(requestID int) tea.Cmd {
	load := m.load
	if load == nil {
		load = loadDashboardWithoutPull
	}
	storage := m.storage
	return func() tea.Msg {
		msg := load(storage)
		msg.requestID = requestID
		return msg
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
	m.nextRefreshID++
	m.activeRefreshID = m.nextRefreshID
	m.refreshing = true
	cmd := m.loadDashboardCmd(m.activeRefreshID)
	if includeNextTick {
		cmd = tea.Batch(cmd, refreshTickCmd())
	}
	return m, cmd
}

func (m Model) withLoadedDashboard(msg dashboardLoadedMsg) Model {
	if msg.requestID != 0 && msg.requestID != m.activeRefreshID {
		return m
	}
	m.refreshing = false
	m.activeRefreshID = 0
	if msg.err != nil {
		m.refreshErr = msg.err
		return m
	}
	selectedKey := ""
	selectedTicketID := ""
	filtered := FilterDashboard(m.dashboard, m.filters)
	if len(filtered.Projects) > 0 {
		selectedIndex := selectedIndexByKey(filtered.Projects, selectedProject(m.dashboard.Projects, m.selected).Key, m.selected)
		project := selectedProject(filtered.Projects, selectedIndex)
		selectedKey = project.Key
		selectedTicketID = selectedTicket(project.TicketRows, m.selectedTicket).ID
	}
	m.dashboard = msg.dashboard
	m.err = nil
	m.refreshErr = msg.warning
	m.selected = selectedIndexByKey(m.dashboard.Projects, selectedKey, m.selected)
	filtered = FilterDashboard(m.dashboard, m.filters)
	selectedIndex := selectedIndexByKey(filtered.Projects, selectedProject(m.dashboard.Projects, m.selected).Key, m.selected)
	project := selectedProject(filtered.Projects, selectedIndex)
	m.selectedTicket = selectedTicketIndexByID(project.TicketRows, selectedTicketID, m.selectedTicket)
	if len(project.TicketRows) == 0 || m.mode != DetailModeTickets {
		m.ticketFocus = false
	}
	return m
}

func (m Model) moveSelection(delta int) Model {
	filtered := FilterDashboard(m.dashboard, m.filters)
	selectedKey := selectedProject(m.dashboard.Projects, m.selected).Key
	selectedIndex := selectedIndexByKey(filtered.Projects, selectedKey, m.selected)
	if m.mode == DetailModeTickets {
		project := selectedProject(filtered.Projects, selectedIndex)
		m.selectedTicket = clampSelection(m.selectedTicket+delta, len(project.TicketRows))
		return m
	}
	selectedIndex = clampSelection(selectedIndex+delta, len(filtered.Projects))
	project := selectedProject(filtered.Projects, selectedIndex)
	m.selected = selectedIndexByKey(m.dashboard.Projects, project.Key, m.selected)
	m.selectedTicket = clampSelection(m.selectedTicket, len(project.TicketRows))
	if len(project.TicketRows) == 0 {
		m.ticketFocus = false
	}
	return m
}

func (m Model) withFilterSelection() Model {
	filtered := FilterDashboard(m.dashboard, m.filters)
	selectedKey := selectedProject(m.dashboard.Projects, m.selected).Key
	selectedIndex := selectedIndexByKey(filtered.Projects, selectedKey, m.selected)
	project := selectedProject(filtered.Projects, selectedIndex)
	if project.Key != "" {
		m.selected = selectedIndexByKey(m.dashboard.Projects, project.Key, m.selected)
	}
	m.selectedTicket = clampSelection(m.selectedTicket, len(project.TicketRows))
	if len(project.TicketRows) == 0 {
		m.ticketFocus = false
	}
	return m
}

func (m Model) updateFilterQuery(msg tea.KeyMsg) Model {
	switch msg.String() {
	case "enter", "esc":
		m.filterEditing = false
	case "backspace":
		runes := []rune(m.filters.Query)
		if len(runes) > 0 {
			m.filters.Query = string(runes[:len(runes)-1])
		}
	case "ctrl+u":
		m.filters.Query = ""
	default:
		if len(msg.Runes) > 0 {
			m.filters.Query += string(msg.Runes)
		}
	}
	return m.withFilterSelection()
}

func loadDashboardWithoutPull(storage string) dashboardLoadedMsg {
	dashboard, err := LoadDashboard(storage)
	return dashboardLoadedMsg{dashboard: dashboard, err: err}
}

func loadInitialDashboard(storage string, options TUIOptions) dashboardLoadedMsg {
	return initialDashboardLoader(options)(storage)
}

func refreshDashboardLoader(options TUIOptions) dashboardLoader {
	return refreshDashboardLoaderWithOptions(options, loadDashboardWithPull, loadDashboardWithoutPull)
}

func initialDashboardLoader(options TUIOptions) dashboardLoader {
	return initialDashboardLoaderWithOptions(options, loadDashboardWithPull, loadDashboardWithoutPull)
}

func refreshDashboardLoaderWithOptions(options TUIOptions, withPull dashboardLoader, withoutPull dashboardLoader) dashboardLoader {
	return selectDashboardLoader(options, withPull, withoutPull)
}

func initialDashboardLoaderWithOptions(options TUIOptions, withPull dashboardLoader, withoutPull dashboardLoader) dashboardLoader {
	return selectDashboardLoader(options, withPull, withoutPull)
}

func selectDashboardLoader(options TUIOptions, withPull dashboardLoader, withoutPull dashboardLoader) dashboardLoader {
	if options.AutoPullBeforeRefresh {
		return withPull
	}
	return withoutPull
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
		eventRows := make([]EventRow, 0, len(project.Events))
		ticketsByGoal := countObjectTicketsByGoal(project.Tickets)
		goalsByID := objectGoalsByID(project.Goals)
		openTickets, doneTickets := countObjectTickets(project.Tickets)
		for _, event := range project.Events {
			eventRows = append(eventRows, eventRowFromObject(project, event))
		}
		eventRows = sortedEventRows(eventRows)
		claims := currentTicketClaims(eventRows)
		for _, goal := range project.Goals {
			goalRows = append(goalRows, GoalRow{
				ID:      goal.ID,
				Title:   goal.Title,
				Status:  goal.Status,
				Tickets: ticketsByGoal[goal.ID],
			})
		}
		for _, ticket := range project.Tickets {
			goalTitle := ""
			if goal, ok := goalsByID[ticket.GoalID]; ok {
				goalTitle = goal.Title
			}
			claim := claims[ticket.ID]
			ticketRows = append(ticketRows, TicketRow{
				ID:          ticket.ID,
				Title:       ticket.Title,
				Status:      ticket.Status,
				GoalID:      ticket.GoalID,
				GoalTitle:   goalTitle,
				ProjectKey:  ticket.ProjectKey,
				Created:     ticket.Created,
				Tags:        ticket.Tags,
				Description: ticket.Description,
				Acceptance:  ticket.Acceptance,
				Owner:       claim.Owner,
				ClaimUntil:  claim.Expires,
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
			EventRows:   eventRows,
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
			events = append(events, eventRowFromObject(project, event))
		}
	}
	return Dashboard{Projects: rows, Summary: summary, Events: sortedEventRows(events)}
}

func FilterDashboard(dashboard Dashboard, filters DashboardFilters) Dashboard {
	filters = normalizeFilters(filters)
	if !filtersActive(filters) {
		return dashboard
	}
	rows := make([]ProjectRow, 0, len(dashboard.Projects))
	for _, project := range dashboard.Projects {
		if filters.ProjectKey != "" && project.Key != filters.ProjectKey {
			continue
		}
		projectMatchesQuery := filters.Query == "" || projectMatchesFilterQuery(project, filters.Query)
		tickets := make([]TicketRow, 0, len(project.TicketRows))
		for _, ticket := range project.TicketRows {
			if !ticketMatchesFilters(ticket, filters) {
				continue
			}
			tickets = append(tickets, ticket)
		}
		if filters.Query != "" && !projectMatchesQuery && len(tickets) == 0 {
			continue
		}
		if (filters.Status != "" || filters.Owner != "") && len(tickets) == 0 {
			continue
		}
		row := project
		row.TicketRows = tickets
		row.Tickets = len(tickets)
		row.OpenTickets, row.DoneTickets = countTicketRows(tickets)
		rows = append(rows, row)
	}
	return Dashboard{Projects: rows, Summary: summarizeProjectRows(rows), Events: filterEventRows(dashboard.Events, rows)}
}

func normalizeFilters(filters DashboardFilters) DashboardFilters {
	filters.ProjectKey = strings.TrimSpace(filters.ProjectKey)
	filters.Query = strings.TrimSpace(filters.Query)
	filters.Status = strings.TrimSpace(strings.ToLower(filters.Status))
	filters.Owner = strings.TrimSpace(strings.ToLower(filters.Owner))
	return filters
}

func filtersActive(filters DashboardFilters) bool {
	filters = normalizeFilters(filters)
	return filters.ProjectKey != "" || filters.Query != "" || filters.Status != "" || filters.Owner != ""
}

func ticketMatchesFilters(ticket TicketRow, filters DashboardFilters) bool {
	if filters.Query != "" && !ticketMatchesFilterQuery(ticket, filters.Query) {
		return false
	}
	if filters.Status != "" && ticketBoardStatus(ticket.Status) != filters.Status {
		return false
	}
	if filters.Owner == "owned" && strings.TrimSpace(ticket.Owner) == "" {
		return false
	}
	if filters.Owner == "unowned" && strings.TrimSpace(ticket.Owner) != "" {
		return false
	}
	return true
}

func projectMatchesFilterQuery(project ProjectRow, query string) bool {
	haystack := strings.Join([]string{
		project.Key,
		project.DisplayName,
		project.Status,
		project.Summary,
	}, " ")
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(query))
}

func ticketMatchesFilterQuery(ticket TicketRow, query string) bool {
	haystack := strings.Join([]string{
		ticket.ID,
		ticket.Title,
		ticket.Status,
		ticket.GoalID,
		ticket.GoalTitle,
		ticket.ProjectKey,
		ticket.Description,
		strings.Join(ticket.Tags, " "),
		strings.Join(ticket.Acceptance, " "),
		ticket.Owner,
	}, " ")
	return strings.Contains(strings.ToLower(haystack), strings.ToLower(query))
}

func countTicketRows(tickets []TicketRow) (int, int) {
	openTickets := 0
	doneTickets := 0
	for _, ticket := range tickets {
		if ticketBoardStatus(ticket.Status) == "done" {
			doneTickets++
		} else {
			openTickets++
		}
	}
	return openTickets, doneTickets
}

func summarizeProjectRows(projects []ProjectRow) maat.StatusSummary {
	summary := maat.StatusSummary{Projects: len(projects)}
	for _, project := range projects {
		summary.Goals += len(project.GoalRows)
		summary.Tickets += len(project.TicketRows)
		summary.OpenTickets += project.OpenTickets
		summary.DoneTickets += project.DoneTickets
		for _, goal := range project.GoalRows {
			if goal.Status == "active" {
				summary.ActiveGoals++
			}
			if goal.Status == "done" {
				summary.DoneGoals++
			}
		}
	}
	return summary
}

func filterEventRows(events []EventRow, projects []ProjectRow) []EventRow {
	projectKeys := map[string]struct{}{}
	ticketIDs := map[string]struct{}{}
	for _, project := range projects {
		projectKeys[project.Key] = struct{}{}
		for _, ticket := range project.TicketRows {
			ticketIDs[ticket.ID] = struct{}{}
		}
	}
	filtered := make([]EventRow, 0, len(events))
	for _, event := range events {
		if _, ok := ticketIDs[event.ObjectID]; ok {
			filtered = append(filtered, event)
			continue
		}
		if _, ok := projectKeys[event.ProjectKey]; ok && event.ObjectID == event.ProjectKey {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

func RenderDashboard(dashboard Dashboard) string {
	return RenderDashboardWithSelection(dashboard, 0)
}

func RenderDashboardWithSelection(dashboard Dashboard, selected int) string {
	return RenderDashboardWithSelectionAndMode(dashboard, selected, DetailModeProject)
}

func RenderDashboardWithSelectionAndMode(dashboard Dashboard, selected int, mode DetailMode) string {
	return RenderDashboardWithSelectionModeAndWidth(dashboard, selected, mode, 0)
}

func RenderDashboardWithSelectionModeAndWidth(dashboard Dashboard, selected int, mode DetailMode, width int) string {
	return RenderDashboardWithState(dashboard, selected, mode, width, -1, false)
}

func RenderDashboardWithState(dashboard Dashboard, selected int, mode DetailMode, width int, selectedTicket int, ticketFocus bool) string {
	return RenderDashboardWithFilters(dashboard, selected, mode, width, selectedTicket, ticketFocus, DashboardFilters{}, false)
}

func RenderDashboardWithFilters(dashboard Dashboard, selected int, mode DetailMode, width int, selectedTicket int, ticketFocus bool, filters DashboardFilters, editingFilter bool) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Maat"))
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("Git-backed project memory"))
	b.WriteString("\n\n")
	b.WriteString(RenderSummary(dashboard.Summary))
	b.WriteString("\n\n")
	if filtersActive(filters) || editingFilter {
		b.WriteString(RenderFilterBar(filters, editingFilter))
		b.WriteString("\n\n")
	}
	if len(dashboard.Projects) == 0 {
		b.WriteString(RenderEmptyDashboard(filters))
		b.WriteString("\n\n")
		b.WriteString(mutedStyle.Render("Use / query, s state, o owner, p project, c clear, r reload, q quit."))
		b.WriteString("\n")
		return b.String()
	}
	b.WriteString(RenderSelectableProjectTable(dashboard.Projects, selected))
	b.WriteString("\n\n")
	project := selectedProject(dashboard.Projects, selected)
	switch mode {
	case DetailModeTickets:
		b.WriteString(RenderProjectTicketBoardWithSelection(project, width, selectedTicket, ticketFocus))
	case DetailModeTimeline:
		b.WriteString(RenderTimeline(dashboard.Events))
	default:
		b.WriteString(RenderProjectDetail(project))
	}
	b.WriteString("\n\n")
	b.WriteString(mutedStyle.Render("Use up/down or k/j to select, enter to open project board/detail, backspace back, tab/right for timeline, / query, s state, o owner, p project, c clear, r reload, q quit."))
	b.WriteString("\n")
	return b.String()
}

func RenderFilterBar(filters DashboardFilters, editing bool) string {
	parts := []string{}
	if filters.ProjectKey != "" {
		parts = append(parts, "project "+filters.ProjectKey)
	}
	if filters.Status != "" {
		parts = append(parts, "state "+filters.Status)
	}
	if filters.Owner != "" {
		parts = append(parts, "owner "+filters.Owner)
	}
	query := filters.Query
	if editing {
		query += "_"
	}
	if strings.TrimSpace(query) != "" {
		parts = append(parts, fmt.Sprintf("query %q", query))
	}
	if len(parts) == 0 {
		parts = append(parts, "none")
	}
	return mutedStyle.Render("Filters: " + strings.Join(parts, " | ") + "  (c clear)")
}

func RenderEmptyDashboard(filters DashboardFilters) string {
	if filtersActive(filters) {
		return strings.Join([]string{
			headerStyle.Render("No Matching Projects"),
			"No projects or tickets match the active filters.",
			"Press c to clear filters, / to edit the query, or r to reload storage.",
		}, "\n")
	}
	return strings.Join([]string{
		headerStyle.Render("No Projects"),
		"No Maat projects were found in this storage directory.",
		"Run maat project create, check the configured storage path, or press r to reload.",
	}, "\n")
}

func RenderDashboardError(err error, storage string) string {
	lines := []string{
		errorStyle.Render(fmt.Sprintf("Maat TUI failed: %v", err)),
		"",
		"Check the storage path, run maat validate, or press r after fixing the issue.",
	}
	if strings.TrimSpace(storage) != "" {
		lines = append(lines, "Storage: "+storage)
	}
	lines = append(lines, "", mutedStyle.Render("Press q to quit."))
	return strings.Join(lines, "\n") + "\n"
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
		b.WriteString(mutedStyle.Render("No goals recorded. Create one with maat goal create or switch to tickets/timeline."))
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
	return RenderProjectTicketBoard(project, 0)
}

func RenderProjectTicketBoard(project ProjectRow, width int) string {
	return RenderProjectTicketBoardWithSelection(project, width, -1, false)
}

func RenderProjectTicketBoardWithSelection(project ProjectRow, width int, selected int, focused bool) string {
	if project.Key == "" && project.DisplayName == "" {
		return mutedStyle.Render("Select a project to see tickets.")
	}

	name := project.DisplayName
	if name == "" {
		name = project.Key
	}

	var b strings.Builder
	b.WriteString(headerStyle.Render("Tickets Board"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("%s  %s  %d open / %d done / %d total\n", titleStyle.Render(name), emptyFallback(project.Status, "status unknown"), project.OpenTickets, project.DoneTickets, project.Tickets))
	b.WriteString("\n")
	if len(project.TicketRows) == 0 {
		b.WriteString(mutedStyle.Render("No tickets recorded. Create one with maat ticket create, adjust filters, or press r to reload."))
		return b.String()
	}
	hasSelection := selected >= 0
	selected = clampSelection(selected, len(project.TicketRows))
	ticket := selectedTicket(project.TicketRows, selected)
	selectedID := ""
	if hasSelection {
		selectedID = ticket.ID
	}
	columns := ticketBoardColumns(project.TicketRows)
	if width <= 0 {
		width = 140
	}
	if width < 72 {
		b.WriteString(renderStackedTicketBoard(columns, 56, selectedID))
		if focused {
			b.WriteString("\n")
			b.WriteString(RenderFocusedTicketPane(project, ticket, width))
		}
		return strings.TrimRight(b.String(), "\n")
	}
	b.WriteString(renderColumnTicketBoard(columns, width, selectedID))
	if focused {
		b.WriteString("\n")
		b.WriteString(RenderFocusedTicketPane(project, ticket, width))
	}
	return strings.TrimRight(b.String(), "\n")
}

func RenderFocusedTicketPane(project ProjectRow, ticket TicketRow, width int) string {
	if ticket.ID == "" && ticket.Title == "" {
		return mutedStyle.Render("Select a ticket to see details.")
	}
	contentWidth := detailContentWidth(width)
	projectName := project.DisplayName
	if projectName == "" {
		projectName = project.Key
	}
	if projectName == "" {
		projectName = ticket.ProjectKey
	}
	goal := "standalone"
	if ticket.GoalID != "" {
		goal = ticket.GoalID
		if ticket.GoalTitle != "" {
			goal = fmt.Sprintf("%s - %s", ticket.GoalID, ticket.GoalTitle)
		}
	}

	var b strings.Builder
	b.WriteString(headerStyle.Render("Ticket Detail"))
	b.WriteString("\n")
	b.WriteString(titleStyle.Render(wrapLine(ticket.Title, contentWidth)))
	b.WriteString("\n")
	b.WriteString(wrapLine(ticketDetailMetadata(ticket), contentWidth))
	b.WriteString("\n")
	b.WriteString(wrapLine(fmt.Sprintf("Project: %s  Goal: %s", emptyFallback(projectName, "project"), goal), contentWidth))
	b.WriteString("\n")
	if ticket.Created != "" || len(ticket.Tags) > 0 {
		b.WriteString(wrapLine(fmt.Sprintf("Created: %s  Tags: %s", emptyFallback(compactTime(ticket.Created), "unknown time"), tagsLabel(ticket.Tags)), contentWidth))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(headerStyle.Render("Description"))
	b.WriteString("\n")
	b.WriteString(wrapText(emptyFallback(ticket.Description, "No description recorded."), contentWidth, 4))
	b.WriteString("\n\n")
	b.WriteString(headerStyle.Render("Acceptance"))
	b.WriteString("\n")
	b.WriteString(renderAcceptance(ticket.Acceptance, contentWidth, 4))
	b.WriteString("\n\n")
	b.WriteString(headerStyle.Render("Recent Activity"))
	b.WriteString("\n")
	b.WriteString(renderTicketActivity(project.EventRows, ticket.ID, contentWidth, 4))
	b.WriteString("\n\n")
	b.WriteString(headerStyle.Render("Actions"))
	b.WriteString("\n")
	b.WriteString(wrapText("r refresh | backspace back | up/down move | enter inspect", contentWidth, 2))
	return strings.TrimRight(b.String(), "\n")
}

type ticketBoardColumn struct {
	Title   string
	Style   lipgloss.Style
	Tickets []TicketRow
}

func ticketBoardColumns(tickets []TicketRow) []ticketBoardColumn {
	columns := []ticketBoardColumn{
		{Title: "Open", Style: openTicketStyle},
		{Title: "Waiting", Style: waitingTicketStyle},
		{Title: "Done", Style: doneTicketStyle},
	}
	for _, ticket := range tickets {
		switch ticketBoardStatus(ticket.Status) {
		case "waiting":
			columns[1].Tickets = append(columns[1].Tickets, ticket)
		case "done":
			columns[2].Tickets = append(columns[2].Tickets, ticket)
		default:
			columns[0].Tickets = append(columns[0].Tickets, ticket)
		}
	}
	return columns
}

func renderColumnTicketBoard(columns []ticketBoardColumn, width int, selectedID string) string {
	gap := "  "
	columnWidth := (width - len(gap)*(len(columns)-1)) / len(columns)
	if columnWidth > 44 {
		columnWidth = 44
	}
	if columnWidth < 20 {
		columnWidth = 20
	}
	renderedColumns := make([][]string, 0, len(columns))
	maxLines := 0
	for _, column := range columns {
		lines := renderTicketColumnLines(column, columnWidth, 6, selectedID)
		renderedColumns = append(renderedColumns, lines)
		if len(lines) > maxLines {
			maxLines = len(lines)
		}
	}

	var b strings.Builder
	for row := 0; row < maxLines; row++ {
		for columnIndex, lines := range renderedColumns {
			cell := ""
			if row < len(lines) {
				cell = lines[row]
			}
			b.WriteString(fmt.Sprintf("%-*s", columnWidth, truncate(cell, columnWidth)))
			if columnIndex < len(renderedColumns)-1 {
				b.WriteString(gap)
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}

func renderStackedTicketBoard(columns []ticketBoardColumn, width int, selectedID string) string {
	var b strings.Builder
	for index, column := range columns {
		if index > 0 {
			b.WriteString("\n")
		}
		lines := renderTicketColumnLines(column, width, 5, selectedID)
		for _, line := range lines {
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	return b.String()
}

func renderTicketColumnLines(column ticketBoardColumn, width int, limit int, selectedID string) []string {
	lines := []string{column.Style.Render(fmt.Sprintf("%s (%d)", column.Title, len(column.Tickets)))}
	if len(column.Tickets) == 0 {
		lines = append(lines, mutedStyle.Render("No tickets"))
		return lines
	}
	start := visibleTicketStart(column.Tickets, limit, selectedID)
	if start > 0 {
		lines = append(lines, mutedStyle.Render(fmt.Sprintf("+ %d earlier", start)))
	}
	end := start + limit
	if end > len(column.Tickets) {
		end = len(column.Tickets)
	}
	for _, ticket := range column.Tickets[start:end] {
		lines = append(lines, renderTicketCardLines(ticket, width, selectedID != "" && ticket.ID == selectedID)...)
	}
	if end < len(column.Tickets) {
		lines = append(lines, mutedStyle.Render(fmt.Sprintf("+ %d more", len(column.Tickets)-end)))
	}
	return lines
}

func visibleTicketStart(tickets []TicketRow, limit int, selectedID string) int {
	if limit <= 0 || selectedID == "" {
		return 0
	}
	for index, ticket := range tickets {
		if ticket.ID == selectedID && index >= limit {
			return index - limit + 1
		}
	}
	return 0
}

func renderTicketCardLines(ticket TicketRow, width int, selected bool) []string {
	label := ticket.ID
	if label == "" {
		label = "ticket"
	}
	goal := "standalone"
	if ticket.GoalID != "" {
		goal = ticket.GoalID
	}
	state := ticketStateLabel(ticket.Status)
	owner := ticketOwnerDisplay(ticket)
	prefix := "-"
	if selected {
		prefix = ">"
	}
	header := fmt.Sprintf("%s %s [%s]", prefix, label, state)
	ownerLine := "  " + owner
	detail := fmt.Sprintf("  %s (%s)", ticket.Title, goal)
	rawStatus := strings.TrimSpace(ticket.Status)
	if rawStatus != "" && !ticketStatusIsCanonical(rawStatus, state) {
		detail = fmt.Sprintf("  %s %s (%s)", rawStatus, ticket.Title, goal)
	}
	if width > 0 {
		header = truncate(header, width)
		ownerLine = truncate(ownerLine, width)
		detail = truncate(detail, width)
	}
	return []string{header, ownerLine, detail}
}

func ticketBoardStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "done", "completed", "complete", "closed", "merged", "shipped":
		return "done"
	case "waiting", "blocked", "paused", "pending", "review", "in-review", "needs-review":
		return "waiting"
	default:
		return "open"
	}
}

type ticketClaimInfo struct {
	Owner   string
	Expires string
}

func currentTicketClaims(events []EventRow) map[string]ticketClaimInfo {
	claims := map[string]ticketClaimInfo{}
	for _, event := range events {
		if event.Type != "ticket.claimed" || event.ObjectID == "" || strings.TrimSpace(event.Actor) == "" {
			continue
		}
		if _, exists := claims[event.ObjectID]; exists {
			continue
		}
		claims[event.ObjectID] = ticketClaimInfo{Owner: event.Actor, Expires: event.Expires}
	}
	return claims
}

func ticketStateDisplay(status string) string {
	state := ticketStateLabel(status)
	raw := strings.TrimSpace(status)
	if raw == "" {
		return state
	}
	if ticketStatusIsCanonical(raw, state) {
		return state
	}
	return fmt.Sprintf("%s:%s", state, raw)
}

func ticketStateLabel(status string) string {
	switch ticketBoardStatus(status) {
	case "waiting":
		return "Waiting"
	case "done":
		return "Done"
	default:
		return "Open"
	}
}

func ticketStatusIsCanonical(status string, state string) bool {
	lowerStatus := strings.ToLower(strings.TrimSpace(status))
	lowerState := strings.ToLower(strings.TrimSpace(state))
	return lowerStatus == lowerState || (lowerState == "open" && lowerStatus == "active")
}

func ticketOwnerDisplay(ticket TicketRow) string {
	if strings.TrimSpace(ticket.Owner) == "" {
		return "unowned"
	}
	return "@" + ticket.Owner
}

func ticketDetailMetadata(ticket TicketRow) string {
	parts := []string{
		emptyFallback(ticket.ID, "ticket"),
		"state " + ticketStateDisplay(ticket.Status),
		"owner " + emptyFallback(ticket.Owner, "unassigned"),
	}
	if strings.TrimSpace(ticket.ClaimUntil) != "" {
		parts = append(parts, "claim until "+compactTime(ticket.ClaimUntil))
	}
	return strings.Join(parts, "  ")
}

func nextStatusFilter(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "":
		return "open"
	case "open":
		return "waiting"
	case "waiting":
		return "done"
	default:
		return ""
	}
}

func nextOwnerFilter(owner string) string {
	switch strings.ToLower(strings.TrimSpace(owner)) {
	case "":
		return "owned"
	case "owned":
		return "unowned"
	default:
		return ""
	}
}

func detailContentWidth(width int) int {
	if width <= 0 {
		return 88
	}
	if width > 96 {
		return 96
	}
	if width < 40 {
		return 40
	}
	return width
}

func tagsLabel(tags []string) string {
	if len(tags) == 0 {
		return "none"
	}
	return strings.Join(tags, ", ")
}

func renderAcceptance(items []string, width int, limit int) string {
	if len(items) == 0 {
		return mutedStyle.Render("No acceptance criteria recorded.")
	}
	var b strings.Builder
	for index, item := range items {
		if index >= limit {
			if index > 0 {
				b.WriteString("\n")
			}
			b.WriteString(mutedStyle.Render(fmt.Sprintf("+ %d more criteria", len(items)-index)))
			break
		}
		if index > 0 {
			b.WriteString("\n")
		}
		b.WriteString(wrapBullet(item, width))
	}
	return b.String()
}

func renderTicketActivity(events []EventRow, ticketID string, width int, limit int) string {
	rendered := 0
	var b strings.Builder
	for _, event := range events {
		if event.ObjectID != ticketID {
			continue
		}
		if rendered >= limit {
			b.WriteString("\n")
			b.WriteString(mutedStyle.Render("+ more activity"))
			break
		}
		if rendered > 0 {
			b.WriteString("\n")
		}
		summary := event.Summary
		if summary == "" {
			summary = event.Type
		}
		line := fmt.Sprintf("- %s %s by %s: %s", compactTime(event.Time), event.Type, emptyFallback(event.Actor, "unknown"), summary)
		b.WriteString(wrapLine(line, width))
		rendered++
	}
	if rendered == 0 {
		return mutedStyle.Render("No recent ticket activity.")
	}
	return b.String()
}

func wrapBullet(value string, width int) string {
	prefix := "- "
	available := width - len(prefix)
	if available < 12 {
		available = 12
	}
	lines := wrapWords(value, available, 3)
	for index, line := range lines {
		if index == 0 {
			lines[index] = prefix + line
			continue
		}
		lines[index] = "  " + line
	}
	return strings.Join(lines, "\n")
}

func wrapText(value string, width int, limit int) string {
	paragraphs := strings.Split(strings.TrimSpace(value), "\n")
	lines := make([]string, 0)
	for _, paragraph := range paragraphs {
		for _, line := range wrapWords(paragraph, width, limit-len(lines)) {
			lines = append(lines, line)
			if len(lines) >= limit {
				break
			}
		}
		if len(lines) >= limit {
			break
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n")
}

func wrapLine(value string, width int) string {
	return strings.Join(wrapWords(value, width, 1), "\n")
}

func wrapWords(value string, width int, limit int) []string {
	if width <= 0 {
		width = 80
	}
	if limit <= 0 {
		return nil
	}
	words := strings.Fields(value)
	if len(words) == 0 {
		return []string{""}
	}
	lines := make([]string, 0, limit)
	current := ""
	for _, word := range words {
		for len(word) > width {
			chunk := word[:width]
			word = word[width:]
			if current != "" {
				lines = append(lines, current)
				current = ""
				if len(lines) >= limit {
					return lines
				}
			}
			lines = append(lines, chunk)
			if len(lines) >= limit {
				return lines
			}
		}
		if current == "" {
			current = word
			continue
		}
		if len(current)+1+len(word) <= width {
			current += " " + word
			continue
		}
		lines = append(lines, current)
		if len(lines) >= limit {
			return lines
		}
		current = word
	}
	if current != "" && len(lines) < limit {
		lines = append(lines, current)
	}
	return lines
}

func RenderTimeline(events []EventRow) string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("Timeline"))
	b.WriteString("\n")
	if len(events) == 0 {
		b.WriteString(mutedStyle.Render("No events recorded. Create, claim, comment, or complete work to build the timeline."))
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

func selectedTicket(tickets []TicketRow, selected int) TicketRow {
	if len(tickets) == 0 {
		return TicketRow{}
	}
	return tickets[clampSelection(selected, len(tickets))]
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

func selectedTicketIndexByID(tickets []TicketRow, id string, fallback int) int {
	if id != "" {
		for index, ticket := range tickets {
			if ticket.ID == id {
				return index
			}
		}
	}
	return clampSelection(fallback, len(tickets))
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

func objectGoalsByID(goals []maat.ObjectGoal) map[string]maat.ObjectGoal {
	byID := map[string]maat.ObjectGoal{}
	for _, goal := range goals {
		if goal.ID == "" {
			continue
		}
		byID[goal.ID] = goal
	}
	return byID
}

func eventRowFromObject(project maat.ObjectProject, event maat.ObjectEvent) EventRow {
	return EventRow{
		ID:          event.ID,
		Time:        event.Time,
		Actor:       event.Actor,
		ProjectKey:  event.ProjectKey,
		ProjectName: project.DisplayName,
		Type:        event.Type,
		ObjectID:    event.ObjectID,
		Expires:     event.Expires,
		Summary:     event.Summary,
	}
}

func sortedEventRows(events []EventRow) []EventRow {
	sort.Slice(events, func(i, j int) bool {
		if events[i].Time == events[j].Time {
			return events[i].ID > events[j].ID
		}
		return events[i].Time > events[j].Time
	})
	return events
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
