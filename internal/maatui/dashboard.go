package maatui

import (
	"fmt"
	"strings"

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

type Dashboard struct {
	Projects []ProjectRow
	Summary  maat.StatusSummary
}

type DetailMode int

const (
	DetailModeProject DetailMode = iota
	DetailModeTickets
)

type Model struct {
	dashboard Dashboard
	err       error
	width     int
	selected  int
	mode      DetailMode
}

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
	dashboard, err := LoadDashboard(storage)
	model := NewModel(dashboard, err)
	_, runErr := tea.NewProgram(model, tea.WithAltScreen()).Run()
	if runErr != nil {
		return runErr
	}
	return nil
}

func NewModel(dashboard Dashboard, err error) Model {
	return Model{dashboard: dashboard, err: err}
}

func (m Model) Init() tea.Cmd {
	return nil
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
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
	}
	return m, nil
}

func (m Model) View() string {
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Maat TUI failed: %v", m.err)) + "\n\n" + mutedStyle.Render("Press q to quit.") + "\n"
	}
	return RenderDashboardWithSelectionAndMode(m.dashboard, m.selected, m.mode)
}

func LoadDashboard(storage string) (Dashboard, error) {
	objectProjects, err := maat.LoadObjectProjects(storage)
	if err != nil {
		return Dashboard{}, err
	}
	if len(objectProjects) > 0 {
		return DashboardFromObjectProjects(objectProjects), nil
	}

	legacyProjects, err := maat.LoadProjects(storage)
	if err != nil {
		return Dashboard{}, err
	}
	return DashboardFromLegacyProjects(legacyProjects), nil
}

func DashboardFromObjectProjects(projects []maat.ObjectProject) Dashboard {
	rows := make([]ProjectRow, 0, len(projects))
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
	}
	return Dashboard{Projects: rows, Summary: summary}
}

func DashboardFromLegacyProjects(projects []maat.Project) Dashboard {
	rows := make([]ProjectRow, 0, len(projects))
	var summary maat.StatusSummary
	summary.Projects = len(projects)
	for _, project := range projects {
		ticketCount := 0
		openTickets := 0
		doneTickets := 0
		goalRows := make([]GoalRow, 0, len(project.Goals))
		ticketRows := make([]TicketRow, 0)
		for _, goal := range project.Goals {
			ticketCount += len(goal.Tickets)
			summary.Goals++
			if goal.Status == "active" {
				summary.ActiveGoals++
			}
			if goal.Status == "done" {
				summary.DoneGoals++
			}
			for _, ticket := range goal.Tickets {
				summary.Tickets++
				if ticket.Done {
					summary.DoneTickets++
					doneTickets++
				} else {
					summary.OpenTickets++
					openTickets++
				}
				ticketRows = append(ticketRows, TicketRow{
					ID:     ticket.ID,
					Title:  ticket.Title,
					Status: legacyTicketStatus(ticket.Done),
					GoalID: goal.ID,
				})
			}
			goalRows = append(goalRows, GoalRow{
				ID:      goal.ID,
				Title:   goal.Title,
				Status:  goal.Status,
				Tickets: len(goal.Tickets),
			})
		}
		rows = append(rows, ProjectRow{
			Key:         project.ID,
			DisplayName: project.Title,
			Status:      project.Status,
			Summary:     project.Current,
			Goals:       len(project.Goals),
			Tickets:     ticketCount,
			OpenTickets: openTickets,
			DoneTickets: doneTickets,
			Updated:     project.Updated,
			GoalRows:    goalRows,
			TicketRows:  ticketRows,
		})
	}
	return Dashboard{Projects: rows, Summary: summary}
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
	if mode == DetailModeTickets {
		b.WriteString(RenderProjectTickets(project))
	} else {
		b.WriteString(RenderProjectDetail(project))
	}
	b.WriteString("\n\n")
	b.WriteString(mutedStyle.Render("Search and timeline views are planned. Use up/down or k/j to select, tab to switch detail/tickets, q to quit."))
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

func nextDetailMode(mode DetailMode) DetailMode {
	if mode == DetailModeTickets {
		return DetailModeProject
	}
	return DetailModeTickets
}

func previousDetailMode(mode DetailMode) DetailMode {
	if mode == DetailModeProject {
		return DetailModeTickets
	}
	return DetailModeProject
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

func legacyTicketStatus(done bool) string {
	if done {
		return "done"
	}
	return "active"
}

func emptyFallback(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
