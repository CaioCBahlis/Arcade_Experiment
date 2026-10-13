package agent

import (
	"testing"

	"tetris_agents/internal/game"
)

func TestMCTSPlaysWell(t *testing.T) {
	if testing.Short() {
		t.Skip("plays full games")
	}
	mctsLines := avgLines(NewMCTS, 3, 200)
	rnd := avgLines(NewRandom, 3, 200)
	t.Logf("avg lines in 200 pieces: mcts=%.1f random=%.1f", mctsLines, rnd)
	if mctsLines < 60 || mctsLines <= rnd {
		t.Fatalf("MCTS too weak: %.1f lines (random %.1f)", mctsLines, rnd)
	}
}

func TestMCTSChoosesLegalPlacement(t *testing.T) {
	g := game.New(3)
	m := NewMCTS().(*planner)
	for i := 0; i < 20 && !g.Over; i++ {
		tg := m.choose(g)
		ok := false
		for _, p := range placements(g) {
			if p.t == tg {
				ok = true
			}
		}
		if !ok {
			t.Fatalf("piece %d: target %+v is not a legal placement", i, tg)
		}
		g.HardDrop()
	}
}
