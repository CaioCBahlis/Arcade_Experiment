package agent

import (
	"fmt"
	"testing"
	"time"

	"tetris_agents/internal/game"
)

func TestZZCompare(t *testing.T) {
	for _, n := range []int{1500} {
		for _, c := range []struct {
			name string
			mk   func() Agent
		}{{"expectimax", NewExpectimax}, {"heuristic", NewHeuristic}} {
			var sc, ln, dead int
			start := time.Now()
			const seeds = 8
			for s := int64(500); s < 500+seeds; s++ {
				g := game.New(s)
				a := c.mk()
				for g.Spawned <= n && !g.Over {
					if !Apply(g, a.Act(g)) {
						g.HardDrop()
					}
				}
				sc += g.Score
				ln += g.Lines
				if g.Over {
					dead++
				}
			}
			fmt.Printf("n=%d %-10s score=%6d lines=%5.1f dead=%d/%d  %v\n", n, c.name, sc/seeds, float64(ln)/seeds, dead, seeds, time.Since(start).Round(time.Millisecond))
		}
	}
}
