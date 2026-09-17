package mathlib

import (
	"math"
	"math/cmplx"
)

func FFT(x []complex128) []complex128 {
	n := len(x)

	if n <= 1 {
		return x
	}

	if n&(n-1) != 0 {
		return dft(x)
	}

	half := n / 2
	even := make([]complex128, half)
	odd := make([]complex128, half)

	for i := 0; i < half; i++ {
		even[i] = x[i*2]
		odd[i] = x[i*2+1]
	}

	fftEven := FFT(even)
	fftOdd := FFT(odd)

	res := make([]complex128, n)

	for k := 0; k < half; k++ {
		angle := -2.0 * math.Pi * float64(k) / float64(n)
		twiddle := cmplx.Rect(1, angle) * fftOdd[k]

		res[k] = fftEven[k] + twiddle
		res[k+half] = fftEven[k] - twiddle
	}

	return res
}

func dft(x []complex128) []complex128 {
	n := len(x)
	res := make([]complex128, n)

	for k := 0; k < n; k++ {
		for sample, value := range x {
			angle := -2.0 * math.Pi * float64(k*sample) / float64(n)
			res[k] += value * cmplx.Rect(1, angle)
		}
	}

	return res
}
