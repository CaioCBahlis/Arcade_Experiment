package agent

import (
	_ "embed"
	"math/rand"
	"sync"

	"tetris_agents/internal/game"
	"tetris_agents/internal/nn"
)

// The deep agent scores the board each placement would leave with a neural
// network that sees the raw 10x20 grid. It is trained offline by cloning the
// Expectimax agent (see deep_train.go and cmd/traindeep), so it plays like a
// 3-piece lookahead without doing any search, and its weights are embedded.

//go:embed deep_weights.json
var deepWeightsJSON []byte

// Network input layout: 200 cells, 10 column heights, 5 summary features and
// a one-hot of the next piece (the teacher plans with it).
const (
	deepCells  = game.Width * game.Height
	deepInputs = deepCells + game.Width + 5 + game.NumKinds
)

// deepHidden is the hidden layer sizes of the value network.
var deepHidden = []int{128, 64}

func deepSizes() []int {
	return append(append([]int{deepInputs}, deepHidden...), 1)
}

// deepFeatures writes the network input for board b, reached by a placement
// that cleared lines rows with next as the upcoming piece, into out.
func deepFeatures(b *game.Board, lines int, next game.Kind, out []float64) {
	var heights [game.Width]int
	for r := 0; r < game.Height; r++ {
		for c := 0; c < game.Width; c++ {
			v := 0.0
			if b[r][c] != 0 {
				v = 1
				if heights[c] == 0 {
					heights[c] = game.Height - r
				}
			}
			out[r*game.Width+c] = v
		}
	}
	for c, h := range heights {
		out[deepCells+c] = float64(h) / game.Height
	}
	st := measure(b)
	i := deepCells + game.Width
	out[i+0] = float64(lines) / 4
	out[i+1] = float64(st.agg) / 100
	out[i+2] = float64(st.holes) / 10
	out[i+3] = float64(st.bumpy) / 30
	out[i+4] = float64(st.max) / game.Height
	for k := 0; k < game.NumKinds; k++ {
		out[i+5+k] = 0
	}
	out[i+5+int(next)] = 1
}

// deepPolicy picks the placement the network values most.
type deepPolicy struct {
	net *nn.Net
	s   *nn.Scratch
	x   []float64
}

func newDeepPolicy(net *nn.Net) *deepPolicy {
	return &deepPolicy{net: net, s: net.NewScratch(), x: make([]float64, deepInputs)}
}

// value scores the afterstate of landing piece p on g's board.
func (d *deepPolicy) value(g *game.Game, p game.Piece) float64 {
	b, lines := g.Board.With(p)
	deepFeatures(&b, lines, g.Next, d.x)
	return d.net.Forward(d.x, d.s)
}

func (d *deepPolicy) choose(g *game.Game) target {
	ps := placements(g)
	if len(ps) == 0 {
		return target{g.Current.Rot, g.Current.Col}
	}
	best, bestV := 0, 0.0
	for i, pl := range ps {
		if v := d.value(g, pl.landed); i == 0 || v > bestV {
			best, bestV = i, v
		}
	}
	return ps[best].t
}

var (
	deepOnce sync.Once
	deepNet  *nn.Net
)

// loadDeepNet parses the embedded weights, falling back to an untrained
// network when none have been generated or they no longer match the
// architecture.
func loadDeepNet() *nn.Net {
	deepOnce.Do(func() {
		if n, err := nn.Unmarshal(deepWeightsJSON); err == nil && len(n.Sizes) == len(deepSizes()) && n.Sizes[0] == deepInputs {
			deepNet = n
			return
		}
		deepNet = nn.New(deepSizes(), rand.New(rand.NewSource(1)))
	})
	return deepNet
}

// NewDeep returns the neural network agent.
func NewDeep() Agent {
	return &planner{choose: newDeepPolicy(loadDeepNet()).choose}
}
