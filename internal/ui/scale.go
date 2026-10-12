package ui

import "tetris_agents/internal/game"

// maxScale keeps boards from becoming absurdly large on huge terminals.
const maxScale = 6

// boardChars is the width and height, in characters, of a framed board at
// scale s. Cells are 2s wide and s tall, so the board is square on screen.
func boardChars(s int) int { return game.Width*2*s + 2 }

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
