package mathlib

import "math"

type HammingWindow struct {
	window []float64
}

func NewHammingWindow(size int, alpha, beta float64) *HammingWindow {
	if size <= 0 {
		return nil
	}
	if size == 1 {
		return &HammingWindow{window: []float64{1.0}}
	}

	w := make([]float64, size)
	denom := float64(size - 1)

	for n := 0; n < size; n++ {
		w[n] = alpha - beta*math.Cos(2.0*math.Pi*float64(n)/denom)
	}

	return &HammingWindow{window: w}
}

func (h *HammingWindow) Windowing(samples []float64) {
	if h == nil {
		return
	}

	for i := 0; i < len(samples) && i < len(h.window); i++ {
		samples[i] *= h.window[i]
	}
}
