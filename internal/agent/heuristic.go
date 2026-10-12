package agent

import "tetris_agents/internal/game"

// Weights from the well-known "El-Tetris" style evaluation.
const (
	wHeight = -0.510066
	wLines  = 0.760666
	wHoles  = -0.35663
	wBumpy  = -0.184483
)

// NewHeuristic returns an agent that greedily maximizes a board score.
func NewHeuristic() Agent {
	return &planner{choose: func(g *game.Game) target {
		best, bestScore := target{g.Current.Rot, g.Current.Col}, -1e18
		for _, pl := range placements(g) {
			b, lines := g.Board.With(pl.landed)
			if s := evaluate(&b, lines); s > bestScore {
				best, bestScore = pl.t, s
			}
		}
		return best
	}}
}

func evaluate(b *game.Board, lines int) float64 {
	var heights [game.Width]int
	holes := 0
	for c := 0; c < game.Width; c++ {
		seen := false
		for r := 0; r < game.Height; r++ {
			if b[r][c] != 0 {
				if !seen {
					heights[c] = game.Height - r
					seen = true
				}
			} else if seen {
				holes++
			}
		}
	}
	agg, bumpy := 0, 0
	for c, h := range heights {
		agg += h
		if c > 0 {
			d := h - heights[c-1]
			if d < 0 {
				d = -d
			}
			bumpy += d
		}
	}
	return wHeight*float64(agg) + wLines*float64(lines) + wHoles*float64(holes) + wBumpy*float64(bumpy)
}
