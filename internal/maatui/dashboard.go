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

type Catalog struct {
	Apps          []CatalogEntry
	Patterns      []CatalogEntry
	Decisions     []CatalogEntry
	Opportunities []CatalogEntry
}

type CatalogEntry struct {
	ID       string
	Kind     string
	Title    string
	Summary  string
	Category string
	Status   string
	Metadata []string
	Links    []string
	Details  []string
}

type Dashboard struct {
	Projects []ProjectRow
	Summary  maat.StatusSummary
	Events   []EventRow
	Catalog  Catalog
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
	DetailModeCatalog
)

type CatalogMode int

const (
	CatalogModeApps CatalogMode = iota
	CatalogModePatterns
	CatalogModeDecisions
	CatalogModeOpportunities
)

type Model struct {
	dashboard           Dashboard
	err                 error
	refreshErr          error
	refreshing          bool
	nextRefreshID       int
	activeRefreshID     int
	width               int
	selected            int
	selectedTicket      int
	selectedApp         int
	selectedPattern     int
	selectedDecision    int
	selectedOpportunity int
	ticketFocus         bool
	catalogInspect      bool
	catalogFilter       string
	filters             DashboardFilters
	filterEditing       bool
	mode                DetailMode
	catalogMode         CatalogMode
	storage             string
	load                dashboardLoader
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
			if m.mode == DetailModeCatalog {
				m.catalogMode = nextCatalogMode(m.catalogMode)
				m.catalogInspect = false
				m = m.withCatalogSelection()
			} else {
				m.mode = nextDetailMode(m.mode)
				m.ticketFocus = false
			}
		case "left", "h":
			if m.mode == DetailModeCatalog {
				m.catalogMode = previousCatalogMode(m.catalogMode)
				m.catalogInspect = false
				m = m.withCatalogSelection()
			} else {
				m.mode = previousDetailMode(m.mode)
				m.ticketFocus = false
			}
		case "enter":
			filtered := FilterDashboard(m.dashboard, m.filters)
			selected := selectedIndexByKey(filtered.Projects, selectedProject(m.dashboard.Projects, m.selected).Key, m.selected)
			if m.mode == DetailModeCatalog {
				m.catalogInspect = true
				m = m.withCatalogSelection()
			} else if m.mode == DetailModeProject && len(filtered.Projects) > 0 {
				m.mode = DetailModeTickets
				m.ticketFocus = false
				m.selectedTicket = clampSelection(m.selectedTicket, len(selectedProject(filtered.Projects, selected).TicketRows))
			} else if m.mode == DetailModeTickets && len(selectedProject(filtered.Projects, selected).TicketRows) > 0 {
				m.ticketFocus = true
				m.selectedTicket = clampSelection(m.selectedTicket, len(selectedProject(filtered.Projects, selected).TicketRows))
			}
		case "backspace":
			if m.mode == DetailModeCatalog && m.catalogInspect {
				m.catalogInspect = false
			} else if m.mode == DetailModeCatalog {
				m.mode = DetailModeProject
			} else if m.mode == DetailModeTickets && m.ticketFocus {
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
			m.catalogFilter = ""
			m.filterEditing = false
			m = m.withFilterSelection()
		case "s":
			m.filters.Status = nextStatusFilter(m.filters.Status)
			m = m.withFilterSelection()
		case "o":
			m.filters.Owner = nextOwnerFilter(m.filters.Owner)
			m = m.withFilterSelection()
		case "f":
			if m.mode == DetailModeCatalog {
				m.catalogFilter = nextCatalogFilter(m.catalogFilter)
				m = m.withCatalogSelection()
			}
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
	projectFilters := m.filters
	if m.mode == DetailModeCatalog {
		projectFilters.Query = ""
	}
	filtered := FilterDashboard(m.dashboard, projectFilters)
	selected := selectedIndexByKey(filtered.Projects, selectedProject(m.dashboard.Projects, m.selected).Key, m.selected)
	view := RenderDashboardWithFiltersAndCatalogState(
		filtered,
		selected,
		m.mode,
		m.width,
		m.selectedTicket,
		m.ticketFocus,
		m.filters,
		m.filterEditing,
		m.catalogMode,
		CatalogSelections{
			App:         m.selectedApp,
			Pattern:     m.selectedPattern,
			Decision:    m.selectedDecision,
			Opportunity: m.selectedOpportunity,
		},
		m.catalogFilter,
		m.catalogInspect,
	)
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
	m = m.withCatalogSelection()
	return m
}

func (m Model) moveSelection(delta int) Model {
	filtered := FilterDashboard(m.dashboard, m.filters)
	selectedKey := selectedProject(m.dashboard.Projects, m.selected).Key
	selectedIndex := selectedIndexByKey(filtered.Projects, selectedKey, m.selected)
	if m.mode == DetailModeCatalog {
		m = m.moveCatalogSelection(delta)
		return m
	}
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

func (m Model) moveCatalogSelection(delta int) Model {
	catalog := FilterCatalog(m.dashboard.Catalog, m.filters.Query, m.catalogFilter)
	switch m.catalogMode {
	case CatalogModePatterns:
		m.selectedPattern = clampSelection(m.selectedPattern+delta, len(catalog.Patterns))
	case CatalogModeDecisions:
		m.selectedDecision = clampSelection(m.selectedDecision+delta, len(catalog.Decisions))
	case CatalogModeOpportunities:
		m.selectedOpportunity = clampSelection(m.selectedOpportunity+delta, len(catalog.Opportunities))
	default:
		m.selectedApp = clampSelection(m.selectedApp+delta, len(catalog.Apps))
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
	m = m.withCatalogSelection()
	return m
}

func (m Model) withCatalogSelection() Model {
	catalog := FilterCatalog(m.dashboard.Catalog, m.filters.Query, m.catalogFilter)
	m.selectedApp = clampSelection(m.selectedApp, len(catalog.Apps))
	m.selectedPattern = clampSelection(m.selectedPattern, len(catalog.Patterns))
	m.selectedDecision = clampSelection(m.selectedDecision, len(catalog.Decisions))
	m.selectedOpportunity = clampSelection(m.selectedOpportunity, len(catalog.Opportunities))
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
	catalog := Catalog{}
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
		if project.Catalog != nil {
			catalog = mergeCatalogs(catalog, catalogFromObject(*project.Catalog))
		}
	}
	if catalogEmpty(catalog) {
		catalog = DefaultTerminalAppsCatalog()
	}
	return Dashboard{Projects: rows, Summary: summary, Events: sortedEventRows(events), Catalog: catalog}
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
	return Dashboard{Projects: rows, Summary: summarizeProjectRows(rows), Events: filterEventRows(dashboard.Events, rows), Catalog: dashboard.Catalog}
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
	return RenderDashboardWithFiltersAndCatalogState(dashboard, selected, mode, width, selectedTicket, ticketFocus, filters, editingFilter, CatalogModeApps, CatalogSelections{}, "", false)
}

func RenderDashboardWithFiltersAndCatalogState(dashboard Dashboard, selected int, mode DetailMode, width int, selectedTicket int, ticketFocus bool, filters DashboardFilters, editingFilter bool, catalogMode CatalogMode, catalogSelections CatalogSelections, catalogFilter string, catalogInspect bool) string {
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
	case DetailModeCatalog:
		b.WriteString(RenderCatalog(dashboard.Catalog, catalogMode, catalogSelections, width, filters.Query, catalogFilter, catalogInspect))
	default:
		b.WriteString(RenderProjectDetail(project))
	}
	b.WriteString("\n\n")
	b.WriteString(mutedStyle.Render("Use up/down or k/j to select, enter to open project board/detail, backspace back, tab/right for timeline/catalog, / query, f catalog filter, s state, o owner, p project, c clear, r reload, q quit."))
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

type CatalogSelections struct {
	App         int
	Pattern     int
	Decision    int
	Opportunity int
}

func DefaultTerminalAppsCatalog() Catalog {
	return Catalog{
		Apps: []CatalogEntry{
			{
				ID:       "app:lazygit",
				Kind:     "app",
				Title:    "lazygit",
				Summary:  "Simple terminal UI for git commands.",
				Category: "git",
				Status:   "review",
				Metadata: []string{"Go", "MIT", "stars unknown locally"},
				Links:    []string{"github.com/jesseduffield/lazygit"},
				Details: []string{
					"Study dashboard density, keyboard-first navigation, and focused object inspection.",
					"Maat relevance: project boards should keep selected work and detail context visible.",
				},
			},
			{
				ID:       "app:btop",
				Kind:     "app",
				Title:    "btop",
				Summary:  "Terminal resource monitor with strong visual hierarchy.",
				Category: "monitoring",
				Status:   "review",
				Metadata: []string{"C++", "Apache-2.0", "stars unknown locally"},
				Links:    []string{"github.com/aristocratos/btop"},
				Details: []string{
					"Study compact summaries, stable regions, and readable state without opening files.",
					"Maat relevance: status and ownership must remain legible at a glance.",
				},
			},
			{
				ID:       "app:gh-dash",
				Kind:     "app",
				Title:    "gh-dash",
				Summary:  "GitHub dashboard for pull requests and issues.",
				Category: "dashboard",
				Status:   "review",
				Metadata: []string{"Go", "license unknown", "stars unknown locally"},
				Links:    []string{"github.com/dlvhdr/gh-dash"},
				Details: []string{
					"Study grouped work queues, filters, and detail panes for work review.",
					"Maat relevance: ticket lists should bridge scanning and verification.",
				},
			},
			{
				ID:       "app:superfile",
				Kind:     "app",
				Title:    "superfile",
				Summary:  "Modern terminal file manager with pane-based navigation.",
				Category: "files",
				Status:   "review",
				Metadata: []string{"Go", "MIT", "stars unknown locally"},
				Links:    []string{"github.com/yorukot/superfile"},
				Details: []string{
					"Study pane switching, selection markers, and compact detail affordances.",
					"Maat relevance: catalog and project panes need predictable keyboard movement.",
				},
			},
		},
		Patterns: []CatalogEntry{
			{
				ID:       "pattern:focused-detail-pane",
				Kind:     "pattern",
				Title:    "Focused detail pane",
				Summary:  "List views need adjacent detail to keep object context visible.",
				Category: "inspection",
				Status:   "adopt",
				Metadata: []string{"observed in lazygit, gh-dash, superfile"},
				Details: []string{
					"Problem: list rows hide description, metadata, and recent activity.",
					"Maat use: selected ticket detail pane with description, acceptance, status, owner, goal, and events.",
					"Related ticket: T-20260527-104752-bb33.",
				},
			},
			{
				ID:       "pattern:keyboard-model",
				Kind:     "pattern",
				Title:    "Keyboard model",
				Summary:  "Fast terminal apps make movement and inspection predictable.",
				Category: "navigation",
				Status:   "adopt",
				Metadata: []string{"up/down, k/j, tab, enter, backspace"},
				Details: []string{
					"Problem: command-heavy navigation slows repeated scanning.",
					"Maat use: project list -> board -> ticket detail, plus catalog pane navigation.",
					"Related ticket: T-20260527-104802-f29d.",
				},
			},
			{
				ID:       "pattern:background-refresh",
				Kind:     "pattern",
				Title:    "Background refresh",
				Summary:  "Long-running terminal sessions should show fresh state without losing context.",
				Category: "refresh",
				Status:   "adopt",
				Metadata: []string{"manual r reload", "stable selected row"},
				Details: []string{
					"Problem: stale dashboards make agent handoffs harder to trust.",
					"Maat use: refresh indicators, recoverable warnings, and selection preservation.",
					"Related ticket: T-20260526-220322-65be.",
				},
			},
			{
				ID:       "pattern:empty-states",
				Kind:     "pattern",
				Title:    "Empty states",
				Summary:  "Empty views should explain the next useful action.",
				Category: "empty-state",
				Status:   "adopt",
				Metadata: []string{"no-color readable", "actionable copy"},
				Details: []string{
					"Problem: blank panes look broken and do not guide the user.",
					"Maat use: filtered empty states, missing tickets, missing catalog rows, and storage errors all name next actions.",
					"Related ticket: T-20260527-104802-f29d.",
				},
			},
		},
		Decisions: []CatalogEntry{
			{
				ID:       "decision:adopt-focused-detail-pane",
				Kind:     "decision",
				Title:    "Adopt focused detail panes",
				Summary:  "Maat should keep selected ticket and catalog detail beside the list.",
				Category: "inspection",
				Status:   "adopt",
				Metadata: []string{"pattern:focused-detail-pane", "2026-05-27"},
				Details:  []string{"Rationale: the user wants to click into projects, scan boards, and read each item without opening Markdown files."},
			},
			{
				ID:       "decision:adopt-keyboard-first-flow",
				Kind:     "decision",
				Title:    "Adopt keyboard-first flow",
				Summary:  "Keep project and catalog movement available through predictable keys.",
				Category: "navigation",
				Status:   "adopt",
				Metadata: []string{"pattern:keyboard-model", "2026-05-27"},
				Details:  []string{"Rationale: terminal work should be fast enough for daily review and agent verification."},
			},
		},
		Opportunities: []CatalogEntry{
			{
				ID:       "opportunity:catalog-mode",
				Kind:     "opportunity",
				Title:    "Catalog mode",
				Summary:  "Expose terminal app examples and reusable patterns inside the TUI.",
				Category: "catalog",
				Status:   "in progress",
				Metadata: []string{"T-20260527-104741-731b", "medium effort"},
				Details:  []string{"Bridge IA observations into project work without making the user leave the dashboard."},
			},
			{
				ID:       "opportunity:board-detail-flow",
				Kind:     "opportunity",
				Title:    "Board detail flow",
				Summary:  "Make project list, board, and ticket detail the primary human path.",
				Category: "project-board",
				Status:   "in progress",
				Metadata: []string{"T-20260527-104752-bb33", "low effort"},
				Details:  []string{"Keep the first screen as projects, then use enter and backspace as terminal equivalents of click in and back out."},
			},
			{
				ID:       "opportunity:no-color-verification",
				Kind:     "opportunity",
				Title:    "No-color verification",
				Summary:  "Prove selected rows, statuses, and empty states are readable without color.",
				Category: "accessibility",
				Status:   "in progress",
				Metadata: []string{"T-20260527-104802-f29d", "low effort"},
				Details:  []string{"Selection markers, status labels, and empty-state copy must carry meaning without relying on color."},
			},
		},
	}
}

func catalogFromObject(catalog maat.Catalog) Catalog {
	converted := Catalog{
		Apps:          make([]CatalogEntry, 0, len(catalog.Apps)),
		Patterns:      make([]CatalogEntry, 0, len(catalog.Patterns)),
		Decisions:     make([]CatalogEntry, 0, len(catalog.Decisions)),
		Opportunities: make([]CatalogEntry, 0, len(catalog.Opportunities)),
	}
	for _, app := range catalog.Apps {
		converted.Apps = append(converted.Apps, CatalogEntry{
			ID:       firstCatalogValue(app.ID, app.Slug),
			Kind:     "app",
			Title:    firstCatalogValue(app.Name, app.Slug, app.ID),
			Summary:  app.Summary,
			Category: app.Category,
			Status:   "reviewed",
			Metadata: compactCatalogMetadata(app.Stars, app.Language, app.License, "reviewed "+app.LastReviewed),
			Links:    compactCatalogMetadata(app.SourceURL, app.WebsiteURL),
			Details: compactCatalogMetadata(
				catalogDetail("Notes", app.Notes),
				catalogDetail("Patterns", strings.Join(app.Patterns, ", ")),
				catalogDetail("Related goals", strings.Join(app.RelatedGoals, ", ")),
				catalogDetail("Related tickets", strings.Join(app.RelatedTickets, ", ")),
			),
		})
	}
	for _, pattern := range catalog.Patterns {
		converted.Patterns = append(converted.Patterns, CatalogEntry{
			ID:       firstCatalogValue(pattern.ID, pattern.Slug),
			Kind:     "pattern",
			Title:    firstCatalogValue(pattern.Title, pattern.Slug, pattern.ID),
			Summary:  pattern.Problem,
			Category: pattern.Category,
			Status:   "pattern",
			Metadata: compactCatalogMetadata("observed in " + strings.Join(pattern.ObservedIn, ", ")),
			Details: compactCatalogMetadata(
				catalogDetail("Maat use", pattern.MaatUse),
				catalogDetail("Implementation", pattern.ImplementationNotes),
				catalogDetail("Related goals", strings.Join(pattern.RelatedGoals, ", ")),
				catalogDetail("Related tickets", strings.Join(pattern.RelatedTickets, ", ")),
			),
		})
	}
	for _, decision := range catalog.Decisions {
		converted.Decisions = append(converted.Decisions, CatalogEntry{
			ID:       firstCatalogValue(decision.ID, decision.Slug),
			Kind:     "decision",
			Title:    firstCatalogValue(decision.Title, decision.Slug, decision.ID),
			Summary:  decision.Rationale,
			Category: decision.Pattern,
			Status:   firstCatalogValue(decision.State, decision.Decision),
			Metadata: compactCatalogMetadata(decision.Date, "source "+decision.SourceApp),
			Details: compactCatalogMetadata(
				catalogDetail("Evidence", strings.Join(decision.Evidence, ", ")),
				catalogDetail("Related goals", strings.Join(decision.RelatedGoals, ", ")),
				catalogDetail("Related tickets", strings.Join(decision.RelatedTickets, ", ")),
			),
		})
	}
	for _, opportunity := range catalog.Opportunities {
		converted.Opportunities = append(converted.Opportunities, CatalogEntry{
			ID:       firstCatalogValue(opportunity.ID, opportunity.Slug),
			Kind:     "opportunity",
			Title:    firstCatalogValue(opportunity.Title, opportunity.Slug, opportunity.ID),
			Summary:  opportunity.Summary,
			Category: opportunity.Area,
			Status:   opportunity.Status,
			Metadata: compactCatalogMetadata(opportunity.Effort, "risk "+opportunity.Risk, "pattern "+opportunity.SourcePattern),
			Details: compactCatalogMetadata(
				catalogDetail("Suggested goal", opportunity.SuggestedGoal),
				catalogDetail("Suggested ticket", opportunity.SuggestedTicket),
			),
		})
	}
	return converted
}

func mergeCatalogs(left Catalog, right Catalog) Catalog {
	left.Apps = append(left.Apps, right.Apps...)
	left.Patterns = append(left.Patterns, right.Patterns...)
	left.Decisions = append(left.Decisions, right.Decisions...)
	left.Opportunities = append(left.Opportunities, right.Opportunities...)
	return left
}

func catalogEmpty(catalog Catalog) bool {
	return len(catalog.Apps) == 0 &&
		len(catalog.Patterns) == 0 &&
		len(catalog.Decisions) == 0 &&
		len(catalog.Opportunities) == 0
}

func firstCatalogValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func compactCatalogMetadata(values ...string) []string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || strings.EqualFold(value, "unknown") || strings.EqualFold(value, "reviewed unknown") {
			continue
		}
		items = append(items, value)
	}
	return items
}

