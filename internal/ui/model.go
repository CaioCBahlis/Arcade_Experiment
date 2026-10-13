// Package ui renders a game.Game as a Bubble Tea program.
package ui

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"tetris_agents/internal/agent"
	"tetris_agents/internal/game"
)

// agentInterval is how often a bot issues an input; fast enough to feel
// live, slow enough to watch pieces move.
const agentInterval = 60 * time.Millisecond

type tickMsg int // carries the generation it was scheduled in

// backMsg asks the parent app to return to the menu.
type backMsg struct{}

var genCounter int64

// nextGen returns a tick generation unique across every game in the process,
// so ticks left over from an abandoned game can never match a new one.
func nextGen() int { return int(atomic.AddInt64(&genCounter, 1)) }

// Model is the Bubble Tea model for a Tetris session.
type Model struct {
	g      *game.Game
	paused bool
	gen    int // bumped whenever the tick chain must restart

	info  *agent.Info // non-nil when a bot is playing
	agent agent.Agent

	w, h int // terminal size; zero until the first WindowSizeMsg
}

// sideMinWidth is the width reserved for the info panel under the board.
const sideMinWidth = 28

// sideRows is the height of the info panel under the board at scale s: the
// fixed lines (title, agent, stats, controls, status) plus the next piece,
// which is at most two cells tall.
func sideRows(s int) int { return 20 + 2*s }

// scale picks the largest board scale that fits the terminal.
func (m Model) scale() int {
	return fitScale(m.w, m.h, func(s int) (int, int) {
		return max(sideMinWidth, boardChars(s)), boardChars(s) + sideRows(s)
	})
}

// New returns a model with a freshly seeded game.
func New() Model {
	return Model{g: game.New(time.Now().UnixNano()), gen: nextGen()}
}

// NewWithAgent returns a model where the given bot plays instead of the user.
func NewWithAgent(info agent.Info) Model {
	m := New()
	m.info = &info
	m.agent = info.New()
	return m
}

func (m Model) tick() tea.Cmd {
	gen := m.gen
	d := m.g.Interval()
	if m.agent != nil {
		d = agentInterval
	}
	return tea.Tick(d, func(time.Time) tea.Msg { return tickMsg(gen) })
}

func (m Model) Init() tea.Cmd { return m.tick() }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		return m, nil

	case tickMsg:
		if int(msg) != m.gen || m.paused || m.g.Over {
			return m, nil
		}
		if m.agent != nil {
			if !agent.Apply(m.g, m.agent.Act(m.g)) {
				m.g.HardDrop() // blocked: give up on the plan, don't stall
			}
		} else {
			m.g.Tick()
		}
		return m, m.tick()

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			return m, func() tea.Msg { return backMsg{} }
		case "r":
			m.g = game.New(time.Now().UnixNano())
			if m.info != nil {
				m.agent = m.info.New()
			}
			m.paused = false
			m.gen = nextGen()
			return m, m.tick()
		case "p":
			if m.g.Over {
				return m, nil
			}
			m.paused = !m.paused
			m.gen = nextGen()
			if m.paused {
				return m, nil
			}
			return m, m.tick()
		}
		if m.paused || m.g.Over || m.agent != nil {
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
	block = "██"
	ghost = "░░"
	empty = "  "
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

func (m Model) boardView() string { return renderBoard(m.g, true, scaled(m.scale())) }

// renderBoard draws g's playfield inside a frame with cells of size cs. The
// ghost piece, when shown, marks where the current piece would land.
func renderBoard(g *game.Game, showGhost bool, cs cellSize) string {
	var cur, gh [game.Height][game.Width]bool
	if showGhost && !g.Over {
		for _, c := range g.Ghost().Cells() {
			if c[0] >= 0 {
				gh[c[0]][c[1]] = true
			}
		}
	}
	for _, c := range g.Current.Cells() {
		if c[0] >= 0 && c[0] < game.Height && c[1] >= 0 && c[1] < game.Width {
			cur[c[0]][c[1]] = true
		}
	}

	filled := strings.Repeat("█", cs.w)
	shadow := strings.Repeat("░", cs.w)
	dot := strings.Repeat(" ", cs.w)
	if cs.w >= 2 {
		dot = dimStyle.Render(" ·") + strings.Repeat(" ", cs.w-2)
	} else {
		dot = dimStyle.Render("·")
	}
	blank := strings.Repeat(" ", cs.w)
	rows := make([]string, 0, game.Height*cs.h)
	for r := 0; r < game.Height; r++ {
		for sub := 0; sub < cs.h; sub++ {
			var sb strings.Builder
			for c := 0; c < game.Width; c++ {
				switch v := g.Board[r][c]; {
				case v != 0:
					sb.WriteString(paint(game.Kind(v-1), filled))
				case cur[r][c]:
					sb.WriteString(paint(g.Current.Kind, filled))
				case gh[r][c]:
					sb.WriteString(paint(g.Current.Kind, shadow))
				case sub == 0:
					// One dot per empty cell, in its top-left corner.
					sb.WriteString(dot)
				default:
					sb.WriteString(blank)
				}
			}
			rows = append(rows, sb.String())
		}
	}
	return frameStyle.Render(strings.Join(rows, "\n"))
}

func (m Model) nextView() string {
	scale := m.scale()
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
				sb.WriteString(paint(m.g.Next, strings.Repeat(block, scale)))
				blank = false
			} else {
				sb.WriteString(strings.Repeat(empty, scale))
			}
		}
		if !blank {
			for i := 0; i < scale; i++ {
				lines = append(lines, sb.String())
			}
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
	parts := []string{titleStyle.Render("TETRIS")}
	if m.info != nil {
		parts = append(parts, dimStyle.Render("Agent: ")+m.info.Name)
	}
	parts = append(parts,
		"",
		dimStyle.Render("Next"),
		m.nextView(),
		"",
		fmt.Sprintf("Score  %d", m.g.Score),
		fmt.Sprintf("Lines  %d", m.g.Lines),
		fmt.Sprintf("Level  %d", m.g.Level()),
		"",
	)
	if m.agent == nil {
		parts = append(parts,
			dimStyle.Render("←/→  move"),
			dimStyle.Render("↑    rotate"),
			dimStyle.Render("↓    soft drop"),
			dimStyle.Render("space hard drop"),
		)
	}
	parts = append(parts,
		dimStyle.Render("p    pause"),
		dimStyle.Render("r    restart"),
		dimStyle.Render("esc  menu"),
		dimStyle.Render("q    quit"),
	)
	if status != "" {
		parts = append(parts, "", status)
	}
	return lipgloss.NewStyle().PaddingLeft(1).Render(strings.Join(parts, "\n"))
}

func (m Model) View() string {
	return lipgloss.JoinVertical(lipgloss.Left, m.boardView(), m.sideView()) + "\n"
}
