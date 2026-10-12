package agent

import (
	"sort"

	"tetris_agents/internal/game"
)

// Expectimax looks ahead through the pieces the game has revealed (the
// current one and the next one) and then averages over all seven possible
// pieces for one more ply. Leaves are scored with the same board features as
// the Heuristic agent, but the lines reward is summed along the path, so the
// agent can choose to set up a big clear rather than take a small one now.
//
//	V(board, piece) = max over placements [ reward(lines) + V'(board') ]
//	V'              = V of the next known piece, else the mean over all 7
//	                  kinds of V, else the static board value
type expectimax struct {
	beam int // children expanded below the root, ranked by static score
}

// deathValue is the value of a board where the next piece cannot spawn. It
// is large against typical scores so a chance of dying is averaged in as a
// serious risk.
const deathValue = -500.0

// NewExpectimax returns the lookahead agent.
func NewExpectimax() Agent {
	e := &expectimax{beam: 5}
	return &planner{choose: e.choose}
}

// lineReward is what clearing n rows at once is worth to the search.
func lineReward(n int) float64 { return wLines * float64(n) }

func boardValue(b *game.Board) float64 {
	st := measure(b)
	return wHeight*float64(st.agg) + wHoles*float64(st.holes) + wBumpy*float64(st.bumpy)
}

type candidate struct {
	t      target
	board  game.Board
	lines  int
	static float64
}

// candidates lists every placement of kind on b, or nil if it cannot spawn.
func candidates(b *game.Board, kind game.Kind) []candidate {
	if b.Collides(game.Spawn(kind)) {
		return nil
	}
	var out []candidate
	for rot := 0; rot < 4; rot++ {
		for col := -2; col < game.Width; col++ {
			p := game.Spawn(kind)
			p.Rot, p.Col = rot, col
			landed, ok := b.Landing(p)
			if !ok {
				continue
			}
			nb, lines := b.With(landed)
			out = append(out, candidate{
				t: target{rot, col}, board: nb, lines: lines,
				static: lineReward(lines) + boardValue(&nb),
			})
		}
	}
	return out
}

// value is the expected final score of playing kind on b, then the pieces in
// rest, then one averaged chance ply when chance is set.
func (e *expectimax) value(b *game.Board, kind game.Kind, rest []game.Kind, chance bool) float64 {
	cs := candidates(b, kind)
	if len(cs) == 0 {
		return deathValue
	}
	if len(rest) == 0 && !chance {
		best := cs[0].static
		for _, c := range cs[1:] {
			best = max(best, c.static)
		}
		return best
	}
	// Only the most promising placements are searched further.
	sort.Slice(cs, func(i, j int) bool { return cs[i].static > cs[j].static })
	if len(cs) > e.beam {
		cs = cs[:e.beam]
	}
	best := -1e18
	for i := range cs {
		best = max(best, lineReward(cs[i].lines)+e.future(&cs[i].board, rest, chance))
	}
	return best
}

// future values board b after a placement, given the known pieces left.
func (e *expectimax) future(b *game.Board, rest []game.Kind, chance bool) float64 {
	switch {
	case len(rest) > 0:
		return e.value(b, rest[0], rest[1:], chance)
	case chance:
		sum := 0.0
		for k := 0; k < game.NumKinds; k++ {
			sum += e.value(b, game.Kind(k), nil, false)
		}
		return sum / float64(game.NumKinds)
	default:
		return boardValue(b)
	}
}

func (e *expectimax) choose(g *game.Game) target {
	cs := candidates(&g.Board, g.Current.Kind)
	if len(cs) == 0 {
		return target{g.Current.Rot, g.Current.Col}
	}
	rest := []game.Kind{g.Next}
	best, bestV := cs[0].t, -1e18
	for i := range cs {
		v := lineReward(cs[i].lines) + e.future(&cs[i].board, rest, true)
		if v > bestV {
			best, bestV = cs[i].t, v
		}
	}
	return best
}
