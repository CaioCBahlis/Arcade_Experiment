// Command traincem runs the cross-entropy method to tune the CEM agent's
// evaluation weights and prints a line to paste into internal/agent/cem.go.
package main

import (
	"flag"
	"fmt"
	"os"

	"tetris_agents/internal/agent"
)

func main() {
	opt := agent.DefaultCEM()
	flag.IntVar(&opt.Population, "pop", opt.Population, "weight vectors per generation")
	flag.IntVar(&opt.Generations, "gens", opt.Generations, "generations")
	flag.IntVar(&opt.Games, "games", opt.Games, "games per candidate")
	flag.IntVar(&opt.PieceCap, "cap", opt.PieceCap, "pieces per game")
	flag.Int64Var(&opt.Seed, "seed", opt.Seed, "random seed")
	flag.Parse()
	opt.Log = func(s string) { fmt.Fprintln(os.Stderr, s) }

	w := agent.TrainCEM(opt)
	fmt.Printf("var cemWeights = [nCem]float64{")
	for i, v := range w {
		if i > 0 {
			fmt.Print(", ")
		}
		fmt.Printf("%.4f", v)
	}
	fmt.Println("}")
}
