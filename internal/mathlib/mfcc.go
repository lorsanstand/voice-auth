package mathlib

import (
	"math"
)

type MFCCExtractor struct {
	NumMelFilters int
	NumCoeffs     int
	DCTCepstral   [][]float64
}

func NewMFCCExtractor(numMelFilters, numCoeffs int) *MFCCExtractor {
	dct := make([][]float64, numCoeffs)
	mFloat := float64(numMelFilters)

	for k := 0; k < numCoeffs; k++ {
		dct[k] = make([]float64, numMelFilters)
		for m := 0; m < numMelFilters; m++ {
			// Формула DCT-II
			angle := (math.Pi * float64(k) * (float64(m) + 0.5)) / mFloat
			dct[k][m] = math.Cos(angle)
		}
	}

	return &MFCCExtractor{
		NumMelFilters: numMelFilters,
		NumCoeffs:     numCoeffs,
		DCTCepstral:   dct,
	}
}

func (m *MFCCExtractor) Compute(logMelEnergies []float64) []float64 {
	mfcc := make([]float64, m.NumCoeffs)

	for k := 0; k < m.NumCoeffs; k++ {
		var sum float64
		for idx := 0; idx < m.NumMelFilters; idx++ {
			sum += logMelEnergies[idx] * m.DCTCepstral[k][idx]
		}
		mfcc[k] = sum
	}

	return mfcc
}

func (m *MFCCExtractor) ApplyLifter(mfcc []float64, lifterCoeff int) {
	if lifterCoeff <= 0 {
		return
	}
	l := float64(lifterCoeff)
	for i := 0; i < len(mfcc); i++ {
		weight := 1.0 + (l/2.0)*math.Sin(math.Pi*float64(i)/l)
		mfcc[i] *= weight
	}
}