func catalogDetail(label, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return label + ": " + value
}

func RenderCatalog(catalog Catalog, mode CatalogMode, selections CatalogSelections, width int, query string, filter string, inspected bool) string {
	catalog = FilterCatalog(catalog, query, filter)
	if width <= 0 {
		width = 120
	}
	if width < 72 {
		return renderStackedCatalog(catalog, mode, selections, query, filter, inspected, width)
	}
	return renderPaneCatalog(catalog, mode, selections, query, filter, inspected, width)
}

func renderPaneCatalog(catalog Catalog, mode CatalogMode, selections CatalogSelections, query string, filter string, inspected bool, width int) string {
	gap := "  "
	listWidth := (width - len(gap)*2) / 4
	if listWidth < 24 {
		listWidth = 24
	}
	if listWidth > 34 {
		listWidth = 34
	}
	detailWidth := width - listWidth*2 - len(gap)*2
	if detailWidth < 36 {
		detailWidth = 36
	}

	primaryTitle, primaryRows, primarySelected := catalogPrimaryList(catalog, mode, selections)
	patternSelected := selections.Pattern
	if mode == CatalogModePatterns {
		patternSelected = selections.Pattern
	}
	detail := selectedCatalogEntry(catalog, mode, selections)
	linesA := renderCatalogListLines(primaryTitle, primaryRows, primarySelected, listWidth, 8)
	linesB := renderCatalogListLines("Patterns", catalog.Patterns, patternSelected, listWidth, 8)
	linesC := renderCatalogDetailLines(detail, mode, query, filter, inspected, detailWidth)
	maxLines := maxInt(len(linesA), maxInt(len(linesB), len(linesC)))

	var b strings.Builder
	b.WriteString(headerStyle.Render("Terminal Apps Catalog"))
	b.WriteString("\n")
	b.WriteString(truncate(catalogSummaryLine(catalog, mode, query, filter), width))
	b.WriteString("\n\n")
	for row := 0; row < maxLines; row++ {
		b.WriteString(fmt.Sprintf("%-*s", listWidth, truncate(lineAt(linesA, row), listWidth)))
		b.WriteString(gap)
		b.WriteString(fmt.Sprintf("%-*s", listWidth, truncate(lineAt(linesB, row), listWidth)))
		b.WriteString(gap)
		b.WriteString(truncate(lineAt(linesC, row), detailWidth))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render(truncate("Catalog keys: tab/right mode, left previous mode, / search, f filter, enter inspect, backspace back.", width)))
	return strings.TrimRight(b.String(), "\n")
}

