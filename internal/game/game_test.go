package game

import (
	"testing"
	"time"
)

func TestRotationReturnsAfterFourTurns(t *testing.T) {
	for k := Kind(0); k < numKinds; k++ {
		p := Piece{Kind: k}
		q := p
		q.Rot = 4
		if p.Offsets() != q.Offsets() {
			t.Errorf("kind %d: 4 rotations should equal 0", k)
		}
	}
}

func TestRotateChangesShape(t *testing.T) {
	p := Piece{Kind: I}
	q := Piece{Kind: I, Rot: 1}
	if p.Offsets() == q.Offsets() {
		t.Fatal("rotating I should change its cells")
	}
}

func TestCollisionWallsAndFloor(t *testing.T) {
	var b Board
	if !b.Collides(Piece{Kind: O, Col: -1}) {
		t.Error("left wall should collide")
	}
	if !b.Collides(Piece{Kind: O, Col: Width - 1}) {
		t.Error("right wall should collide")
	}
	if !b.Collides(Piece{Kind: O, Row: Height - 1}) {
		t.Error("floor should collide")
	}
	if b.Collides(Piece{Kind: O, Col: 4}) {
		t.Error("open space should not collide")
	}
}

func TestCollisionLockedCell(t *testing.T) {
	var b Board
	b[5][5] = 1
	if !b.Collides(Piece{Kind: O, Row: 4, Col: 4}) {
		t.Error("locked cell should collide")
	}
}

func TestClearLines(t *testing.T) {
	var b Board
	for c := 0; c < Width; c++ {
		b[Height-1][c] = 1
		b[Height-2][c] = 1
	}
	b[Height-3][2] = 3
	if n := b.clearLines(); n != 2 {
		t.Fatalf("cleared %d, want 2", n)
	}
	if b[Height-1][2] != 3 {
		t.Error("row above should have shifted down")
	}
	for c := 0; c < Width; c++ {
		if c != 2 && b[Height-1][c] != 0 {
			t.Errorf("unexpected cell at col %d", c)
		}
	}
}

func TestClearLinesKeepsPartialRows(t *testing.T) {
	var b Board
	for c := 0; c < Width; c++ {
		b[Height-1][c] = 1
	}
	b[Height-2][0] = 2
	b.clearLines()
	if b[Height-1][0] != 2 || b[Height-1][1] != 0 {
		t.Error("partial row should drop intact")
	}
}

func TestHardDropLocksAndSpawns(t *testing.T) {
	g := New(1)
	first := g.Current.Kind
	next := g.Next
	g.HardDrop()
	if g.Current.Kind != next {
		t.Errorf("spawned %d, want %d", g.Current.Kind, next)
	}
	filled := 0
	for r := range g.Board {
		for c := range g.Board[r] {
			if g.Board[r][c] == int(first)+1 {
				filled++
			}
		}
	}
	if filled != 4 {
		t.Errorf("locked %d cells, want 4", filled)
	}
	if g.Score == 0 {
		t.Error("hard drop should score")
	}
}

func TestLineClearScoring(t *testing.T) {
	g := New(1)
	for c := 4; c < Width; c++ {
		g.Board[Height-1][c] = 1
	}
	// Horizontal I occupies row 1 of its box, so Row = Height-2 puts it on the floor row.
	g.Current = Piece{Kind: I, Row: Height - 2, Col: 0}
	g.HardDrop()
	if g.Lines != 1 {
		t.Fatalf("lines = %d, want 1", g.Lines)
	}
	if g.Score != 40 {
		t.Errorf("score = %d, want 40", g.Score)
	}
}

func TestGameOver(t *testing.T) {
	g := New(1)
	for r := 0; r < 4; r++ {
		for c := 0; c < Width; c++ {
			g.Board[r][c] = 1
		}
	}
	g.spawn()
	if !g.Over {
		t.Fatal("blocked spawn should end the game")
	}
	if g.MoveLeft() {
		t.Error("no moves after game over")
	}
}

func TestBagContainsEachKindOnce(t *testing.T) {
	g := New(42)
	seen := map[Kind]int{}
	// New already drew 2 from the first bag; draw the remaining 5.
	seen[g.Current.Kind]++
	seen[g.Next]++
	for i := 0; i < 5; i++ {
		seen[g.draw()]++
	}
	for k := Kind(0); k < numKinds; k++ {
		if seen[k] != 1 {
			t.Errorf("kind %d seen %d times in first bag", k, seen[k])
		}
	}
}

func TestLevelAndInterval(t *testing.T) {
	g := New(1)
	if g.Level() != 1 {
		t.Fatal("start at level 1")
	}
	g.Lines = 25
	if g.Level() != 3 {
		t.Errorf("level = %d, want 3", g.Level())
	}
	g.Lines = 1000
	if g.Interval() != 100*time.Millisecond {
		t.Errorf("interval floor not applied: %v", g.Interval())
	}
}

func TestGhostLandsOnFloor(t *testing.T) {
	g := New(1)
	gh := g.Ghost()
	if g.Board.Collides(gh) {
		t.Fatal("ghost should not collide")
	}
	gh.Row++
	if !g.Board.Collides(gh) {
		t.Fatal("ghost should rest on something")
	}
}
