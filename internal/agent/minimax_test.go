package agent

import (
	"math"
	"testing"

	"tetris_agents/internal/game"
)

func TestMinimaxPlaysWell(t *testing.T) {
	if testing.Short() {
		t.Skip("plays full games")
	}
	got := avgLines(NewMinimax, 3, 200)
	rnd := avgLines(NewRandom, 3, 200)
	t.Logf("avg lines in 200 pieces: minimax=%.1f random=%.1f", got, rnd)
	if got < 60 || got <= rnd {
		t.Fatalf("minimax too weak: %.1f lines (random %.1f)", got, rnd)
	}
}

// Alpha-beta must change only the speed, never the answer: searching with a
// full window must match searching with a pruned root window.
func TestMinimaxAlphaBetaMatchesPlainMinimax(t *testing.T) {
	g := game.New(11)
	m := &minimax{beam: 6}
	for i := 0; i < 8 && !g.Over; i++ {
		cs := candidates(&g.Board, g.Current.Kind)
		rest := []game.Kind{g.Next}

		// Plain: every root child evaluated with an open window.
		plain, plainV := cs[0].t, math.Inf(-1)
		for j := range cs {
			r := lineReward(cs[j].lines)
			v := r + m.after(&cs[j].board, rest, 1, math.Inf(-1), math.Inf(1))
			if v > plainV {
				plain, plainV = cs[j].t, v
			}
		}
		got := m.choose(g)
		if got != plain {
			// Ties may resolve to a different move; require equal value.
			var gotV float64
			for j := range cs {
				if cs[j].t == got {
					gotV = lineReward(cs[j].lines) + m.after(&cs[j].board, rest, 1, math.Inf(-1), math.Inf(1))
				}
			}
			if gotV != plainV {
				t.Fatalf("piece %d: alpha-beta chose value %v, plain minimax %v", i, gotV, plainV)
			}
		}
		g.HardDrop()
	}
}

func TestMinimaxIsPessimisticVersusExpectimax(t *testing.T) {
	// On an empty board with a known next piece, minimax's value for any
	// placement can never exceed the value expectimax assigns, since a min
	// over pieces is at most the average over them.
	var b game.Board
	cs := candidates(&b, game.T)
	m := &minimax{beam: 1000}
	e := &expectimax{beam: 1000}
	for _, c := range cs[:6] {
		mv := m.after(&c.board, []game.Kind{game.O}, 1, math.Inf(-1), math.Inf(1))
		ev := e.future(&c.board, []game.Kind{game.O}, true)
		if mv > ev+1e-9 {
			t.Fatalf("minimax %v exceeds expectimax %v", mv, ev)
		}
	}
}
