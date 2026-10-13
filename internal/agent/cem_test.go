package agent

import (
	"testing"

	"tetris_agents/internal/game"
)

func TestCEMFeatures(t *testing.T) {
	var empty game.Board
	f := cemFeatures(&empty, 0, game.Piece{Kind: game.O, Row: game.Height - 2})
	// An empty board has no holes, wells or transitions inside rows.
	if f[4] != 0 || f[5] != 0 {
		t.Errorf("empty board: holes %v wells %v, want 0", f[4], f[5])
	}

	// One covered gap: a hole, and exactly one column with extra transitions.
	var b game.Board
	b[game.Height-1][3] = 0
	b[game.Height-1][2], b[game.Height-1][4] = 1, 1
	b[game.Height-2][2], b[game.Height-2][3], b[game.Height-2][4] = 1, 1, 1
	f = cemFeatures(&b, 0, game.Piece{Kind: game.O, Row: 0})
	if f[4] != 1.0/20 {
		t.Errorf("holes feature = %v, want %v", f[4], 1.0/20)
	}

	// A single-width well three deep: 1+2+3 = 6 well units.
	var w game.Board
	for r := game.Height - 3; r < game.Height; r++ {
		w[r][3], w[r][5] = 1, 1
	}
	f = cemFeatures(&w, 0, game.Piece{Kind: game.O, Row: 0})
	if f[5] != 6.0/50 {
		t.Errorf("wells feature = %v, want %v", f[5], 6.0/50)
	}
}

func TestCEMTrainingImproves(t *testing.T) {
	if testing.Short() {
		t.Skip("trains")
	}
	opt := CEMOptions{Population: 24, Generations: 6, EliteFrac: 0.25, Games: 2, PieceCap: 250, Seed: 3}
	seeds := []int64{101, 102, 103}
	var zero [nCem]float64
	before := cemFitness(zero, seeds, opt.PieceCap)
	w := TrainCEM(opt)
	after := cemFitness(w, seeds, opt.PieceCap)
	t.Logf("held-out mean score: %.0f -> %.0f", before, after)
	if after <= before*2 {
		t.Fatalf("training did not improve the weights enough: %.0f -> %.0f", before, after)
	}
}

func TestCEMAgentBeatsRandom(t *testing.T) {
	if testing.Short() {
		t.Skip("plays full games")
	}
	got, rnd := avgLines(NewCEM, 3, 200), avgLines(NewRandom, 3, 200)
	t.Logf("avg lines in 200 pieces: cem=%.1f random=%.1f", got, rnd)
	if got < 60 || got <= rnd {
		t.Fatalf("cem agent too weak: %.1f lines (random %.1f)", got, rnd)
	}
}
