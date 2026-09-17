package mathlib

import (
	"math"
	"math/cmplx"
)

func HzToMel(hz float64) float64 {
	return 2595.0 * math.Log10(1.0+hz/700.0)
}

func MelToHz(mel float64) float64 {
	return 700.0 * (math.Pow(10.0, mel/2595.0) - 1.0)
}

type MelFilterBank struct {
	NumFilters int
	FFTSize    int
	SampleRate int
	Weights    [][]float64
}

func NewMelFilterBank(numFilters, fftSize, sampleRate int, minHz, maxHz float64) *MelFilterBank {
	numBins := fftSize/2 + 1
	minMel := HzToMel(minHz)
	maxMel := HzToMel(maxHz)

	melStep := (maxMel - minMel) / float64(numFilters+1)
	binPoints := make([]int, numFilters+2)

	for i := 0; i < len(binPoints); i++ {
		mel := minMel + float64(i)*melStep
		hz := MelToHz(mel)
		bin := int(math.Floor((float64(fftSize) + 1.0) * hz / float64(sampleRate)))
		if bin >= numBins {
			bin = numBins - 1
		}
		binPoints[i] = bin
	}

	weights := make([][]float64, numFilters)
	for m := 0; m < numFilters; m++ {
		weights[m] = make([]float64, numBins)

		left := binPoints[m]
		center := binPoints[m+1]
		right := binPoints[m+2]

		for k := left; k < center; k++ {
			if center > left {
				weights[m][k] = float64(k-left) / float64(center-left)
			}
		}

		for k := center; k < right; k++ {
			if right > center {
				weights[m][k] = float64(right-k) / float64(right-center)
			}
		}
	}

	return &MelFilterBank{
		NumFilters: numFilters,
		FFTSize:    fftSize,
		SampleRate: sampleRate,
		Weights:    weights,
	}
}

func (fb *MelFilterBank) ComputeMelEnergies(spectrum []complex128) []float64 {
	numBins := fb.FFTSize/2 + 1

	powerSpectrum := make([]float64, numBins)
	for k := 0; k < numBins; k++ {
		mag := cmplx.Abs(spectrum[k])
		powerSpectrum[k] = (mag * mag) / float64(fb.FFTSize)
	}

	melEnergies := make([]float64, fb.NumFilters)
	for m := 0; m < fb.NumFilters; m++ {
		var sum float64
		for k := 0; k < numBins; k++ {
			sum += powerSpectrum[k] * fb.Weights[m][k]
		}

		if sum < 1e-10 {
			sum = 1e-10
		}

		melEnergies[m] = math.Log(sum)
	}

	return melEnergies
}
