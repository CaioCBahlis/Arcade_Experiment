package agent

import (
	"math/rand"
	"testing"

	"tetris_agents/internal/game"
	"tetris_agents/internal/nn"
)

func TestDeepFeaturesLayout(t *testing.T) {
	var b game.Board
	b[game.Height-1][3] = 1
	b[game.Height-2][3] = 1
	x := make([]float64, deepInputs)
	deepFeatures(&b, 2, game.T, x)

	if x[(game.Height-1)*game.Width+3] != 1 || x[0] != 0 {
		t.Error("cell encoding wrong")
	}
	if got := x[deepCells+3]; got != 2.0/game.Height {
		t.Errorf("column 3 height feature = %v, want %v", got, 2.0/game.Height)
	}
	base := deepCells + game.Width
	if x[base] != 0.5 {
		t.Errorf("lines feature = %v, want 0.5", x[base])
	}
	for k := 0; k < game.NumKinds; k++ {
		want := 0.0
		if game.Kind(k) == game.T {
			want = 1
		}
		if x[base+5+k] != want {
			t.Errorf("next one-hot[%d] = %v, want %v", k, x[base+5+k], want)
		}
	}
	// Reusing the buffer must fully overwrite the one-hot.
	deepFeatures(&b, 0, game.I, x)
	if x[base+5+int(game.T)] != 0 || x[base+5+int(game.I)] != 1 {
		t.Error("one-hot not reset between calls")
	}
}

func TestDeepEmbeddedWeightsMatchArchitecture(t *testing.T) {
	n, err := nn.Unmarshal(deepWeightsJSON)
	if err != nil || len(n.Sizes) != len(deepSizes()) || n.Sizes[0] != deepInputs {
		t.Fatalf("embedded deep_weights.json is not a trained network of the current architecture (err=%v); run: go run ./cmd/traindeep", err)
	}
}

func TestDeepTrainingImprovesAgreement(t *testing.T) {
	if testing.Short() {
		t.Skip("trains a network")
	}
	data := generate(300, 2, nil, 0.1, 5)
	if len(data) < 300 {
		t.Fatalf("generated %d samples", len(data))
	}
	net := nn.New(deepSizes(), newRand(1))
	w := newWorker(net)
	agree := func() float64 {
		w.loss, w.correct, w.examples = 0, 0, 0
		for i := range data {
			w.example(net, &data[i], false)
		}
		return float64(w.correct) / float64(w.examples)
	}
	before := agree()

	adam := nn.NewAdam(net, 2e-3)
	for epoch := 0; epoch < 30; epoch++ {
		for i := range data {
			w.example(net, &data[i], true)
			if i%32 == 31 {
				adam.Step(net, w.grads, 1.0/32)
			}
		}
	}
	after := agree()
	t.Logf("teacher agreement on its own training data: %.1f%% -> %.1f%%", 100*before, 100*after)
	if after < before+0.2 {
		t.Fatalf("agreement %.2f -> %.2f; training should clearly improve it", before, after)
	}
}

func TestDeepPlaysWell(t *testing.T) {
	if testing.Short() {
		t.Skip("plays full games")
	}
	deep := avgLines(NewDeep, 4, 200)
	rnd := avgLines(NewRandom, 4, 200)
	t.Logf("avg lines in 200 pieces: deep=%.1f random=%.1f", deep, rnd)
	if deep < 40 || deep <= rnd {
		t.Fatalf("deep agent too weak: %.1f lines (random %.1f)", deep, rnd)
	}
}

func newRand(seed int64) *rand.Rand { return rand.New(rand.NewSource(seed)) }
