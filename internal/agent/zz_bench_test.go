package agent

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"tetris_agents/internal/game"
)

func TestZZBeam(t *testing.T) {
	const n, seeds = 1500, 10
	type res struct {
		score, lines, pieces int
		dead                 bool
	}
	out := make([]res, seeds)
	var wg sync.WaitGroup
	start := time.Now()
	for i := 0; i < seeds; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			g := game.New(int64(700 + i))
			a := NewBeam()
			for g.Spawned <= n && !g.Over {
				if !Apply(g, a.Act(g)) {
					g.HardDrop()
				}
			}
			out[i] = res{g.Score, g.Lines, g.Spawned - 1, g.Over}
		}(i)
	}
	wg.Wait()
	var sc, ln, pc, dead int
	for _, r := range out {
		sc, ln, pc = sc+r.score, ln+r.lines, pc+r.pieces
		if r.dead {
			dead++
		}
	}
	fmt.Printf("n=%d beam       avg score=%7d lines=%6.1f pieces=%6.1f dead=%d/%d  %v\n", n, sc/seeds, float64(ln)/seeds, float64(pc)/seeds, dead, seeds, time.Since(start).Round(time.Second))
}
