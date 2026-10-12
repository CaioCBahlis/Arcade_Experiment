package agent

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"tetris_agents/internal/game"
	"tetris_agents/internal/nn"
)

// Training the deep agent is behavior cloning with a ranking loss:
//
//   - The Expectimax agent labels decisions: for a state (board, current and
//     next piece) it picks one placement.
//   - The network scores every candidate placement of that state; a softmax
//     over those scores is trained with cross-entropy to put its mass on the
//     teacher's choice. Only the relative order of sibling candidates
//     matters, which is exactly what picking a move needs.
//   - Round 0 records the teacher playing with some random moves mixed in,
//     for variety. Later rounds record the *student* playing (still with a
//     little noise) while the teacher relabels each state, so the network
//     learns to recover from its own mistakes (DAgger).

// DeepTrainOptions configures TrainDeep.
type DeepTrainOptions struct {
	Samples int // labeled decisions per round
	Rounds  int
	Epochs  int // passes over all data per round
	Workers int
	Seed    int64
	Log     func(string) // progress messages; may be nil
}

const (
	dtBatch     = 64
	dtLRHi      = 2e-3
	dtLRLo      = 2e-4
	dtGameCap   = 400  // pieces per recorded game
	dtNoise0    = 0.15 // random-move rate while recording the teacher
	dtNoiseN    = 0.05 // ... and while recording the student
	dtHoldout   = 0.1
	dtEvalGames = 8
	dtEvalCap   = 400
)

// labeled is one teacher decision: the state and the placement it chose.
type labeled struct {
	cells    [deepCells]uint8 // 1 where the board is filled
	kind     game.Kind
	next     game.Kind
	rot, col int8
}

func (l *labeled) board() game.Board {
	var b game.Board
	for i, v := range l.cells {
		if v != 0 {
			b[i/game.Width][i%game.Width] = 1
		}
	}
	return b
}

// generate records n teacher-labeled decisions using workers goroutines.
// Moves are taken from student when it is non-nil, else from the teacher,
// with a noise fraction replaced by random placements.
func generate(n, workers int, student *nn.Net, noise float64, seed int64) []labeled {
	out := make([]labeled, 0, n)
	var mu sync.Mutex
	var wg sync.WaitGroup
	per := (n + workers - 1) / workers
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(seed + int64(w)*7919))
			teacher := &expectimax{beam: 5}
			var stu *deepPolicy
			if student != nil {
				stu = newDeepPolicy(student)
			}
			local := make([]labeled, 0, per)
			for len(local) < per {
				g := game.New(rng.Int63())
				pl := &planner{choose: func(g *game.Game) target {
					ps := placements(g)
					if len(ps) == 0 {
						return target{g.Current.Rot, g.Current.Col}
					}
					t := teacher.choose(g)
					if len(local) < per {
						var l labeled
						for r := 0; r < game.Height; r++ {
							for c := 0; c < game.Width; c++ {
								if g.Board[r][c] != 0 {
									l.cells[r*game.Width+c] = 1
								}
							}
						}
						l.kind, l.next = g.Current.Kind, g.Next
						l.rot, l.col = int8(t.rot), int8(t.col)
						local = append(local, l)
					}
					switch {
					case rng.Float64() < noise:
						return ps[rng.Intn(len(ps))].t
					case stu != nil:
						return stu.choose(g)
					default:
						return t
					}
				}}
				for g.Spawned <= dtGameCap && !g.Over && len(local) < per {
					if !Apply(g, pl.Act(g)) {
						g.HardDrop()
					}
				}
			}
			mu.Lock()
			out = append(out, local...)
			mu.Unlock()
		}(w)
	}
	wg.Wait()
	return out
}

// worker holds per-goroutine scratch space for batched training.
type worker struct {
	grads    *nn.Grads
	scratch  []*nn.Scratch
	x        [][]float64
	loss     float64
	correct  int
	examples int
}

func newWorker(net *nn.Net) *worker {
	w := &worker{grads: net.NewGrads()}
	for i := 0; i < 48; i++ { // more than the most placements any piece has
		w.scratch = append(w.scratch, net.NewScratch())
		w.x = append(w.x, make([]float64, deepInputs))
	}
	return w
}

// example scores every candidate of l, accumulates the ranking-loss gradient
// when train is set, and records loss and top-1 agreement with the teacher.
func (w *worker) example(net *nn.Net, l *labeled, train bool) {
	b := l.board()
	cs := candidates(&b, l.kind)
	if len(cs) == 0 {
		return
	}
	if len(cs) > len(w.scratch) {
		cs = cs[:len(w.scratch)]
	}
	// Candidates that leave the same board as the teacher's choice (e.g. the
	// four rotations of the O piece) are equally right.
	var want game.Board
	found := false
	for i := range cs {
		if cs[i].t.rot == int(l.rot) && cs[i].t.col == int(l.col) {
			want, found = cs[i].board, true
			break
		}
	}
	if !found {
		return
	}
	v := make([]float64, len(cs))
	pos := make([]bool, len(cs))
	npos := 0
	top := 0
	for i := range cs {
		deepFeatures(&cs[i].board, cs[i].lines, l.next, w.x[i])
		v[i] = net.Forward(w.x[i], w.scratch[i])
		if v[i] > v[top] {
			top = i
		}
		if cs[i].board == want {
			pos[i] = true
			npos++
		}
	}
	// softmax
	m := v[0]
	for _, x := range v {
		m = max(m, x)
	}
	var z float64
	p := make([]float64, len(v))
	for i, x := range v {
		p[i] = fastExp(x - m)
		z += p[i]
	}
	var loss float64
	for i := range p {
		p[i] /= z
		if pos[i] {
			loss -= safeLog(p[i]) / float64(npos)
		}
	}
	w.loss += loss
	w.examples++
	if pos[top] {
		w.correct++
	}
	if !train {
		return
	}
	for i := range cs {
		d := p[i]
		if pos[i] {
			d -= 1 / float64(npos)
		}
		net.Backward(w.scratch[i], d, w.grads)
	}
}

