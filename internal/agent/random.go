package agent

import (
	"math/rand"
	"time"

	"tetris_agents/internal/game"
)

// NewRandom returns an agent that picks any legal placement at random.
func NewRandom() Agent {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return &planner{choose: func(g *game.Game) target {
		ps := placements(g)
		if len(ps) == 0 {
			return target{g.Current.Rot, g.Current.Col}
		}
		return ps[rng.Intn(len(ps))].t
	}}
}
