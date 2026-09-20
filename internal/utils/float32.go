package utils

func SliceToFloat32(s []float64) []float32 {
	slice := make([]float32, len(s))
	for i, num := range s {
		slice[i] = float32(num)
	}

	return slice
}
