package agent

import (
	"testing"
	"time"

	"tetris_agents/internal/game"
)

// avgLines plays n seeded games with a fresh agent from mk, capped at
// maxPieces each, and returns the mean lines cleared.
func avgLines(mk func() Agent, n, maxPieces int) float64 {
	total := 0
	for seed := int64(100); seed < 100+int64(n); seed++ {
		g := game.New(seed)
		a := mk()
		for g.Spawned <= maxPieces && !g.Over {
			if !Apply(g, a.Act(g)) {
				g.HardDrop()
			}
		}
		total += g.Lines
	}
	return float64(total) / float64(n)
}

func TestRLLearns(t *testing.T) {
	start := time.Now()
	WarmRL()
	t.Logf("training took %v, weights %v", time.Since(start), trained)

	untrained := func() Agent {
		var w weights
		return &planner{choose: (&policy{w: &w}).choose}
	}
	rl := avgLines(NewRL, 10, 300)
	base := avgLines(untrained, 10, 300)
	rnd := avgLines(NewRandom, 10, 300)
	heur := avgLines(NewHeuristic, 10, 300)
	t.Logf("avg lines in 300 pieces: rl=%.1f untrained=%.1f random=%.1f heuristic=%.1f", rl, base, rnd, heur)
	if rl < 0.8*heur || rl <= rnd || rl <= base {
		t.Fatalf("RL agent did not learn: rl=%.1f random=%.1f untrained=%.1f heuristic=%.1f", rl, rnd, base, heur)
	}
}