// TrainDeep trains a network to imitate the Expectimax agent and returns the
// version with the best agreement on held-out decisions.
func TrainDeep(opt DeepTrainOptions) *nn.Net {
	logf := func(format string, a ...any) {
		if opt.Log != nil {
			opt.Log(fmt.Sprintf(format, a...))
		}
	}
	rng := rand.New(rand.NewSource(opt.Seed))
	net := nn.New(deepSizes(), rng)
	adam := nn.NewAdam(net, dtLRHi)
	workers := make([]*worker, opt.Workers)
	for i := range workers {
		workers[i] = newWorker(net)
	}
	total := opt.Rounds * opt.Epochs
	epoch := 0

	var train, hold []labeled
	start := time.Now()
	for round := 0; round < opt.Rounds; round++ {
		t0 := time.Now()
		var data []labeled
		if round == 0 {
			data = generate(opt.Samples, opt.Workers, nil, dtNoise0, rng.Int63())
		} else {
			data = generate(opt.Samples, opt.Workers, net, dtNoiseN, rng.Int63())
		}
		nh := int(float64(len(data)) * dtHoldout)
		hold = append(hold, data[:nh]...)
		train = append(train, data[nh:]...)
		logf("round %d: +%d labeled decisions in %v (train %d, held-out %d)",
			round, len(data), time.Since(t0).Round(time.Second), len(train), len(hold))

		for e := 0; e < opt.Epochs; e++ {
			epoch++
			adam.LR = dtLRHi + (dtLRLo-dtLRHi)*float64(epoch-1)/float64(max(total-1, 1))
			rng.Shuffle(len(train), func(i, j int) { train[i], train[j] = train[j], train[i] })
			for _, w := range workers {
				w.loss, w.correct, w.examples = 0, 0, 0
			}
			for lo := 0; lo+dtBatch <= len(train); lo += dtBatch {
				var wg sync.WaitGroup
				chunk := dtBatch / len(workers)
				for wi, w := range workers {
					wg.Add(1)
					go func(wi int, w *worker) {
						defer wg.Done()
						from := lo + wi*chunk
						to := from + chunk
						if wi == len(workers)-1 {
							to = lo + dtBatch
						}
						for i := from; i < to; i++ {
							w.example(net, &train[i], true)
						}
					}(wi, w)
				}
				wg.Wait()
				for _, w := range workers[1:] {
					addGrads(workers[0].grads, w.grads)
				}
				adam.Step(net, workers[0].grads, 1.0/dtBatch)
			}
			var tl float64
			var tc, tn int
			for _, w := range workers {
				tl, tc, tn = tl+w.loss, tc+w.correct, tn+w.examples
			}
			hl, ha := evalHoldout(net, hold, workers)
			logf("round %d epoch %2d  lr %.4f  train loss %.3f agree %.1f%%  held-out loss %.3f agree %.1f%%  [%v]",
				round, e+1, adam.LR, tl/float64(max(tn, 1)), 100*float64(tc)/float64(max(tn, 1)), hl, 100*ha,
				time.Since(start).Round(time.Second))
		}
		lines, dead := playEval(net)
		logf("round %d play: %.1f lines, %d/%d topped out in %d pieces", round, lines, dead, dtEvalGames, dtEvalCap)
	}
	return net
}

func addGrads(dst, src *nn.Grads) {
	for l := range dst.W {
		for i, v := range src.W[l] {
			dst.W[l][i] += v
			src.W[l][i] = 0
		}
		for i, v := range src.B[l] {
			dst.B[l][i] += v
			src.B[l][i] = 0
		}
	}
}

// evalHoldout returns the mean ranking loss and top-1 agreement on hold.
func evalHoldout(net *nn.Net, hold []labeled, workers []*worker) (loss, agree float64) {
	var wg sync.WaitGroup
	for _, w := range workers {
		w.loss, w.correct, w.examples = 0, 0, 0
	}
	for wi, w := range workers {
		wg.Add(1)
		go func(wi int, w *worker) {
			defer wg.Done()
			for i := wi; i < len(hold); i += len(workers) {
				w.example(net, &hold[i], false)
			}
		}(wi, w)
	}
	wg.Wait()
	var n, c int
	for _, w := range workers {
		loss += w.loss
		n += w.examples
		c += w.correct
	}
	if n == 0 {
		return 0, 0
	}
	return loss / float64(n), float64(c) / float64(n)
}

// playEval plays the network greedily on fixed seeds, in parallel, and
// returns mean lines cleared and how many games topped out.
func playEval(net *nn.Net) (lines float64, dead int) {
	var wg sync.WaitGroup
	res := make([][2]int, dtEvalGames)
	for i := 0; i < dtEvalGames; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			g := game.New(int64(9000 + i))
			a := &planner{choose: newDeepPolicy(net).choose}
			for g.Spawned <= dtEvalCap && !g.Over {
				if !Apply(g, a.Act(g)) {
					g.HardDrop()
				}
			}
			res[i][0] = g.Lines
			if g.Over {
				res[i][1] = 1
			}
		}(i)
	}
	wg.Wait()
	total := 0
	for _, r := range res {
		total += r[0]
		dead += r[1]
	}
	return float64(total) / dtEvalGames, dead
}

func fastExp(x float64) float64 { return math.Exp(x) }
func safeLog(x float64) float64 { return math.Log(max(x, 1e-12)) }
