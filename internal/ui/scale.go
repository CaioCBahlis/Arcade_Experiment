package ui

import "tetris_agents/internal/game"

// maxScale keeps boards from becoming absurdly large on huge terminals.
const maxScale = 6

// cellSize is how many characters wide and rows tall one board cell is.
type cellSize struct{ w, h int }

// scaled is the size used at integer scale s: 2s wide and s tall, so the
// board is roughly square on screen.
func scaled(s int) cellSize { return cellSize{2 * s, s} }

// compact is the smallest size: one character per cell.
var compact = cellSize{1, 1}

// boardW and boardH are the framed board's width and height in characters.
func (c cellSize) boardW() int { return game.Width*c.w + 2 }
func (c cellSize) boardH() int { return game.Height*c.h + 2 }

// boardChars is the framed board's width and height at scale s, which are
// equal because cells are 2s wide and s tall.
func boardChars(s int) int { return scaled(s).boardW() }

// fitScale returns the largest scale in [1, maxScale] for which need(s)
// fits inside a w x h terminal, or 1 when the size is unknown or too small.
func fitScale(w, h int, need func(s int) (width, height int)) int {
	if w <= 0 || h <= 0 {
		return 1
	}
	for s := maxScale; s > 1; s-- {
		nw, nh := need(s)
		if nw <= w && nh <= h {
			return s
		}
	}
	return 1
}

// fitSize is fitScale for callers that can fall back to the compact cell
// size when even scale 1 does not fit.
func fitSize(w, h int, need func(c cellSize) (width, height int)) cellSize {
	if w <= 0 || h <= 0 {
		return scaled(1)
	}
	for s := maxScale; s >= 1; s-- {
		nw, nh := need(scaled(s))
		if nw <= w && nh <= h {
			return scaled(s)
		}
	}
	if nw, nh := need(compact); nw <= w && nh <= h {
		return compact
	}
	return scaled(1) // nothing fits; scale 1 overflows least awkwardly
}
