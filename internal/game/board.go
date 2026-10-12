package game

const (
	Width  = 10
	Height = 20
)

// Board is the playfield. A zero cell is empty; otherwise it holds Kind+1 of
// the piece that locked there.
type Board [Height][Width]int

// Collides reports whether p overlaps a wall, the floor, or a locked cell.
// Cells above the top of the board are allowed.
func (b *Board) Collides(p Piece) bool {
	for _, c := range p.Cells() {
		r, col := c[0], c[1]
		if col < 0 || col >= Width || r >= Height {
			return true
		}
		if r >= 0 && b[r][col] != 0 {
			return true
		}
	}
	return false
}

// lock writes p into the board.
func (b *Board) lock(p Piece) {
	for _, c := range p.Cells() {
		if c[0] >= 0 {
			b[c[0]][c[1]] = int(p.Kind) + 1
		}
	}
}

// clearLines removes full rows, shifting the rows above down, and returns
// how many were removed.
func (b *Board) clearLines() int {
	cleared := 0
	for r := Height - 1; r >= 0; r-- {
		full := true
		for c := 0; c < Width; c++ {
			if b[r][c] == 0 {
				full = false
				break
			}
		}
		if !full {
			if cleared > 0 {
				b[r+cleared] = b[r]
			}
			continue
		}
		cleared++
	}
	for r := 0; r < cleared; r++ {
		b[r] = [Width]int{}
	}
	return cleared
}
