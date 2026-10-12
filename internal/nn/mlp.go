// Package nn is a minimal fully connected network with ReLU hidden layers, a
// single linear output, backpropagation and the Adam optimizer. It has no
// dependencies and is just big enough to learn a Tetris value function.
package nn

import (
	"encoding/json"
	"math"
	"math/rand"
)

// Net is an MLP mapping sizes[0] inputs to one output. W[l] is stored
// input-major (W[l][i*out+j]) so sparse inputs can skip whole rows.
type Net struct {
	Sizes []int       `json:"sizes"`
	W     [][]float64 `json:"w"`
	B     [][]float64 `json:"b"`
}

// New returns a network with He-initialized weights.
func New(sizes []int, rng *rand.Rand) *Net {
	n := &Net{Sizes: append([]int(nil), sizes...)}
	for l := 0; l+1 < len(sizes); l++ {
		in, out := sizes[l], sizes[l+1]
		w := make([]float64, in*out)
		std := math.Sqrt(2 / float64(in))
		for i := range w {
			w[i] = rng.NormFloat64() * std
		}
		n.W = append(n.W, w)
		n.B = append(n.B, make([]float64, out))
	}
	return n
}

// Clone returns an independent copy of n.
func (n *Net) Clone() *Net {
	c := &Net{Sizes: append([]int(nil), n.Sizes...)}
	for l := range n.W {
		c.W = append(c.W, append([]float64(nil), n.W[l]...))
		c.B = append(c.B, append([]float64(nil), n.B[l]...))
	}
	return c
}

// CopyFrom overwrites n's parameters with src's (same architecture).
func (n *Net) CopyFrom(src *Net) {
	for l := range n.W {
		copy(n.W[l], src.W[l])
		copy(n.B[l], src.B[l])
	}
}

// Marshal serializes the network as JSON.
func (n *Net) Marshal() ([]byte, error) { return json.Marshal(n) }

// Unmarshal parses a network written by Marshal.
func Unmarshal(data []byte) (*Net, error) {
	var n Net
	if err := json.Unmarshal(data, &n); err != nil {
		return nil, err
	}
	return &n, nil
}

// Scratch holds the activations of one forward pass, needed by Backward.
type Scratch struct {
	a [][]float64 // a[0] is the input; a[l] the output of layer l-1
}

// NewScratch allocates activation storage for n.
func (n *Net) NewScratch() *Scratch {
	s := &Scratch{}
	for _, size := range n.Sizes {
		s.a = append(s.a, make([]float64, size))
	}
	return s
}

// Forward evaluates the network on x, recording activations in s.
func (n *Net) Forward(x []float64, s *Scratch) float64 {
	copy(s.a[0], x)
	last := len(n.W) - 1
	for l := range n.W {
		in, out := n.Sizes[l], n.Sizes[l+1]
		prev, cur := s.a[l], s.a[l+1]
		copy(cur, n.B[l])
		w := n.W[l]
		for i := 0; i < in; i++ {
			v := prev[i]
			if v == 0 {
				continue
			}
			row := w[i*out : (i+1)*out]
			for j, wj := range row {
				cur[j] += v * wj
			}
		}
		if l != last {
			for j, v := range cur {
				if v < 0 {
					cur[j] = 0
				}
			}
		}
	}
	return s.a[len(n.W)][0]
}

// Grads accumulates parameter gradients across a batch.
type Grads struct {
	W, B [][]float64
}

// NewGrads allocates zeroed gradients for n.
func (n *Net) NewGrads() *Grads {
	g := &Grads{}
	for l := range n.W {
		g.W = append(g.W, make([]float64, len(n.W[l])))
		g.B = append(g.B, make([]float64, len(n.B[l])))
	}
	return g
}

// Backward adds to g the gradient of the loss for the pass recorded in s,
// where dOut is the loss derivative with respect to the network output.
func (n *Net) Backward(s *Scratch, dOut float64, g *Grads) {
	delta := []float64{dOut}
	for l := len(n.W) - 1; l >= 0; l-- {
		in, out := n.Sizes[l], n.Sizes[l+1]
		prev := s.a[l]
		w := n.W[l]
		var next []float64
		if l > 0 {
			next = make([]float64, in)
		}
		for j, d := range delta {
			g.B[l][j] += d
		}
		for i := 0; i < in; i++ {
			v := prev[i]
			row := w[i*out : (i+1)*out]
			if v != 0 {
				gw := g.W[l][i*out : (i+1)*out]
				for j, d := range delta {
					gw[j] += v * d
				}
			}
			if next != nil && v > 0 { // ReLU gate: inactive units pass nothing
				sum := 0.0
				for j, d := range delta {
					sum += row[j] * d
				}
				next[i] = sum
			}
		}
		delta = next
	}
}

// Adam is the Adam optimizer's state for one network.
type Adam struct {
	LR, Beta1, Beta2, Eps float64
	t                     int
	mW, vW, mB, vB        [][]float64
}

// NewAdam returns an optimizer for n with the usual defaults.
func NewAdam(n *Net, lr float64) *Adam {
	a := &Adam{LR: lr, Beta1: 0.9, Beta2: 0.999, Eps: 1e-8}
	for l := range n.W {
		a.mW = append(a.mW, make([]float64, len(n.W[l])))
		a.vW = append(a.vW, make([]float64, len(n.W[l])))
		a.mB = append(a.mB, make([]float64, len(n.B[l])))
		a.vB = append(a.vB, make([]float64, len(n.B[l])))
	}
	return a
}

// Step applies g (scaled by scale, e.g. 1/batch) to n and zeroes g.
func (a *Adam) Step(n *Net, g *Grads, scale float64) {
	a.t++
	c1 := 1 - math.Pow(a.Beta1, float64(a.t))
	c2 := 1 - math.Pow(a.Beta2, float64(a.t))
	upd := func(p, grad, m, v []float64) {
		for i := range p {
			gi := grad[i] * scale
			grad[i] = 0
			m[i] = a.Beta1*m[i] + (1-a.Beta1)*gi
			v[i] = a.Beta2*v[i] + (1-a.Beta2)*gi*gi
			p[i] -= a.LR * (m[i] / c1) / (math.Sqrt(v[i]/c2) + a.Eps)
		}
	}
	for l := range n.W {
		upd(n.W[l], g.W[l], a.mW[l], a.vW[l])
		upd(n.B[l], g.B[l], a.mB[l], a.vB[l])
	}
}
