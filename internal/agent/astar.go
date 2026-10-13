package agent

import (
	"container/heap"
	"math/rand"
	"time"

	"tetris_agents/internal/game"
)

// A* searches for the cheapest sequence of placements over a known run of
// pieces.
//
// The game reveals only the current and next piece, so the agent samples a few
// extra pieces to form a sequence of `depth` pieces, then runs A* on it. A
// state is a board after k placements; the goal is any board after all
// `depth` placements. Each placement costs how far its static score (lines
// reward + board value) falls short of the best a single placement could
// ever score, so costs are never negative and the cheapest path is the
// highest-scoring plan:
//
//	g(n) = sum of step costs so far
//	h(n) = remaining pieces * cheapest step cost seen leaving n's parent
//	f(n) = g(n) + h(n)
//
// h assumes the remaining pieces will each be as cheap as the best option
// just seen. That is optimistic, since boards tend to get worse, but it is
// not provably admissible, so the plan is not guaranteed optimal. A cap on
// expansions keeps each decision cheap; if the cap is hit before a goal is
// reached the deepest, cheapest frontier node is used instead.
//
// Each sampled future yields one plan; the first move that wins the most
// futures is played, ties going to the lower total cost.
type astar struct {
	depth     int // pieces planned, including the current one
	scenarios int // sampled futures per decision
	maxExpand int // node expansions allowed per future
	rng       *rand.Rand
}

// stepCeiling bounds the static score of any one placement: boardValue is
// never positive and at most four rows clear at once.
var stepCeiling = lineReward(4)

// NewAStar returns the A* planning agent.
func NewAStar() Agent {
	a := &astar{
		depth:     4,
		scenarios: 5,
		maxExpand: 300,
		rng:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	return &planner{choose: a.choose}
}

// anode is one board in the search, tagged with the first move that led to it.
type anode struct {
	board game.Board
	depth int // placements made
	root  int // index into the root candidates
	g, f  float64
	idx   int
}

type frontier []*anode

func (q frontier) Len() int { return len(q) }
func (q frontier) Less(i, j int) bool {
	if q[i].f != q[j].f {
		return q[i].f < q[j].f
	}
	return q[i].depth > q[j].depth // dive on ties
}
func (q frontier) Swap(i, j int) { q[i], q[j] = q[j], q[i]; q[i].idx, q[j].idx = i, j }
func (q *frontier) Push(x any)   { n := x.(*anode); n.idx = len(*q); *q = append(*q, n) }
func (q *frontier) Pop() any {
	old := *q
	n := old[len(old)-1]
	*q = old[:len(old)-1]
	return n
}

func (a *astar) choose(g *game.Game) target {
	roots := candidates(&g.Board, g.Current.Kind)
	if len(roots) == 0 {
		return target{g.Current.Rot, g.Current.Col}
	}
	votes := make([]int, len(roots))
	cost := make([]float64, len(roots))
	for s := 0; s < a.scenarios; s++ {
		seq := make([]game.Kind, 0, a.depth-1)
		seq = append(seq, g.Next)
		for len(seq) < a.depth-1 {
			seq = append(seq, game.Kind(a.rng.Intn(game.NumKinds)))
		}
		root, c := a.plan(roots, seq)
		votes[root]++
		cost[root] += c
	}
	best := 0
	for i := range roots {
		if votes[i] > votes[best] || (votes[i] == votes[best] && votes[i] > 0 && cost[i] < cost[best]) {
			best = i
		}
	}
	return roots[best].t
}

// plan runs A* over the placements of seq after each root placement and
// returns the winning root and the plan's cost.
func (a *astar) plan(roots []candidate, seq []game.Kind) (int, float64) {
	type key struct {
		d int
		b game.Board
	}
	var open frontier
	bestG := map[key]float64{}
	push := func(n *anode, h float64) {
		n.f = n.g + h
		k := key{n.depth, n.board}
		if old, ok := bestG[k]; ok && old <= n.g {
			return
		}
		bestG[k] = n.g
		heap.Push(&open, n)
	}

	// Roots are the first layer: the cheapest of them sets h for the layer.
	c0 := make([]float64, len(roots))
	cmin := 0.0
	for i, r := range roots {
		c0[i] = stepCeiling - r.static
		if i == 0 || c0[i] < cmin {
			cmin = c0[i]
		}
	}
	rest := float64(len(seq))
	for i, r := range roots {
		push(&anode{board: r.board, depth: 1, root: i, g: c0[i]}, rest*cmin)
	}

	fallback := open[0]
	for expanded := 0; open.Len() > 0 && expanded < a.maxExpand; expanded++ {
		n := heap.Pop(&open).(*anode)
		if g, ok := bestG[key{n.depth, n.board}]; ok && g < n.g {
			continue // a cheaper route to this board was found since
		}
		if n.depth == len(seq)+1 {
			return n.root, n.g
		}
		if n.depth > fallback.depth || (n.depth == fallback.depth && n.f < fallback.f) {
			fallback = n
		}
		cs := candidates(&n.board, seq[n.depth-1])
		if len(cs) == 0 {
			continue // stuck: this line of play dies
		}
		step := make([]float64, len(cs))
		cheapest := 0.0
		for i, c := range cs {
			step[i] = stepCeiling - c.static
			if i == 0 || step[i] < cheapest {
				cheapest = step[i]
			}
		}
		left := float64(len(seq) - n.depth)
		for i, c := range cs {
			push(&anode{board: c.board, depth: n.depth + 1, root: n.root, g: n.g + step[i]}, left*cheapest)
		}
	}
	// Out of budget (or every line dies): fall back to the deepest node
	// seen. Charge dead ends the death value so living roots win.
	if fallback.depth == 1 && open.Len() == 0 {
		return fallback.root, fallback.g - deathValue
	}
	return fallback.root, fallback.g + float64(len(seq)+1-fallback.depth)*cmin
}
