package nn

import (
	"math"
	"math/rand"
	"testing"
)

func TestGradientsMatchNumerical(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	n := New([]int{5, 7, 4, 1}, rng)
	s := n.NewScratch()
	x := []float64{0.3, 0, -0.7, 1.2, 0.5}
	target := 0.9
	loss := func() float64 {
		d := n.Forward(x, s) - target
		return 0.5 * d * d
	}

	g := n.NewGrads()
	y := n.Forward(x, s)
	n.Backward(s, y-target, g)

	const eps = 1e-6
	for l := range n.W {
		for i := range n.W[l] {
			old := n.W[l][i]
			n.W[l][i] = old + eps
			up := loss()
			n.W[l][i] = old - eps
			down := loss()
			n.W[l][i] = old
			num := (up - down) / (2 * eps)
			if math.Abs(num-g.W[l][i]) > 1e-4 {
				t.Fatalf("W[%d][%d]: numerical %g, analytic %g", l, i, num, g.W[l][i])
			}
		}
		for i := range n.B[l] {
			old := n.B[l][i]
			n.B[l][i] = old + eps
			up := loss()
			n.B[l][i] = old - eps
			down := loss()
			n.B[l][i] = old
			num := (up - down) / (2 * eps)
			if math.Abs(num-g.B[l][i]) > 1e-4 {
				t.Fatalf("B[%d][%d]: numerical %g, analytic %g", l, i, num, g.B[l][i])
			}
		}
	}
}

func TestLearnsNonlinearFunction(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	n := New([]int{2, 16, 16, 1}, rng)
	opt := NewAdam(n, 0.01)
	s := n.NewScratch()
	g := n.NewGrads()
	f := func(a, b float64) float64 { return math.Abs(a - b) } // not linear

	for step := 0; step < 3000; step++ {
		for k := 0; k < 16; k++ {
			x := []float64{rng.Float64(), rng.Float64()}
			y := n.Forward(x, s)
			n.Backward(s, y-f(x[0], x[1]), g)
		}
		opt.Step(n, g, 1.0/16)
	}
	var err float64
	for k := 0; k < 200; k++ {
		x := []float64{rng.Float64(), rng.Float64()}
		err += math.Abs(n.Forward(x, s) - f(x[0], x[1]))
	}
	if avg := err / 200; avg > 0.05 {
		t.Fatalf("mean abs error %.3f, want < 0.05", avg)
	}
}

func TestMarshalRoundTrip(t *testing.T) {
	n := New([]int{3, 4, 1}, rand.New(rand.NewSource(3)))
	data, err := n.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	m, err := Unmarshal(data)
	if err != nil {
		t.Fatal(err)
	}
	x := []float64{1, 2, 3}
	if a, b := n.Forward(x, n.NewScratch()), m.Forward(x, m.NewScratch()); a != b {
		t.Fatalf("round trip changed output: %v vs %v", a, b)
	}
}
