package agent

import (
	"testing"

	"tetris_agents/internal/game"
)

// play feeds g the agent's actions for up to maxActions inputs.
func play(g *game.Game, a Agent, maxActions int) {
	for i := 0; i < maxActions && !g.Over; i++ {
		if !Apply(g, a.Act(g)) {
			g.HardDrop() // same fallback the UI uses
		}
	}
}

func TestHeuristicClearsLines(t *testing.T) {
	g := game.New(42)
	play(g, NewHeuristic(), 5000)
	if g.Lines < 50 {
		t.Fatalf("heuristic cleared only %d lines (over=%v)", g.Lines, g.Over)
	}
}

func TestRandomTerminates(t *testing.T) {
	g := game.New(7)
	play(g, NewRandom(), 20000)
	if g.Spawned < 5 {
		t.Fatalf("random agent only saw %d pieces", g.Spawned)
	}
}

func TestEveryAgentConstructs(t *testing.T) {
	for _, info := range All() {
		if info.Name == "" || info.New() == nil {
			t.Errorf("bad agent entry %+v", info)
		}
	}
}
