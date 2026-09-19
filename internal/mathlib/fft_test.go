package mathlib

import (
	"math"
	"testing"
)

func TestFFTImpulse(t *testing.T) {
	samples := []float64{1, 0, 0, 0}
	spectrum := FFT(samples)

	if len(spectrum) != len(samples) {
		t.Fatalf("got %d bins, want %d", len(spectrum), len(samples))
	}

	for i, bin := range spectrum {
		if math.Abs(real(bin)-1) > 1e-12 || math.Abs(imag(bin)) > 1e-12 {
			t.Fatalf("bin %d = %v, want 1+0i", i, bin)
		}
	}
}

func TestFFTNonPowerOfTwoInput(t *testing.T) {
	spectrum := FFT([]float64{1, 2, 3})

	if len(spectrum) != 3 {
		t.Fatalf("got %d bins, want 3", len(spectrum))
	}
	if math.Abs(real(spectrum[0])-6) > 1e-12 || math.Abs(imag(spectrum[0])) > 1e-12 {
		t.Fatalf("DC bin = %v, want 6+0i", spectrum[0])
	}
}
