package agent

import (
	"math/rand"
	"sort"
	"time"

	"tetris_agents/internal/game"
)

// Beam search plans a sequence of placements several pieces deep.
//
// The game reveals only the current and next piece, so the pieces after that
// are sampled: the agent draws several random futures, and within each one
// runs a beam search. At every depth it expands each kept board with every
// placement of the next piece and keeps only the `width` best-scoring boards
// (score = lines reward earned so far + the board's static value). Every kept
// board remembers which first move it came from. A first move's value is its
// best surviving score, averaged over the sampled futures.
//
// Compared with Expectimax it looks further ahead for the same cost, but it
// prunes by a static score and sees only a sample of the unknown pieces.
type beam struct {
	width     int // boards kept per depth
	scenarios int // sampled futures per decision
	extra     int // sampled pieces beyond the known next piece
	rng       *rand.Rand
}

// NewBeam returns the beam search agent.
func NewBeam() Agent {
	b := &beam{
		width:     12,
		scenarios: 6,
		extra:     3,
		rng:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	return &planner{choose: b.choose}
}

// node is one board in the beam, tagged with the first move that led to it.
type node struct {
	board game.Board
	root  int
	acc   float64 // lines reward earned along the way
	score float64 // acc plus the board's static value
}

func (b *beam) choose(g *game.Game) target {
	roots := candidates(&g.Board, g.Current.Kind)
	if len(roots) == 0 {
		return target{g.Current.Rot, g.Current.Col}
	}
	sum := make([]float64, len(roots))
	for s := 0; s < b.scenarios; s++ {
		seq := make([]game.Kind, 0, 1+b.extra)
		seq = append(seq, g.Next)
		for i := 0; i < b.extra; i++ {
			seq = append(seq, game.Kind(b.rng.Intn(game.NumKinds)))
		}
		for i, v := range b.run(roots, seq) {
			sum[i] += v
		}
	}
	best := 0
	for i := range sum {
		if sum[i] > sum[best] {
			best = i
		}
	}
	return roots[best].t
}

// run beam-searches the pieces in seq from every root placement and returns
// each root's value in this sampled future.
func (b *beam) run(roots []candidate, seq []game.Kind) []float64 {
	cur := make([]node, len(roots))
	for i, r := range roots {
		acc := lineReward(r.lines)
		cur[i] = node{board: r.board, root: i, acc: acc, score: acc + boardValue(&r.board)}
	}
	dead := make([]bool, len(roots))

	for _, kind := range seq {
		var next []node
		hadChild := make([]bool, len(roots))
		inBeam := make([]bool, len(roots))
		for i := range cur {
			n := &cur[i]
			inBeam[n.root] = true
			for _, c := range candidates(&n.board, kind) {
				acc := n.acc + lineReward(c.lines)
				next = append(next, node{c.board, n.root, acc, acc + boardValue(&c.board)})
				hadChild[n.root] = true
			}
		}
		// A root whose every surviving board is stuck has lost this future.
		for r := range roots {
			if inBeam[r] && !hadChild[r] {
				dead[r] = true
			}
		}
		if len(next) == 0 {
			break
		}
		sort.Slice(next, func(i, j int) bool { return next[i].score > next[j].score })
		if len(next) > b.width {
			next = next[:b.width]
		}
		cur = next
	}

	// Pruned roots are no better than the weakest board still in the beam.
	cutoff := cur[len(cur)-1].score
	values := make([]float64, len(roots))
	for r := range values {
		values[r] = cutoff
	}
	seen := make([]bool, len(roots))
	for _, n := range cur {
		if !seen[n.root] {
			values[n.root], seen[n.root] = n.score, true // cur is sorted best first
		}
	}
	for r, d := range dead {
		if d && !seen[r] {
			values[r] = deathValue
		}
	}
	return values
}