func renderStackedCatalog(catalog Catalog, mode CatalogMode, selections CatalogSelections, query string, filter string, inspected bool, width int) string {
	primaryTitle, primaryRows, primarySelected := catalogPrimaryList(catalog, mode, selections)
	var b strings.Builder
	b.WriteString(headerStyle.Render("Terminal Apps Catalog"))
	b.WriteString("\n")
	b.WriteString(truncate(catalogSummaryLine(catalog, mode, query, filter), width))
	b.WriteString("\n\n")
	for _, line := range renderCatalogListLines(primaryTitle, primaryRows, primarySelected, width, 6) {
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	for _, line := range renderCatalogListLines("Patterns", catalog.Patterns, selections.Pattern, width, 5) {
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	for _, line := range renderCatalogDetailLines(selectedCatalogEntry(catalog, mode, selections), mode, query, filter, inspected, width) {
		b.WriteString(line)
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render(truncate("Catalog keys: tab/right mode, left previous mode, / search, f filter, enter inspect, backspace back.", width)))
	return strings.TrimRight(b.String(), "\n")
}

func catalogSummaryLine(catalog Catalog, mode CatalogMode, query string, filter string) string {
	parts := []string{
		fmt.Sprintf("%d apps", len(catalog.Apps)),
		fmt.Sprintf("%d patterns", len(catalog.Patterns)),
		fmt.Sprintf("%d decisions", len(catalog.Decisions)),
		fmt.Sprintf("%d opportunities", len(catalog.Opportunities)),
		"mode " + catalogModeLabel(mode),
	}
	if strings.TrimSpace(filter) != "" {
		parts = append(parts, "filter "+filter)
	}
	if strings.TrimSpace(query) != "" {
		parts = append(parts, fmt.Sprintf("query %q", strings.TrimSpace(query)))
	}
	return strings.Join(parts, " | ")
}

func catalogPrimaryList(catalog Catalog, mode CatalogMode, selections CatalogSelections) (string, []CatalogEntry, int) {
	switch mode {
	case CatalogModePatterns:
		return "Patterns", catalog.Patterns, selections.Pattern
	case CatalogModeDecisions:
		return "Decisions", catalog.Decisions, selections.Decision
	case CatalogModeOpportunities:
		return "Opportunities", catalog.Opportunities, selections.Opportunity
	default:
		return "Apps", catalog.Apps, selections.App
	}
}

func renderCatalogListLines(title string, rows []CatalogEntry, selected int, width int, limit int) []string {
	lines := []string{headerStyle.Render(fmt.Sprintf("%s (%d)", title, len(rows)))}
	if len(rows) == 0 {
		lines = append(lines, mutedStyle.Render("No catalog items"))
		return lines
	}
	selected = clampSelection(selected, len(rows))
	start := visibleCatalogStart(len(rows), limit, selected)
	if start > 0 {
		lines = append(lines, mutedStyle.Render(fmt.Sprintf("+ %d earlier", start)))
	}
	end := start + limit
	if end > len(rows) {
		end = len(rows)
	}
	for index := start; index < end; index++ {
		row := rows[index]
		prefix := "-"
		if index == selected {
			prefix = ">"
		}
		label := row.Title
		if label == "" {
			label = row.ID
		}
		if row.Status != "" {
			label = fmt.Sprintf("%s [%s]", label, row.Status)
		}
		lines = append(lines, truncate(fmt.Sprintf("%s %s", prefix, label), width))
		if row.Summary != "" {
			lines = append(lines, truncate("  "+row.Summary, width))
		}
	}
	if end < len(rows) {
		lines = append(lines, mutedStyle.Render(fmt.Sprintf("+ %d more", len(rows)-end)))
	}
	return lines
}

func renderCatalogDetailLines(entry CatalogEntry, mode CatalogMode, query string, filter string, inspected bool, width int) []string {
	lines := []string{headerStyle.Render("Detail")}
	if entry.ID == "" && entry.Title == "" {
		lines = append(lines, "No "+catalogModeLabel(mode)+" match the current catalog filters.")
		lines = append(lines, "Use / to search, f to change the category filter, or tab to switch modes.")
		return lines
	}
	title := entry.Title
	if title == "" {
		title = entry.ID
	}
	lines = append(lines, titleStyle.Render(truncate(title, width)))
	metadata := []string{entry.Kind}
	if entry.Category != "" {
		metadata = append(metadata, entry.Category)
	}
	if entry.Status != "" {
		metadata = append(metadata, "status "+entry.Status)
	}
	metadata = append(metadata, entry.Metadata...)
	lines = append(lines, wrapLine(strings.Join(metadata, " | "), width))
	if entry.Summary != "" {
		lines = append(lines, "")
		lines = appendRenderedLines(lines, wrapText(entry.Summary, width, 2))
	}
	if inspected {
		if len(entry.Details) > 0 {
			lines = append(lines, "")
			lines = append(lines, headerStyle.Render("Notes"))
			for _, detail := range entry.Details {
				lines = appendRenderedLines(lines, wrapBullet(detail, width))
			}
		}
		if len(entry.Links) > 0 {
			lines = append(lines, "")
			lines = append(lines, headerStyle.Render("Links"))
			for _, link := range entry.Links {
				lines = appendRenderedLines(lines, wrapBullet(link, width))
			}
		}
	} else {
		lines = append(lines, "")
		lines = append(lines, mutedStyle.Render("Press enter to inspect notes and links."))
	}
	return lines
}

func appendRenderedLines(lines []string, rendered string) []string {
	for _, line := range strings.Split(rendered, "\n") {
		lines = append(lines, line)
	}
	return lines
}

func selectedCatalogEntry(catalog Catalog, mode CatalogMode, selections CatalogSelections) CatalogEntry {
	switch mode {
	case CatalogModePatterns:
		return selectedCatalogRow(catalog.Patterns, selections.Pattern)
	case CatalogModeDecisions:
		return selectedCatalogRow(catalog.Decisions, selections.Decision)
	case CatalogModeOpportunities:
		return selectedCatalogRow(catalog.Opportunities, selections.Opportunity)
	default:
		return selectedCatalogRow(catalog.Apps, selections.App)
	}
}

func selectedCatalogRow(rows []CatalogEntry, selected int) CatalogEntry {
	if len(rows) == 0 {
		return CatalogEntry{}
	}
	return rows[clampSelection(selected, len(rows))]
}

func visibleCatalogStart(count int, limit int, selected int) int {
	if count <= 0 || limit <= 0 || selected < limit {
		return 0
	}
	return selected - limit + 1
}

func FilterCatalog(catalog Catalog, query string, filter string) Catalog {
	query = strings.ToLower(strings.TrimSpace(query))
	filter = strings.ToLower(strings.TrimSpace(filter))
	return Catalog{
		Apps:          filterCatalogEntries(catalog.Apps, query, filter),
		Patterns:      filterCatalogEntries(catalog.Patterns, query, filter),
		Decisions:     filterCatalogEntries(catalog.Decisions, query, filter),
		Opportunities: filterCatalogEntries(catalog.Opportunities, query, filter),
	}
}

func filterCatalogEntries(rows []CatalogEntry, query string, filter string) []CatalogEntry {
	filtered := make([]CatalogEntry, 0, len(rows))
	for _, row := range rows {
		haystack := strings.ToLower(strings.Join([]string{
			row.ID,
			row.Kind,
			row.Title,
			row.Summary,
			row.Category,
			row.Status,
			strings.Join(row.Metadata, " "),
			strings.Join(row.Links, " "),
			strings.Join(row.Details, " "),
		}, " "))
		if query != "" && !strings.Contains(haystack, query) {
			continue
		}
		if filter != "" && !strings.Contains(haystack, filter) {
			continue
		}
		filtered = append(filtered, row)
	}
	return filtered
}

func catalogModeLabel(mode CatalogMode) string {
	switch mode {
	case CatalogModePatterns:
		return "patterns"
	case CatalogModeDecisions:
		return "decisions"
	case CatalogModeOpportunities:
		return "opportunities"
	default:
		return "apps"
	}
}

func nextCatalogMode(mode CatalogMode) CatalogMode {
	if mode == CatalogModeOpportunities {
		return CatalogModeApps
	}
	return mode + 1
}

func previousCatalogMode(mode CatalogMode) CatalogMode {
	if mode == CatalogModeApps {
		return CatalogModeOpportunities
	}
	return mode - 1
}

func nextCatalogFilter(filter string) string {
	switch strings.ToLower(strings.TrimSpace(filter)) {
	case "":
		return "navigation"
	case "navigation":
		return "inspection"
	case "inspection":
		return "refresh"
	case "refresh":
		return "empty-state"
	default:
		return ""
	}
}

func lineAt(lines []string, index int) string {
	if index < 0 || index >= len(lines) {
		return ""
	}
	return lines[index]
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
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
	if mode == DetailModeCatalog {
		return DetailModeProject
	}
	return mode + 1
}

func previousDetailMode(mode DetailMode) DetailMode {
	if mode == DetailModeProject {
		return DetailModeCatalog
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
