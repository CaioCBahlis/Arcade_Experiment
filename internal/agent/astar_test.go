package agent

import (
	"testing"

	"tetris_agents/internal/game"
)

func TestAStarPlaysWell(t *testing.T) {
	if testing.Short() {
		t.Skip("plays full games")
	}
	got := avgLines(NewAStar, 3, 200)
	rnd := avgLines(NewRandom, 3, 200)
	t.Logf("avg lines in 200 pieces: astar=%.1f random=%.1f", got, rnd)
	if got < 60 || got <= rnd {
		t.Fatalf("A* too weak: %.1f lines (random %.1f)", got, rnd)
	}
}

func TestAStarChoosesLegalPlacement(t *testing.T) {
	g := game.New(3)
	a := NewAStar().(*planner)
	for i := 0; i < 20 && !g.Over; i++ {
		tg := a.choose(g)
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

// Placing the I flat in the gap clears a row immediately, so with a
// one-piece plan A* must take the clear.
func TestAStarTakesAvailableClear(t *testing.T) {
	var b game.Board
	for c := 0; c < game.Width; c++ {
		if c < 3 || c > 6 {
			b[game.Height-1][c] = 1
		}
	}
	roots := candidates(&b, game.I)
	a := &astar{depth: 1, scenarios: 1, maxExpand: 100}
	i, _ := a.plan(roots, nil)
	if roots[i].lines != 1 {
		t.Fatalf("chose placement clearing %d lines, want 1", roots[i].lines)
	}
}
