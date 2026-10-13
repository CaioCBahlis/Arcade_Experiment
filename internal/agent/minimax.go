package agent

import (
	"math"
	"sort"

	"tetris_agents/internal/game"
)

// Minimax plays as if an adversary were choosing the pieces. After the known
// current and next pieces, the adversary picks whichever of the seven pieces
// is worst for us, and we place that one:
//
//	max over our placements of the current piece
//	  max over our placements of the known next piece
//	    min over the adversary's choice of piece
//	      max over our placements of that piece, then evaluate the board
//
// Alpha-beta pruning skips branches that cannot change the decision, and
// below the root only the most promising placements are searched (beam).
//
// Unlike Expectimax it never gambles on a lucky piece: it prefers the board
// that is best against the worst case, so it plays safe and rarely dies, at
// the cost of ignoring setups that only pay off with the common pieces.
type minimax struct {
	beam int
}

// NewMinimax returns the worst-case lookahead agent.
func NewMinimax() Agent {
	m := &minimax{beam: 6}
	return &planner{choose: m.choose}
}

// adversaryOrder tries the traditionally nastiest pieces first, so the min
// node finds its bound early and alpha-beta cuts more.
var adversaryOrder = [...]game.Kind{game.S, game.Z, game.O, game.I, game.T, game.L, game.J}

// place is the max node: we place kind on b. rest holds further known
// pieces; adv counts adversary-chosen pieces still to come.
func (m *minimax) place(b *game.Board, kind game.Kind, rest []game.Kind, adv int, alpha, beta float64) float64 {
	cs := candidates(b, kind)
	if len(cs) == 0 {
		return deathValue
	}
	if len(rest) == 0 && adv == 0 {
		best := cs[0].static
		for _, c := range cs[1:] {
			best = max(best, c.static)
		}
		return best
	}
	sort.Slice(cs, func(i, j int) bool { return cs[i].static > cs[j].static })
	if len(cs) > m.beam {
		cs = cs[:m.beam]
	}
	best := math.Inf(-1)
	for i := range cs {
		r := lineReward(cs[i].lines)
		v := r + m.after(&cs[i].board, rest, adv, alpha-r, beta-r)
		best = max(best, v)
		alpha = max(alpha, best)
		if alpha >= beta {
			break
		}
	}
	return best
}

// after values board b once a placement is done: continue with the known
// pieces, then hand the choice of piece to the adversary.
func (m *minimax) after(b *game.Board, rest []game.Kind, adv int, alpha, beta float64) float64 {
	if len(rest) > 0 {
		return m.place(b, rest[0], rest[1:], adv, alpha, beta)
	}
	if adv == 0 {
		return boardValue(b)
	}
	worst := math.Inf(1)
	for _, k := range adversaryOrder {
		v := m.place(b, k, nil, adv-1, alpha, beta)
		worst = min(worst, v)
		beta = min(beta, worst)
		if beta <= alpha {
			break
		}
	}
	return worst
}

func (m *minimax) choose(g *game.Game) target {
	cs := candidates(&g.Board, g.Current.Kind)
	if len(cs) == 0 {
		return target{g.Current.Rot, g.Current.Col}
	}
	rest := []game.Kind{g.Next}
	best, bestV := cs[0].t, math.Inf(-1)
	for i := range cs {
		r := lineReward(cs[i].lines)
		v := r + m.after(&cs[i].board, rest, 1, bestV-r, math.Inf(1))
		if v > bestV {
			best, bestV = cs[i].t, v
		}
	}
	return best
}
