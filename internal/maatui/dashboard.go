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
	Goals       int
	Tickets     int
	Updated     string
}

type Dashboard struct {
	Projects []ProjectRow
	Summary  maat.StatusSummary
}

type Model struct {
	dashboard Dashboard
	err       error
	width     int
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
	return RenderDashboard(m.dashboard)
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
		row := ProjectRow{
			Key:         project.Key,
			DisplayName: project.DisplayName,
			Status:      project.Status,
			Goals:       len(project.Goals),
			Tickets:     len(project.Tickets),
			Updated:     project.Updated,
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
				} else {
					summary.OpenTickets++
				}
			}
		}
		rows = append(rows, ProjectRow{
			Key:         project.ID,
			DisplayName: project.Title,
			Status:      project.Status,
			Goals:       len(project.Goals),
			Tickets:     ticketCount,
			Updated:     project.Updated,
		})
	}
	return Dashboard{Projects: rows, Summary: summary}
}

func RenderDashboard(dashboard Dashboard) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Maat"))
	b.WriteString("\n")
	b.WriteString(mutedStyle.Render("Git-backed project memory"))
	b.WriteString("\n\n")
	b.WriteString(RenderSummary(dashboard.Summary))
	b.WriteString("\n\n")
	b.WriteString(RenderProjectTable(dashboard.Projects))
	b.WriteString("\n\n")
	b.WriteString(mutedStyle.Render("Press q to quit."))
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
	if len(projects) == 0 {
		return mutedStyle.Render("No projects found.")
	}

	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("%-16s %-10s %7s %8s %s", "Project", "Status", "Goals", "Tickets", "Updated")))
	b.WriteString("\n")
	for _, project := range projects {
		name := project.DisplayName
		if name == "" {
			name = project.Key
		}
		b.WriteString(fmt.Sprintf("%-16s %-10s %7d %8d %s\n", truncate(name, 16), project.Status, project.Goals, project.Tickets, project.Updated))
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
