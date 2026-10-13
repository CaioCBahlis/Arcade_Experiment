// Package agent provides bots that play Tetris through the game package.
package agent

import "tetris_agents/internal/game"

// Action is a single input an agent can issue.
type Action int

const (
	Left Action = iota
	Right
	Rotate
	Drop
)

// Agent decides one input at a time for the game's current piece.
type Agent interface {
	Act(g *game.Game) Action
}

// Info describes an agent for menus and creates fresh instances of it.
type Info struct {
	Name string
	Desc string
	New  func() Agent
}

// All lists the available agents in menu order.
func All() []Info {
	return []Info{
		{"Heuristic", "Scores every placement by height, holes and bumpiness", NewHeuristic},
		{"Expectimax", "Looks ahead over the next pieces and averages the unknown ones", NewExpectimax},
		{"Cross-Entropy", "Evaluation weights tuned by the cross-entropy method", NewCEM},
		{"Minimax", "Assumes an adversary deals the worst possible piece", NewMinimax},
		{"Beam Search", "Keeps the best boards over several sampled futures", NewBeam},
		{"MCTS", "Monte Carlo tree search over sampled futures with greedy rollouts", NewMCTS},
		{"Deep Learning", "Neural network over the raw board, trained by self-play (DQN)", NewDeep},
		{"RL (TD learning)", "Value function learned by self-play at startup", NewRL},
		{"Random", "Picks a random placement for every piece", NewRandom},
	}
}

// Apply performs a on g and reports whether the game accepted it. A
// rejected action leaves g unchanged, except Drop which always succeeds.
func Apply(g *game.Game, a Action) bool {
	switch a {
	case Left:
		return g.MoveLeft()
	case Right:
		return g.MoveRight()
	case Rotate:
		return g.Rotate()
	default:
		g.HardDrop()
		return true
	}
}

// target is where a planner wants the current piece to end up.
type target struct{ rot, col int }

// planner turns "choose a placement" strategies into step-by-step inputs.
// It plans once per spawned piece, so strategies may be random.
type planner struct {
	choose  func(g *game.Game) target
	spawned int
	t       target
}

func (p *planner) Act(g *game.Game) Action {
	if p.spawned != g.Spawned {
		p.spawned = g.Spawned
		p.t = p.choose(g)
	}
	cur := g.Current
	switch {
	case cur.Rot != p.t.rot:
		return Rotate
	case cur.Col < p.t.col:
		return Right
	case cur.Col > p.t.col:
		return Left
	}
	return Drop
}

// placements lists every distinct rotation/column the current piece can be
// dropped into from its spawn row, with the piece as it would land.
func placements(g *game.Game) []struct {
	t      target
	landed game.Piece
} {
	var out []struct {
		t      target
		landed game.Piece
	}
	for rot := 0; rot < 4; rot++ {
		for col := -2; col < game.Width; col++ {
			p := g.Current
			p.Rot, p.Col = rot, col
			if landed, ok := g.Board.Landing(p); ok {
				out = append(out, struct {
					t      target
					landed game.Piece
				}{target{rot, col}, landed})
			}
		}
	}
	return out
}
