package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"tetris_agents/internal/agent"
)

func TestRaceUsesSamePieces(t *testing.T) {
	r := NewRace()
	if len(r.racers) != len(agent.All()) {
		t.Fatalf("got %d racers, want %d", len(r.racers), len(agent.All()))
	}
	for _, rc := range r.racers {
		if rc.g.Current.Kind != r.racers[0].g.Current.Kind || rc.g.Next != r.racers[0].g.Next {
			t.Fatal("racers must receive identical piece sequences")
		}
	}
}

func TestRaceHasNoPieceCapAndSpeedsUp(t *testing.T) {
	var m tea.Model = NewRace()
	start := m.(Race).interval()
	// Long enough that the old 200-piece cap would have ended the race for
	// anyone still alive.
	for i := 0; i < 20000; i++ {
		m, _ = m.Update(tickMsg(m.(Race).gen))
		r := m.(Race)
		if r.done() {
			break
		}
	}
	r := m.(Race)
	best := 0
	for _, rc := range r.racers {
		best = max(best, rc.placed())
	}
	if best <= 200 {
		t.Fatalf("leader placed only %d pieces; race should run past 200", best)
	}
	if r.round() <= 1 || r.interval() >= start {
		t.Errorf("round %d interval %v should be faster than %v", r.round(), r.interval(), start)
	}
	if r.interval() < raceMinInterval {
		t.Errorf("interval %v below floor %v", r.interval(), raceMinInterval)
	}
}

func TestRaceFinishesWhenAllTopOut(t *testing.T) {
	r := NewRace()
	for i, rc := range r.racers {
		if r.done() {
			t.Fatal("race should not be done while racers are alive")
		}
		rc.g.Score = 100 * (i + 1)
		rc.g.Over = true
	}
	if !r.done() {
		t.Fatal("race should be done once everyone topped out")
	}
	v := r.View()
	if !strings.Contains(v, "Winner:") {
		t.Errorf("finished race should name a winner:\n%s", v)
	}
	last := r.racers[len(r.racers)-1].info.Name
	if !strings.Contains(v, "Winner: "+last) {
		t.Errorf("highest score should win, want %q:\n%s", last, v)
	}
}

func TestRaceIntervalSchedule(t *testing.T) {
	r := NewRace()
	if r.round() != 1 || r.interval() != raceInterval {
		t.Fatalf("start: round %d interval %v", r.round(), r.interval())
	}
	r.racers[0].g.Spawned = roundPieces*3 + 1 // 75 placed
	if r.round() != 4 {
		t.Fatalf("round = %d, want 4", r.round())
	}
	if r.interval() >= raceInterval {
		t.Fatal("interval should have shrunk")
	}
	r.racers[0].g.Spawned = 1_000_000
	if r.interval() != raceMinInterval {
		t.Fatalf("interval = %v, want floor %v", r.interval(), raceMinInterval)
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
