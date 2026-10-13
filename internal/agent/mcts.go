package agent

import (
	"math"
	"math/rand"
	"time"

	"tetris_agents/internal/game"
)

// Monte Carlo Tree Search over placements.
//
// Each iteration walks the tree from the current board, picking placements
// by UCT, adds one new placement to the tree, then finishes the line with a
// short greedy rollout and scores the board it ends on (lines reward earned
// along the way + the board's static value). That score is backed up the
// path. After all iterations the most visited first placement is played.
//
// The game reveals only the current and next piece, so deeper pieces are
// sampled afresh on every iteration. The tree is therefore "open loop": a
// node stands for a sequence of placements, not for one exact board, and
// its statistics average over the futures that were sampled through it.
type mcts struct {
	iters   int     // simulations per decision
	depth   int     // placements the tree itself may grow
	rollout int     // greedy placements played after the tree
	explore float64 // UCT exploration constant, on values scaled to [0,1]
	rng     *rand.Rand
	lo, hi  float64 // range of values seen this decision, for scaling
}

// mctsDeath is the value of a line of play that tops out. It is clamped
// well above deathValue so one death does not flatten every other score
// once values are scaled to [0,1].
const mctsDeath = -100.0

// NewMCTS returns the Monte Carlo Tree Search agent.
func NewMCTS() Agent {
	m := &mcts{
		iters:   500,
		depth:   4,
		rollout: 3,
		explore: 0.7,
		rng:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	return &planner{choose: m.choose}
}

type mctsNode struct {
	kids map[target]*mctsNode
	n    int
	sum  float64
}

func newMCTSNode() *mctsNode { return &mctsNode{kids: map[target]*mctsNode{}} }

func (m *mcts) choose(g *game.Game) target {
	roots := candidates(&g.Board, g.Current.Kind)
	switch len(roots) {
	case 0:
		return target{g.Current.Rot, g.Current.Col}
	case 1:
		return roots[0].t
	}
	root := newMCTSNode()
	m.lo, m.hi = math.Inf(1), math.Inf(-1)
	seq := make([]game.Kind, 1+m.depth+m.rollout)
	for i := 0; i < m.iters; i++ {
		seq[0] = g.Next
		for j := 1; j < len(seq); j++ {
			seq[j] = game.Kind(m.rng.Intn(game.NumKinds))
		}
		m.simulate(root, roots, seq)
	}
	best, bestN := roots[0].t, -1
	for _, c := range roots {
		if k := root.kids[c.t]; k != nil && k.n > bestN {
			best, bestN = c.t, k.n
		}
	}
	return best
}

// simulate runs one select / expand / rollout / backup pass. seq[i] is the
// piece placed at tree level i+1 (level 0 is the current piece, in roots).
func (m *mcts) simulate(root *mctsNode, roots []candidate, seq []game.Kind) {
	path := []*mctsNode{root}
	node, cs := root, roots
	acc := 0.0
	var value float64
	for level := 0; ; {
		if len(cs) == 0 {
			value = mctsDeath
			break
		}
		c, fresh := m.pick(node, cs)
		child := node.kids[c.t]
		if child == nil {
			child = newMCTSNode()
			node.kids[c.t] = child
		}
		path = append(path, child)
		acc += lineReward(c.lines)
		level++
		if fresh || level > m.depth {
			value = m.playout(&c.board, acc, seq[level-1:])
			break
		}
		node, cs = child, candidates(&c.board, seq[level-1])
	}

	m.lo, m.hi = math.Min(m.lo, value), math.Max(m.hi, value)
	for _, n := range path {
		n.n++
		n.sum += value
	}
}

// pick chooses the placement to follow from node among the legal cs: an
// untried one if any (fresh reports that), otherwise the best by UCT.
func (m *mcts) pick(node *mctsNode, cs []candidate) (c candidate, fresh bool) {
	untried := 0
	for i := range cs {
		if node.kids[cs[i].t] == nil {
			untried++
		}
	}
	if untried > 0 {
		k := m.rng.Intn(untried)
		for i := range cs {
			if node.kids[cs[i].t] == nil {
				if k == 0 {
					return cs[i], true
				}
				k--
			}
		}
	}
	span := m.hi - m.lo
	if span < 1e-9 {
		span = 1
	}
	logN := math.Log(float64(node.n) + 1)
	bestScore := math.Inf(-1)
	for i := range cs {
		k := node.kids[cs[i].t]
		mean := (k.sum/float64(k.n) - m.lo) / span
		if s := mean + m.explore*math.Sqrt(logN/float64(k.n)); s > bestScore {
			c, bestScore = cs[i], s
		}
	}
	return c, false
}

// playout plays the next pieces greedily from b and returns the final score.
func (m *mcts) playout(b *game.Board, acc float64, pieces []game.Kind) float64 {
	cur := *b
	for i := 0; i < m.rollout && i < len(pieces); i++ {
		cs := candidates(&cur, pieces[i])
		if len(cs) == 0 {
			return mctsDeath
		}
		best := 0
		for j := range cs {
			if cs[j].static > cs[best].static {
				best = j
			}
		}
		acc += lineReward(cs[best].lines)
		cur = cs[best].board
	}
	return acc + boardValue(&cur)
}
