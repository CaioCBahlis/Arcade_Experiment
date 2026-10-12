package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func resize(m tea.Model, w, h int) tea.Model {
	m, _ = m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return m
}

// dims returns the widest line and the line count of a rendered view.
func dims(v string) (w, h int) {
	lines := strings.Split(strings.TrimRight(v, "\n"), "\n")
	for _, l := range lines {
		w = max(w, ansi.StringWidth(l))
	}
	return w, len(lines)
}

func TestScaleGrowsWithTerminal(t *testing.T) {
	small := Model{w: 80, h: 24}.scale()
	big := Model{w: 200, h: 60}.scale()
	if small != 1 {
		t.Errorf("80x24 scale = %d, want 1", small)
	}
	if big <= small {
		t.Errorf("200x60 scale = %d, should exceed %d", big, small)
	}
	if got := (Model{}).scale(); got != 1 {
		t.Errorf("unknown size scale = %d, want 1", got)
	}
}

func TestViewsFitTerminal(t *testing.T) {
	sizes := [][2]int{{80, 24}, {120, 40}, {200, 60}, {300, 90}}
	for _, sz := range sizes {
		w, h := sz[0], sz[1]
		for _, key := range []string{"", "1", "2", "3"} {
			var m tea.Model = NewApp()
			m = resize(m, w, h)
			if key == "2" { // agent menu, then first agent
				m = press(m, runes("2"))
				m = press(m, runes("1"))
			} else if key != "" {
				m = press(m, runes(key))
			}
			gw, gh := dims(m.View())
			// 80x24 is below the minimum a 3-board race needs; only
			// require that bigger terminals are respected.
			if key == "3" && w < 100 {
				continue
			}
			if gw > w || gh > h {
				t.Errorf("%dx%d key %q: view is %dx%d", w, h, key, gw, gh)
			}
		}
	}
}

// content measures a screen before centering pads it to the terminal size.
func content(m tea.Model) (w, h int) { return dims(m.(App).screenView()) }

func TestBigTerminalFillsMostOfScreen(t *testing.T) {
	var m tea.Model = NewApp()
	m = resize(m, 160, 50)
	m = press(m, runes("1"))
	w, h := content(m)
	t.Logf("single-player at 160x50 uses %dx%d", w, h)
	if h < 42 { // scale 2: 40 board rows + frame
		t.Errorf("single-player uses %d of 50 rows", h)
	}

	m = resize(NewApp(), 160, 50)
	m = press(m, runes("3"))
	w, h = content(m)
	t.Logf("race at 160x50 uses %dx%d", w, h)
	if h < 42 || w < 120 {
		t.Errorf("race uses %dx%d of 160x50", w, h)
	}
}
