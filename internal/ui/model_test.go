package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func key(m tea.Model, s string) tea.Model {
	var k tea.KeyMsg
	switch s {
	case " ":
		k = tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
	case "left":
		k = tea.KeyMsg{Type: tea.KeyLeft}
	default:
		k = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
	}
	m, _ = m.Update(k)
	return m
}

func TestViewRenders(t *testing.T) {
	v := New().View()
	for _, want := range []string{"TETRIS", "Score", "Next"} {
		if !strings.Contains(v, want) {
			t.Errorf("view missing %q", want)
		}
	}
}

func TestPauseIgnoresTicksAndInput(t *testing.T) {
	var m tea.Model = New()
	m = key(m, "p")
	before := m.(Model).g.Current
	m, _ = m.Update(tickMsg(m.(Model).gen))
	m = key(m, "left")
	if m.(Model).g.Current != before {
		t.Fatal("paused game should not change")
	}
	if !strings.Contains(m.View(), "PAUSED") {
		t.Error("view should show PAUSED")
	}
}

func TestStaleTickIgnored(t *testing.T) {
	var m tea.Model = New()
	before := m.(Model).g.Current
	m, _ = m.Update(tickMsg(-1))
	if m.(Model).g.Current != before {
		t.Fatal("stale tick should be dropped")
	}
}

func TestHardDropsUntilGameOverThenRestart(t *testing.T) {
	var m tea.Model = New()
	for i := 0; i < 100 && !m.(Model).g.Over; i++ {
		m = key(m, " ")
	}
	if !m.(Model).g.Over {
		t.Fatal("stacking pieces in place should end the game")
	}
	if !strings.Contains(m.View(), "GAME OVER") {
		t.Error("view should show GAME OVER")
	}
	m = key(m, "r")
	if m.(Model).g.Over {
		t.Fatal("restart should begin a new game")
	}
}
