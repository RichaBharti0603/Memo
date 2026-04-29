package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	
	"redis_golang/internal/metrics"
	"redis_golang/internal/replication"
	"redis_golang/internal/storage/memory"
	"redis_golang/pkg/logger"
)

// Styles
var (
	baseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FFCC")).
			Bold(true).
			Padding(0, 1).
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#00FFCC"))

	statBoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2).
			Align(lipgloss.Center)

	statLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Bold(true)
	statValueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF00FF")).Bold(true)

	logBoxStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 2).
			Width(80).
			Height(10)
)

type tickMsg struct {
	conns    int64
	keys     int64
	commands int64
	hits     int64
	misses   int64
	role     string
	replicas int64
}

type model struct {
	quitting bool
	conns    int64
	keys     int64
	commands int64
	hits     int64
	misses   int64
	role     string
	replicas int64
}

func initialModel() model {
	return model{}
}

func (m model) Init() tea.Cmd {
	return tickCmd()
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*500, func(t time.Time) tea.Msg {
		return tickMsg{
			conns:    metrics.GetActiveConnections(),
			keys:     int64(len(memory.GetAllKeys())),
			commands: metrics.GetTotalCommands(),
			hits:     metrics.GetCacheHits(),
			misses:   metrics.GetCacheMisses(),
			role:     string(replication.GlobalRole),
			replicas: metrics.GetConnectedReplicas(),
		}
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	case tickMsg:
		m.conns = msg.conns
		m.keys = msg.keys
		m.commands = msg.commands
		m.hits = msg.hits
		m.misses = msg.misses
		m.role = msg.role
		m.replicas = msg.replicas
		return m, tickCmd()
	}
	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return "Shutting down TUI...\n"
	}

	hitRate := 0.0
	if m.hits+m.misses > 0 {
		hitRate = float64(m.hits) / float64(m.hits+m.misses) * 100
	}

	// Render Title
	title := titleStyle.Render("REDIS-GOLANG DASHBOARD")
	roleStr := "Role: " + strings.ToUpper(m.role)
	if m.role == "primary" {
		roleStr += fmt.Sprintf(" | Replicas: %d", m.replicas)
	}

	header := lipgloss.JoinVertical(lipgloss.Center, title, lipgloss.NewStyle().Foreground(lipgloss.Color("#00ffcc")).Render(roleStr))

	// Render Stats
	statConns := statBoxStyle.Render(fmt.Sprintf("%s\n%s", statLabelStyle.Render("CONNS"), statValueStyle.Render(fmt.Sprintf("%d", m.conns))))
	statKeys := statBoxStyle.Render(fmt.Sprintf("%s\n%s", statLabelStyle.Render("KEYS"), statValueStyle.Render(fmt.Sprintf("%d", m.keys))))
	statCmds := statBoxStyle.Render(fmt.Sprintf("%s\n%s", statLabelStyle.Render("CMDS"), statValueStyle.Render(fmt.Sprintf("%d", m.commands))))
	statHits := statBoxStyle.Render(fmt.Sprintf("%s\n%s", statLabelStyle.Render("HIT RATE"), statValueStyle.Render(fmt.Sprintf("%.1f%%", hitRate))))

	statsRow := lipgloss.JoinHorizontal(lipgloss.Top, statConns, statKeys, statCmds, statHits)

	// Render Logs
	logs := logger.MemHandler.FormatLogs()
	start := 0
	if len(logs) > 8 {
		start = len(logs) - 8
	}
	logContent := strings.Join(logs[start:], "\n")
	logBox := logBoxStyle.Render("RECENT LOGS\n\n" + logContent)

	// Layout
	layout := lipgloss.JoinVertical(lipgloss.Left,
		header,
		"\n",
		statsRow,
		"\n",
		logBox,
		"\nPress 'q' or 'ctrl+c' to quit.",
	)

	return lipgloss.Place(
		100, 30,
		lipgloss.Center, lipgloss.Center,
		layout,
	)
}

func StartApp() error {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
