package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"tetris_agents/internal/agent"
)

func TestRaceUsesSamePiecesAndFinishes(t *testing.T) {
	r := NewRace()
	if len(r.racers) != len(agent.All()) {
		t.Fatalf("got %d racers, want %d", len(r.racers), len(agent.All()))
	}
	first := r.racers[0].g.Current.Kind
	for _, rc := range r.racers {
		if rc.g.Current.Kind != first || rc.g.Next != r.racers[0].g.Next {
			t.Fatal("racers must receive identical piece sequences")
		}
	}

	var m tea.Model = r
	for i := 0; i < 200000 && !m.(Race).done(); i++ {
		m, _ = m.Update(tickMsg(m.(Race).gen))
	}
	r = m.(Race)
	if !r.done() {
		t.Fatal("race never finished")
	}
	v := r.View()
	if !strings.Contains(v, "Winner:") {
		t.Errorf("finished race should name a winner:\n%s", v)
	}
	for _, info := range agent.All() {
		if !strings.Contains(v, info.Name) {
			t.Errorf("view missing %q", info.Name)
		}
	}
}

func TestMenuRaceStartsAndEscReturns(t *testing.T) {
	var m tea.Model = NewApp()
	if !strings.Contains(m.View(), "3. Competition") {
		t.Fatal("main menu missing Competition")
	}
	m = press(m, runes("3"))
	if m.(App).screen != raceScreen {
		t.Fatal("3 should start the race")
	}
	m = press(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.(App).screen != menuScreen {
		t.Fatal("esc should return to the menu")
	}
}

func TestRacePauseStopsProgress(t *testing.T) {
	var m tea.Model = NewRace()
	m = press(m, runes("p"))
	before := m.(Race).racers[0].g.Current
	m, _ = m.Update(tickMsg(m.(Race).gen))
	if m.(Race).racers[0].g.Current != before {
		t.Fatal("paused race should not advance")
	}
}
