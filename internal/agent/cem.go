package agent

import "tetris_agents/internal/game"

// The CEM agent is a one-piece greedy player like Heuristic, but with a
// richer board evaluation whose weights were found by the cross-entropy
// method (see cem_train.go and cmd/traincem) instead of by hand.
//
// The features are the classic Dellacherie set plus bumpiness:
// landing height, lines cleared, row transitions, column transitions, holes,
// wells, aggregate height, bumpiness and maximum height.

const nCem = 9

// cemWeights are the trained weights, in cemFeatures order, found with
// `go run ./cmd/traincem` (25 generations, population 60, fitness = mean game
// score over 1000-piece games). Paste the command's printed line to retrain.
var cemWeights = [nCem]float64{-1.4276, -0.1344, -1.0123, -1.0450, -3.9570, -0.2814, 0.7273, 0.0715, -0.1352}

// cemFeatures describes the board left by landing a piece. Values are scaled
// to roughly [0, 1] so a common initial spread works for every weight.
func cemFeatures(b *game.Board, lines int, landed game.Piece) [nCem]float64 {
	filled := func(r, c int) bool {
		switch {
		case c < 0 || c >= game.Width || r >= game.Height:
			return true // walls and floor
		case r < 0:
			return false
		}
		return b[r][c] != 0
	}

	rowTr, colTr, wells := 0, 0, 0
	for r := 0; r < game.Height; r++ {
		prev := true
		for c := 0; c < game.Width; c++ {
			cur := filled(r, c)
			if cur != prev {
				rowTr++
			}
			prev = cur
		}
		if !prev {
			rowTr++
		}
	}
	for c := 0; c < game.Width; c++ {
		prev := true // the floor
		run := 0
		for r := game.Height - 1; r >= 0; r-- {
			cur := b[r][c] != 0
			if cur != prev {
				colTr++
			}
			prev = cur
			if !cur && filled(r, c-1) && filled(r, c+1) {
				run++
				wells += run
			} else {
				run = 0
			}
		}
	}

	st := measure(b)
	landing := 0.0
	for _, cell := range landed.Cells() {
		landing += float64(game.Height - cell[0])
	}
	landing /= 4

	return [nCem]float64{
		landing / game.Height,
		float64(lines) / 4,
		float64(rowTr) / 50,
		float64(colTr) / 50,
		float64(st.holes) / 20,
		float64(wells) / 50,
		float64(st.agg) / 100,
		float64(st.bumpy) / 40,
		float64(st.max) / game.Height,
	}
}

func cemScore(w *[nCem]float64, f [nCem]float64) float64 {
	s := 0.0
	for i, x := range f {
		s += w[i] * x
	}
	return s
}

// cemPolicy returns a placement chooser that plays greedily with weights w.
func cemPolicy(w [nCem]float64) func(g *game.Game) target {
	return func(g *game.Game) target {
		ps := placements(g)
		if len(ps) == 0 {
			return target{g.Current.Rot, g.Current.Col}
		}
		best, bestV := 0, 0.0
		for i, pl := range ps {
			b, lines := g.Board.With(pl.landed)
			if v := cemScore(&w, cemFeatures(&b, lines, pl.landed)); i == 0 || v > bestV {
				best, bestV = i, v
			}
		}
		return ps[best].t
	}
}

// NewCEM returns the agent that plays with the cross-entropy-trained weights.
func NewCEM() Agent {
	return &planner{choose: cemPolicy(cemWeights)}
}
