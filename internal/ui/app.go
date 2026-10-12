package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type screen int

const (
	menuScreen screen = iota
	gameScreen
)

type menuItem struct {
	label string
	start func() Model
}

var menuItems = []menuItem{
	{"Singleplayer", New},
}

// App is the top-level model: a menu that launches game sessions.
type App struct {
	screen screen
	cursor int
	game   Model
}

// NewApp returns an App showing the main menu.
func NewApp() App { return App{} }

func (a App) Init() tea.Cmd { return nil }

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(backMsg); ok {
		a.screen = menuScreen
		return a, nil
	}
	if a.screen == gameScreen {
		m, cmd := a.game.Update(msg)
		a.game = m.(Model)
		return a, cmd
	}

	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return a, nil
	}
	switch s := k.String(); s {
	case "q", "esc", "ctrl+c":
		return a, tea.Quit
	case "up", "k":
		a.cursor = (a.cursor + len(menuItems) - 1) % len(menuItems)
	case "down", "j":
		a.cursor = (a.cursor + 1) % len(menuItems)
	case "enter", " ":
		return a.launch(a.cursor)
	default:
		// Number keys jump straight to an entry.
		if len(s) == 1 && s[0] >= '1' && int(s[0]-'1') < len(menuItems) {
			return a.launch(int(s[0] - '1'))
		}
	}
	return a, nil
}

func (a App) launch(i int) (tea.Model, tea.Cmd) {
	a.cursor = i
	a.game = menuItems[i].start()
	a.screen = gameScreen
	return a, a.game.Init()
}

func (a App) View() string {
	if a.screen == gameScreen {
		return a.game.View()
	}
	var sb strings.Builder
	sb.WriteString(titleStyle.Render("TETRIS") + "\n\n")
	for i, it := range menuItems {
		line := itoa(i+1) + ". " + it.label
		if i == a.cursor {
			sb.WriteString(selectedStyle.Render("> " + line))
		} else {
			sb.WriteString("  " + line)
		}
		sb.WriteByte('\n')
	}
	sb.WriteString("\n" + dimStyle.Render("↑/↓ select · enter start · q quit"))
	return frameStyle.Padding(1, 4).Render(sb.String()) + "\n"
}

var selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))

func itoa(n int) string { return string(rune('0' + n)) }
