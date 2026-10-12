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
	st := measure(b)
	return wHeight*float64(st.agg) + wLines*float64(lines) + wHoles*float64(st.holes) + wBumpy*float64(st.bumpy)
}

// stats summarizes the shape of a board.
type stats struct {
	agg   int // sum of column heights
	max   int // tallest column
	holes int // empty cells with a filled cell above them
	bumpy int // sum of height differences between adjacent columns
}

func measure(b *game.Board) stats {
	var heights [game.Width]int
	var st stats
	for c := 0; c < game.Width; c++ {
		seen := false
		for r := 0; r < game.Height; r++ {
			if b[r][c] != 0 {
				if !seen {
					heights[c] = game.Height - r
					seen = true
				}
			} else if seen {
				st.holes++
			}
		}
	}
	for c, h := range heights {
		st.agg += h
		if h > st.max {
			st.max = h
		}
		if c > 0 {
			d := h - heights[c-1]
			if d < 0 {
				d = -d
			}
			st.bumpy += d
		}
	}
	return st
}
