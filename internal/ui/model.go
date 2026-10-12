// Package ui renders a game.Game as a Bubble Tea program.
package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"tetris_agents/internal/game"
)

type tickMsg int // carries the generation it was scheduled in

// Model is the Bubble Tea model for a Tetris session.
type Model struct {
	g      *game.Game
	paused bool
	gen    int // bumped whenever the tick chain must restart
}

// New returns a model with a freshly seeded game.
func New() Model {
	return Model{g: game.New(time.Now().UnixNano())}
}

func (m Model) tick() tea.Cmd {
	gen := m.gen
	return tea.Tick(m.g.Interval(), func(time.Time) tea.Msg { return tickMsg(gen) })
}

func (m Model) Init() tea.Cmd { return m.tick() }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		if int(msg) != m.gen || m.paused || m.g.Over {
			return m, nil
		}
		m.g.Tick()
		return m, m.tick()

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "r":
			m.g = game.New(time.Now().UnixNano())
			m.paused = false
			m.gen++
			return m, m.tick()
		case "p":
			if m.g.Over {
				return m, nil
			}
			m.paused = !m.paused
			m.gen++
			if m.paused {
				return m, nil
			}
			return m, m.tick()
		}
		if m.paused || m.g.Over {
			return m, nil
		}
		switch msg.String() {
		case "left", "h", "a":
			m.g.MoveLeft()
		case "right", "l", "d":
			m.g.MoveRight()
		case "up", "k", "w", "x":
			m.g.Rotate()
		case "down", "j", "s":
			m.g.SoftDrop()
		case " ":
			m.g.HardDrop()
		}
	}
	return m, nil
}

var kindColors = [...]lipgloss.Color{
	game.I: "51",  // cyan
	game.O: "226", // yellow
	game.T: "129", // purple
	game.S: "46",  // green
	game.Z: "196", // red
	game.J: "27",  // blue
	game.L: "208", // orange
}

const (
	block  = "██"
	ghost  = "░░"
	empty  = "  "
	dotted = " ·"
)

var (
	frameStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240"))
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	overStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("196"))
)

func paint(kind game.Kind, s string) string {
	return lipgloss.NewStyle().Foreground(kindColors[kind]).Render(s)
}

func (m Model) boardView() string {
	var cur, gh [game.Height][game.Width]bool
	if !m.g.Over {
		for _, c := range m.g.Ghost().Cells() {
			if c[0] >= 0 {
				gh[c[0]][c[1]] = true
			}
		}
	}
	for _, c := range m.g.Current.Cells() {
		if c[0] >= 0 && c[0] < game.Height && c[1] >= 0 && c[1] < game.Width {
			cur[c[0]][c[1]] = true
		}
	}

	var sb strings.Builder
	for r := 0; r < game.Height; r++ {
		for c := 0; c < game.Width; c++ {
			switch v := m.g.Board[r][c]; {
			case v != 0:
				sb.WriteString(paint(game.Kind(v-1), block))
			case cur[r][c]:
				sb.WriteString(paint(m.g.Current.Kind, block))
			case gh[r][c]:
				sb.WriteString(paint(m.g.Current.Kind, ghost))
			default:
				sb.WriteString(dimStyle.Render(dotted))
			}
		}
		if r < game.Height-1 {
			sb.WriteByte('\n')
		}
	}
	return frameStyle.Render(sb.String())
}

func (m Model) nextView() string {
	p := game.Piece{Kind: m.g.Next}
	var grid [4][4]bool
	for _, c := range p.Offsets() {
		grid[c[0]][c[1]] = true
	}
	var lines []string
	for r := 0; r < 4; r++ {
		var sb strings.Builder
		blank := true
		for c := 0; c < 4; c++ {
			if grid[r][c] {
				sb.WriteString(paint(m.g.Next, block))
				blank = false
			} else {
				sb.WriteString(empty)
			}
		}
		if !blank {
			lines = append(lines, sb.String())
		}
	}
	return strings.Join(lines, "\n")
}

func (m Model) sideView() string {
	var status string
	switch {
	case m.g.Over:
		status = overStyle.Render("GAME OVER") + "\n" + dimStyle.Render("r: play again")
	case m.paused:
		status = titleStyle.Render("PAUSED")
	}
	parts := []string{
		titleStyle.Render("TETRIS"),
		"",
		dimStyle.Render("Next"),
		m.nextView(),
		"",
		fmt.Sprintf("Score  %d", m.g.Score),
		fmt.Sprintf("Lines  %d", m.g.Lines),
		fmt.Sprintf("Level  %d", m.g.Level()),
		"",
		dimStyle.Render("←/→  move"),
		dimStyle.Render("↑    rotate"),
		dimStyle.Render("↓    soft drop"),
		dimStyle.Render("space hard drop"),
		dimStyle.Render("p    pause"),
		dimStyle.Render("r    restart"),
		dimStyle.Render("q    quit"),
	}
	if status != "" {
		parts = append(parts, "", status)
	}
	return lipgloss.NewStyle().PaddingLeft(2).Render(strings.Join(parts, "\n"))
}

func (m Model) View() string {
	return lipgloss.JoinHorizontal(lipgloss.Top, m.boardView(), m.sideView()) + "\n"
}
