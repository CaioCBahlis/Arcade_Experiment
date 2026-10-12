package game

import (
	"math/rand"
	"time"
)

var lineScores = [5]int{0, 40, 100, 300, 1200}

// Game holds the complete state of a Tetris session.
type Game struct {
	Board   Board
	Current Piece
	Next    Kind
	Score   int
	Lines   int
	Over    bool
	Spawned int // pieces spawned so far; lets agents tell pieces apart

	rng *rand.Rand
	bag []Kind
}

// New starts a game using seed for the piece sequence.
func New(seed int64) *Game {
	g := &Game{rng: rand.New(rand.NewSource(seed))}
	g.Next = g.draw()
	g.spawn()
	return g
}

// Level starts at 1 and rises every 10 lines.
func (g *Game) Level() int { return g.Lines/10 + 1 }

// Interval is the gravity delay for the current level.
func (g *Game) Interval() time.Duration {
	d := 800*time.Millisecond - time.Duration(g.Level()-1)*70*time.Millisecond
	if d < 100*time.Millisecond {
		d = 100 * time.Millisecond
	}
	return d
}

// draw takes the next kind from a shuffled bag of all seven.
func (g *Game) draw() Kind {
	if len(g.bag) == 0 {
		for k := Kind(0); k < numKinds; k++ {
			g.bag = append(g.bag, k)
		}
		g.rng.Shuffle(len(g.bag), func(i, j int) { g.bag[i], g.bag[j] = g.bag[j], g.bag[i] })
	}
	k := g.bag[0]
	g.bag = g.bag[1:]
	return k
}

func (g *Game) spawn() {
	g.Spawned++
	g.Current = Spawn(g.Next)
	g.Next = g.draw()
	if g.Board.Collides(g.Current) {
		g.Over = true
	}
}

// try moves the current piece to p if it fits.
func (g *Game) try(p Piece) bool {
	if g.Over || g.Board.Collides(p) {
		return false
	}
	g.Current = p
	return true
}

func (g *Game) MoveLeft() bool {
	p := g.Current
	p.Col--
	return g.try(p)
}

func (g *Game) MoveRight() bool {
	p := g.Current
	p.Col++
	return g.try(p)
}

// Rotate turns the piece clockwise, trying a one-cell shift either way if the
// plain rotation is blocked.
func (g *Game) Rotate() bool {
	p := g.Current
	p.Rot = (p.Rot + 1) % 4
	for _, dc := range [...]int{0, -1, 1} {
		q := p
		q.Col += dc
		if g.try(q) {
			return true
		}
	}
	return false
}

// SoftDrop moves the piece down one row, locking it if it cannot move.
func (g *Game) SoftDrop() {
	if g.Over {
		return
	}
	if g.step() {
		g.Score++
	}
}

// HardDrop drops the piece as far as it goes and locks it.
func (g *Game) HardDrop() {
	if g.Over {
		return
	}
	for g.step() {
		g.Score += 2
	}
}

// Tick applies one gravity step.
func (g *Game) Tick() {
	if g.Over {
		return
	}
	g.step()
}

// step moves the piece down one row and reports whether it moved. When it
// cannot move, the piece locks and the next one spawns.
func (g *Game) step() bool {
	p := g.Current
	p.Row++
	if g.try(p) {
		return true
	}
	g.Board.lock(g.Current)
	n := g.Board.clearLines()
	g.Score += lineScores[n] * g.Level()
	g.Lines += n
	g.spawn()
	return false
}

// Ghost returns the piece at the row where a hard drop would land it.
func (g *Game) Ghost() Piece {
	p := g.Current
	for {
		q := p
		q.Row++
		if g.Board.Collides(q) {
			return p
		}
		p = q
	}
}
