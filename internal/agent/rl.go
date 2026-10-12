package agent

import (
	"math/rand"
	"sync"

	"tetris_agents/internal/game"
)

// The RL agent learns a value function V(board) over the board left by a
// placement (an "afterstate"). V is linear in a handful of features and is
// trained by temporal-difference (Q-learning style) updates from self-play:
//
//	w += alpha * (reward + gamma*max V(next afterstate) - V(afterstate)) * features
//
// Reward is 1 per surviving piece plus lines² for clears, so it learns to
// stay alive and to prefer multi-line clears.

const (
	nFeat = 6

	rlAlpha  = 0.005
	rlGamma  = 0.95
	rlEpsHi  = 0.2  // exploration at the start of training
	rlEpsLo  = 0.02 // and at the end
	rlEpochs = 1500 // training episodes
	rlCap    = 150  // max pieces per training episode
	rlSeed   = 1
)

type weights [nFeat]float64

// features maps a post-placement board to the value function's inputs, each
// scaled to roughly [0, 1].
func features(b *game.Board, lines int) [nFeat]float64 {
	st := measure(b)
	return [nFeat]float64{
		1,
		float64(lines) / 4,
		float64(st.agg) / 100,
		float64(st.holes) / 10,
		float64(st.bumpy) / 30,
		float64(st.max) / game.Height,
	}
}

func (w *weights) value(f [nFeat]float64) float64 {
	v := 0.0
	for i, x := range f {
		v += w[i] * x
	}
	return v
}

func (w *weights) update(f [nFeat]float64, delta float64) {
	for i, x := range f {
		w[i] += rlAlpha * delta * x
	}
}

// reward for having just made a placement that cleared lines.
func reward(lines int) float64 { return 1 + float64(lines*lines) }

// policy picks placements from w and, when learning, improves w as it goes.
type policy struct {
	w     *weights
	learn bool
	eps   float64
	rng   *rand.Rand

	prev      [nFeat]float64 // afterstate of the previous decision
	prevLines int
	hasPrev   bool
}

func (p *policy) choose(g *game.Game) target {
	ps := placements(g)
	if len(ps) == 0 {
		return target{g.Current.Rot, g.Current.Col}
	}
	feats := make([][nFeat]float64, len(ps))
	lines := make([]int, len(ps))
	best, bestV := 0, 0.0
	for i, pl := range ps {
		b, n := g.Board.With(pl.landed)
		feats[i], lines[i] = features(&b, n), n
		if v := p.w.value(feats[i]); i == 0 || v > bestV {
			best, bestV = i, v
		}
	}
	if !p.learn {
		return ps[best].t
	}
	if p.hasPrev {
		delta := reward(p.prevLines) + rlGamma*bestV - p.w.value(p.prev)
		p.w.update(p.prev, delta)
	}
	pick := best
	if p.rng.Float64() < p.eps {
		pick = p.rng.Intn(len(ps))
	}
	p.prev, p.prevLines, p.hasPrev = feats[pick], lines[pick], true
	return ps[pick].t
}

// finish applies the terminal update (the game ended, so the future is 0).
func (p *policy) finish() {
	if p.learn && p.hasPrev {
		p.w.update(p.prev, reward(p.prevLines)-p.w.value(p.prev))
	}
	p.hasPrev = false
}

// train runs headless self-play episodes and returns the learned weights.
func train(episodes int, seed int64) weights {
	var w weights
	rng := rand.New(rand.NewSource(seed))
	for ep := 0; ep < episodes; ep++ {
		frac := float64(ep) / float64(episodes)
		p := &policy{w: &w, learn: true, rng: rng, eps: rlEpsHi + (rlEpsLo-rlEpsHi)*frac}
		pl := &planner{choose: p.choose}
		g := game.New(rng.Int63())
		for g.Spawned <= rlCap && !g.Over {
			if !Apply(g, pl.Act(g)) {
				g.HardDrop()
			}
		}
		p.finish()
	}
	return w
}

var (
	trainOnce sync.Once
	trained   weights
)

// WarmRL trains the RL agent now. Calling it in the background at startup
// means the first NewRL doesn't block; NewRL calls it itself if needed.
func WarmRL() {
	trainOnce.Do(func() { trained = train(rlEpochs, rlSeed) })
}

// NewRL returns an agent that plays greedily with the learned value function.
func NewRL() Agent {
	WarmRL()
	w := trained
	p := &policy{w: &w}
	return &planner{choose: p.choose}
}
