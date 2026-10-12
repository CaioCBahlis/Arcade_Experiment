package game

// Kind identifies one of the seven tetrominoes.
type Kind int

const (
	I Kind = iota
	O
	T
	S
	Z
	J
	L
	numKinds
)

// shapes holds each tetromino's cells as (row, col) offsets inside its
// bounding box, plus the box size used for rotation.
var shapes = [numKinds]struct {
	size  int
	cells [4][2]int
}{
	I: {4, [4][2]int{{1, 0}, {1, 1}, {1, 2}, {1, 3}}},
	O: {2, [4][2]int{{0, 0}, {0, 1}, {1, 0}, {1, 1}}},
	T: {3, [4][2]int{{0, 1}, {1, 0}, {1, 1}, {1, 2}}},
	S: {3, [4][2]int{{0, 1}, {0, 2}, {1, 0}, {1, 1}}},
	Z: {3, [4][2]int{{0, 0}, {0, 1}, {1, 1}, {1, 2}}},
	J: {3, [4][2]int{{0, 0}, {1, 0}, {1, 1}, {1, 2}}},
	L: {3, [4][2]int{{0, 2}, {1, 0}, {1, 1}, {1, 2}}},
}

// Piece is a tetromino placed on the board. Row and Col locate the top-left
// corner of its bounding box and may be negative while a piece is spawning.
type Piece struct {
	Kind Kind
	Rot  int // number of clockwise quarter turns, 0-3
	Row  int
	Col  int
}

// Offsets returns the piece's cells relative to its bounding box after
// applying its rotation.
func (p Piece) Offsets() [4][2]int {
	s := shapes[p.Kind]
	cells := s.cells
	for i := 0; i < p.Rot%4; i++ {
		for j, c := range cells {
			cells[j] = [2]int{c[1], s.size - 1 - c[0]}
		}
	}
	return cells
}

// Cells returns the absolute (row, col) board positions the piece occupies.
func (p Piece) Cells() [4][2]int {
	cells := p.Offsets()
	for i := range cells {
		cells[i][0] += p.Row
		cells[i][1] += p.Col
	}
	return cells
}
