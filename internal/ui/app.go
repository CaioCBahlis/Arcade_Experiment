package ui

import (
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"tetris_agents/internal/agent"
)

type screen int

const (
	menuScreen screen = iota
	agentScreen
	gameScreen
	raceScreen
)

// entry is one selectable menu row.
type entry struct {
	label string
	desc  string
	run   func(a App) (App, tea.Cmd)
}

// App is the top-level model: menus that launch game sessions.
type App struct {
	screen screen
	cursor int
	game   Model
	race   Race

	// Where Esc in a game returns to, and which row was selected there.
	back       screen
	backCursor int
}

// NewApp returns an App showing the main menu.
func NewApp() App { return App{} }

func (a App) Init() tea.Cmd { return nil }

func (a App) entries() []entry {
	if a.screen == agentScreen {
		var es []entry
		for _, info := range agent.All() {
			info := info
			es = append(es, entry{info.Name, info.Desc, func(a App) (App, tea.Cmd) {
				return a.launch(NewWithAgent(info))
			}})
		}
		return es
	}
	return []entry{
		{"Singleplayer", "", func(a App) (App, tea.Cmd) { return a.launch(New()) }},
		{"Agents", "", func(a App) (App, tea.Cmd) {
			a.screen, a.cursor = agentScreen, 0
			return a, nil
		}},
		{"Competition", "All agents race side by side on the same pieces", func(a App) (App, tea.Cmd) {
			a.back, a.backCursor = a.screen, a.cursor
			a.race = NewRace()
			a.screen = raceScreen
			return a, a.race.Init()
		}},
	}
}

func (a App) launch(m Model) (App, tea.Cmd) {
	a.back, a.backCursor = a.screen, a.cursor
	a.game = m
	a.screen = gameScreen
	return a, a.game.Init()
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(backMsg); ok {
		a.screen, a.cursor = a.back, a.backCursor
		return a, nil
	}
	switch a.screen {
	case gameScreen:
		m, cmd := a.game.Update(msg)
		a.game = m.(Model)
		return a, cmd
	case raceScreen:
		m, cmd := a.race.Update(msg)
		a.race = m.(Race)
		return a, cmd
	}

	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return a, nil
	}
	es := a.entries()
	switch s := k.String(); s {
	case "ctrl+c":
		return a, tea.Quit
	case "q", "esc":
		if a.screen == agentScreen {
			a.screen, a.cursor = menuScreen, 1 // back onto "Agents"
			return a, nil
		}
		return a, tea.Quit
	case "up", "k":
		a.cursor = (a.cursor + len(es) - 1) % len(es)
	case "down", "j":
		a.cursor = (a.cursor + 1) % len(es)
	case "enter", " ":
		return es[a.cursor].run(a)
	default:
		// Number keys jump straight to an entry.
		if n, err := strconv.Atoi(s); err == nil && n >= 1 && n <= len(es) {
			a.cursor = n - 1
			return es[a.cursor].run(a)
		}
	}
	return a, nil
}

var selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))

func (a App) View() string {
	switch a.screen {
	case gameScreen:
		return a.game.View()
	case raceScreen:
		return a.race.View()
	}
	title, hint := "TETRIS", "↑/↓ select · enter start · q quit"
	if a.screen == agentScreen {
		title, hint = "AGENTS", "↑/↓ select · enter watch · esc back"
	}
	var sb strings.Builder
	sb.WriteString(titleStyle.Render(title) + "\n\n")
	for i, e := range a.entries() {
		line := strconv.Itoa(i+1) + ". " + e.label
		if i == a.cursor {
			sb.WriteString(selectedStyle.Render("> " + line))
		} else {
			sb.WriteString("  " + line)
		}
		sb.WriteByte('\n')
		if e.desc != "" {
			sb.WriteString("     " + dimStyle.Render(e.desc) + "\n")
		}
	}
	sb.WriteString("\n" + dimStyle.Render(hint))
	return frameStyle.Padding(1, 4).Render(sb.String()) + "\n"
}
