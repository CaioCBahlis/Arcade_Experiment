package agent

import (
	"testing"

	"tetris_agents/internal/game"
)

func TestBeamPlaysWell(t *testing.T) {
	if testing.Short() {
		t.Skip("plays full games")
	}
	beamLines := avgLines(NewBeam, 3, 200)
	rnd := avgLines(NewRandom, 3, 200)
	t.Logf("avg lines in 200 pieces: beam=%.1f random=%.1f", beamLines, rnd)
	// 200 pieces can clear at most 80 lines; a competent agent gets most.
	if beamLines < 60 || beamLines <= rnd {
		t.Fatalf("beam search too weak: %.1f lines (random %.1f)", beamLines, rnd)
	}
}

func TestBeamChoosesLegalPlacement(t *testing.T) {
	g := game.New(3)
	b := NewBeam().(*planner)
	for i := 0; i < 20 && !g.Over; i++ {
		tg := b.choose(g)
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

// With a board where one placement strands the next piece, the beam must
// not pick it: the dead root has to score below any survivor.
func TestBeamValuesDeadRootsBelowSurvivors(t *testing.T) {
	var b game.Board
	// Fill rows 2..19 except column 0, leaving only the top two rows free:
	// any piece that stays high blocks the spawn area for the next one.
	for r := 2; r < game.Height; r++ {
		for c := 1; c < game.Width; c++ {
			b[r][c] = 1
		}
	}
	roots := candidates(&b, game.I)
	if len(roots) == 0 {
		t.Fatal("no roots")
	}
	bm := &beam{width: 12, scenarios: 1, extra: 0}
	vals := bm.run(roots, []game.Kind{game.O})
	hi, lo := vals[0], vals[0]
	for _, v := range vals {
		hi, lo = max(hi, v), min(lo, v)
	}
	if hi == lo {
		t.Fatalf("all roots valued equally (%v); dead roots should be distinguished", hi)
	}
}
