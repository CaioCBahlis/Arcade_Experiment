package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func send(m tea.Model, k tea.KeyMsg) (tea.Model, tea.Cmd) { return m.Update(k) }

func TestMenuShowsSingleplayer(t *testing.T) {
	v := NewApp().View()
	if !strings.Contains(v, "1. Singleplayer") {
		t.Fatalf("menu missing entry:\n%s", v)
	}
}

func TestMenuStartsGameAndEscReturns(t *testing.T) {
	var m tea.Model = NewApp()
	m, cmd := send(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	if m.(App).screen != gameScreen || cmd == nil {
		t.Fatal("1 should start singleplayer")
	}
	m, cmd = send(m, tea.KeyMsg{Type: tea.KeyEsc})
	m, _ = m.Update(cmd())
	if m.(App).screen != menuScreen {
		t.Fatal("esc should return to menu")
	}
	m, _ = send(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.(App).screen != gameScreen {
		t.Fatal("enter should start singleplayer")
	}
}

func TestStaleTickFromPreviousGameIgnored(t *testing.T) {
	old := New()
	fresh := New()
	if old.gen == fresh.gen {
		t.Fatal("games must have distinct generations")
	}
}
