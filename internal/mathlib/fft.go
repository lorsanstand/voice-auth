package mathlib

import (
	"math"
)

func FFT(samples []float64) []complex128 {
	n := len(samples)
	if n == 0 {
		return nil
	}

	spectrum := make([]complex128, n)
	for i, sample := range samples {
		spectrum[i] = complex(sample, 0)
	}

	if n == 1 {
		return spectrum
	}
	if n&(n-1) != 0 {
		return dft(spectrum)
	}

	for i, j := 1, 0; i < n; i++ {
		bit := n >> 1
		for j&bit != 0 {
			j ^= bit
			bit >>= 1
		}
		j ^= bit

		if i < j {
			spectrum[i], spectrum[j] = spectrum[j], spectrum[i]
		}
	}

	for stageSize := 2; stageSize <= n; stageSize <<= 1 {
		halfStage := stageSize / 2
		angle := -2.0 * math.Pi / float64(stageSize)
		stageTwiddle := complex(math.Cos(angle), math.Sin(angle))

		for start := 0; start < n; start += stageSize {
			twiddle := complex(1, 0)
			for offset := 0; offset < halfStage; offset++ {
				even := spectrum[start+offset]
				odd := spectrum[start+offset+halfStage] * twiddle

				spectrum[start+offset] = even + odd
				spectrum[start+offset+halfStage] = even - odd
				twiddle *= stageTwiddle
			}
		}
	}

	return spectrum
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
