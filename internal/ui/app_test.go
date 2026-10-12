package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"tetris_agents/internal/agent"
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

func press(m tea.Model, k tea.KeyMsg) tea.Model {
	m, cmd := m.Update(k)
	if cmd != nil {
		if msg, ok := cmd().(backMsg); ok {
			m, _ = m.Update(msg)
		}
	}
	return m
}

func runes(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

func TestAgentMenuListsAgentsAndReturns(t *testing.T) {
	var m tea.Model = NewApp()
	if !strings.Contains(m.View(), "2. Agents") {
		t.Fatal("main menu missing Agents")
	}
	m = press(m, runes("2"))
	v := m.View()
	for _, info := range agent.All() {
		if !strings.Contains(v, info.Name) {
			t.Errorf("agent menu missing %q", info.Name)
		}
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.(App).screen != menuScreen {
		t.Fatal("esc should go back to main menu")
	}
}

func TestAgentPlaysItself(t *testing.T) {
	var m tea.Model = NewApp()
	m = press(m, runes("2"))
	m = press(m, runes("1"))
	app := m.(App)
	if app.screen != gameScreen || app.game.agent == nil {
		t.Fatal("should be watching an agent")
	}
	for i := 0; i < 200; i++ {
		m, _ = m.Update(tickMsg(m.(App).game.gen))
	}
	if m.(App).game.g.Spawned < 5 {
		t.Fatal("agent should have placed pieces")
	}
	// User movement keys must not affect a bot game.
	before := m.(App).game.g.Current
	m = press(m, tea.KeyMsg{Type: tea.KeyLeft})
	if m.(App).game.g.Current != before {
		t.Fatal("input should be ignored in agent mode")
	}
	// Esc returns to the agent menu, cursor preserved.
	m = press(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.(App).screen != agentScreen {
		t.Fatal("esc should return to agent menu")
	}
}
