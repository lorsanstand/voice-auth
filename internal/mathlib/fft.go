package mathlib

import (
	"math"
	"sync"
)

type FFTPlan struct {
	size       int
	bitReverse []int
	twiddles   [][]complex128
}

var fftPlans sync.Map

func NewFFTPlan(size int) *FFTPlan {
	if size <= 0 || size&(size-1) != 0 {
		return nil
	}

	plan := &FFTPlan{
		size:       size,
		bitReverse: make([]int, size),
		twiddles:   make([][]complex128, 0),
	}

	for i := 1; i < size; i++ {
		plan.bitReverse[i] = (plan.bitReverse[i>>1] >> 1) | ((i & 1) * (size >> 1))
	}

	for stageSize := 2; stageSize <= size; stageSize <<= 1 {
		halfStage := stageSize / 2
		stageTwiddles := make([]complex128, halfStage)
		angle := -2.0 * math.Pi / float64(stageSize)
		stageTwiddle := complex(math.Cos(angle), math.Sin(angle))
		twiddle := complex(1, 0)

		for i := range stageTwiddles {
			stageTwiddles[i] = twiddle
			twiddle *= stageTwiddle
		}

		plan.twiddles = append(plan.twiddles, stageTwiddles)
	}

	return plan
}

func (p *FFTPlan) Transform(samples []float64) []complex128 {
	if p == nil || len(samples) != p.size {
		return FFT(samples)
	}

	spectrum := make([]complex128, p.size)
	for i, sample := range samples {
		spectrum[p.bitReverse[i]] = complex(sample, 0)
	}

	for stage, stageSize := 0, 2; stageSize <= p.size; stage, stageSize = stage+1, stageSize<<1 {
		halfStage := stageSize / 2
		stageTwiddles := p.twiddles[stage]

		for start := 0; start < p.size; start += stageSize {
			for offset, twiddle := range stageTwiddles {
				even := spectrum[start+offset]
				odd := spectrum[start+offset+halfStage] * twiddle

				spectrum[start+offset] = even + odd
				spectrum[start+offset+halfStage] = even - odd
			}
		}
	}

	return spectrum
}

func FFT(samples []float64) []complex128 {
	n := len(samples)
	if n == 0 {
		return nil
	}

	if n == 1 {
		return []complex128{complex(samples[0], 0)}
	}
	if n&(n-1) != 0 {
		spectrum := make([]complex128, n)
		for i, sample := range samples {
			spectrum[i] = complex(sample, 0)
		}
		return dft(spectrum)
	}

	if cached, ok := fftPlans.Load(n); ok {
		return cached.(*FFTPlan).Transform(samples)
	}

	plan := NewFFTPlan(n)
	actual, loaded := fftPlans.LoadOrStore(n, plan)
	if loaded {
		plan = actual.(*FFTPlan)
	}

	return plan.Transform(samples)
}

func dft(x []complex128) []complex128 {
	n := len(x)
	res := make([]complex128, n)

	for k := 0; k < n; k++ {
		for sample, value := range x {
			angle := -2.0 * math.Pi * float64(k*sample) / float64(n)
			res[k] += value * complex(math.Cos(angle), math.Sin(angle))
		}
	}

	return res
}
