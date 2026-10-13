package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"tetris_agents/internal/agent"
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
			// Below the size where even scale 1 fits every board (possibly
			// wrapped onto rows) there is no layout to check.
			if key == "3" {
				n := len(agent.All())
				bc := boardChars(1)
				per := max(1, min(n, (w+1)/(bc+1)))
				rows := (n + per - 1) / per
				if per*bc+per-1 > w || rows*(bc+raceRowChrome)+raceFooter > h {
					continue
				}
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
	// With every agent racing, normal-size boards do not fit 160x50, so
	// the race drops to compact cells rather than overflowing.
	if w < 100 || w > 160 || h > 50 {
		t.Errorf("race uses %dx%d of 160x50", w, h)
	}
}

func TestCompactBoardsFitWhenNormalOnesCannot(t *testing.T) {
	r := NewRace()
	r.w, r.h = 120, 40 // too small for the roster at any normal cell size
	if r.size() != compact {
		t.Fatalf("size = %+v, want compact", r.size())
	}
	w, h := dims(r.View())
	if w > 120 || h > 40 {
		t.Errorf("compact race is %dx%d, want within 120x40", w, h)
	}
	r.w, r.h = 400, 100
	if r.size() == compact {
		t.Error("a huge terminal should not use compact cells")
	}
}
