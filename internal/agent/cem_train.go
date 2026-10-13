package agent

import (
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"sort"
	"sync"

	"tetris_agents/internal/game"
)

// CEMOptions configures TrainCEM.
type CEMOptions struct {
	Population  int // weight vectors sampled per generation
	Generations int
	EliteFrac   float64 // fraction kept to refit the distribution
	Games       int     // games per candidate per generation
	PieceCap    int     // pieces per game; survival beyond this is not rewarded
	Seed        int64
	Log         func(string)
}

// DefaultCEM returns settings that train in a minute or two on a laptop.
func DefaultCEM() CEMOptions {
	return CEMOptions{Population: 60, Generations: 25, EliteFrac: 0.2, Games: 3, PieceCap: 1000, Seed: 1}
}

// cemFitness is the mean game score over games seeded from seeds. Score is
// used rather than lines cleared because lines saturate: any competent weights
// clear the maximum possible in a capped game, leaving nothing to rank. Every
// candidate in a generation plays the same seeds, so they are compared on
// identical piece sequences.
func cemFitness(w [nCem]float64, seeds []int64, cap int) float64 {
	total := 0
	for _, s := range seeds {
		g := game.New(s)
		pl := &planner{choose: cemPolicy(w)}
		for g.Spawned <= cap && !g.Over {
			if !Apply(g, pl.Act(g)) {
				g.HardDrop()
			}
		}
		total += g.Score
	}
	return float64(total) / float64(len(seeds))
}

// TrainCEM runs the cross-entropy method over the evaluation weights: sample
// vectors from a diagonal Gaussian, score each by playing games, refit the
// Gaussian to the best fraction, repeat. It returns the final mean.
func TrainCEM(opt CEMOptions) [nCem]float64 {
	logf := func(format string, a ...any) {
		if opt.Log != nil {
			opt.Log(fmt.Sprintf(format, a...))
		}
	}
	rng := rand.New(rand.NewSource(opt.Seed))
	var mean, std [nCem]float64
	for i := range std {
		std[i] = 1
	}
	nElite := max(2, int(float64(opt.Population)*opt.EliteFrac))

	type cand struct {
		w       [nCem]float64
		fitness float64
	}
	for gen := 1; gen <= opt.Generations; gen++ {
		cands := make([]cand, opt.Population)
		for i := range cands {
			for j := range cands[i].w {
				cands[i].w[j] = mean[j] + std[j]*rng.NormFloat64()
			}
		}
		// The first candidate is always the current mean, so the reported
		// best never regresses unnoticed.
		cands[0].w = mean

		seeds := make([]int64, opt.Games)
		for i := range seeds {
			seeds[i] = rng.Int63()
		}
		var wg sync.WaitGroup
		sem := make(chan struct{}, runtime.NumCPU())
		for i := range cands {
			wg.Add(1)
			sem <- struct{}{}
			go func(i int) {
				defer wg.Done()
				cands[i].fitness = cemFitness(cands[i].w, seeds, opt.PieceCap)
				<-sem
			}(i)
		}
		wg.Wait()

		sort.Slice(cands, func(i, j int) bool { return cands[i].fitness > cands[j].fitness })
		var sum float64
		for _, c := range cands {
			sum += c.fitness
		}

		// Refit to the elites; extra noise that shrinks over time keeps the
		// search from collapsing onto one point too early.
		extra := 0.3 * (1 - float64(gen)/float64(opt.Generations+1))
		for j := 0; j < nCem; j++ {
			var m float64
			for _, c := range cands[:nElite] {
				m += c.w[j]
			}
			m /= float64(nElite)
			var v float64
			for _, c := range cands[:nElite] {
				v += (c.w[j] - m) * (c.w[j] - m)
			}
			mean[j] = m
			std[j] = math.Sqrt(v/float64(nElite)) + extra*0.1
		}
		logf("gen %2d  best %.0f  elite-min %.0f  pop-mean %.0f score", gen,
			cands[0].fitness, cands[nElite-1].fitness, sum/float64(len(cands)))
	}
	return mean
}
