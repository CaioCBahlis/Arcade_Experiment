// Command traindeep trains the deep learning agent by imitating the
// Expectimax agent and writes its weights to internal/agent/deep_weights.json,
// which is embedded into the game binary.
package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"tetris_agents/internal/agent"
)

func main() {
	samples := flag.Int("samples", 40000, "labeled decisions recorded per round")
	rounds := flag.Int("rounds", 3, "data rounds (round 0 is the teacher, later rounds the student)")
	epochs := flag.Int("epochs", 6, "passes over the data per round")
	workers := flag.Int("workers", runtime.NumCPU(), "parallel workers")
	seed := flag.Int64("seed", 1, "random seed")
	out := flag.String("out", "internal/agent/deep_weights.json", "where to write the weights")
	flag.Parse()

	net := agent.TrainDeep(agent.DeepTrainOptions{
		Samples: *samples,
		Rounds:  *rounds,
		Epochs:  *epochs,
		Workers: *workers,
		Seed:    *seed,
		Log:     func(s string) { fmt.Fprintln(os.Stderr, s) },
	})
	data, err := net.Marshal()
	if err != nil {
		fmt.Fprintln(os.Stderr, "marshal:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write:", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "wrote %s (%d bytes)\n", *out, len(data))
}
