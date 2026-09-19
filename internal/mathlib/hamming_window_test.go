package mathlib

import (
	"math"
	"testing"
)

func TestHammingWindowingUsesRealSamples(t *testing.T) {
	window := NewHammingWindow(3, 0.54, 0.46)
	samples := []float64{1, 1, 1}

	window.Windowing(samples)

	want := []float64{0.08, 1, 0.08}
	for i := range want {
		if math.Abs(samples[i]-want[i]) > 1e-12 {
			t.Fatalf("sample[%d] = %v, want %v", i, samples[i], want[i])
		}
	}
}
