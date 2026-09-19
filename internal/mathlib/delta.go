package mathlib

// DeltaMatrix calculates first-order temporal derivatives for every column.
// At the signal boundaries, the nearest available frame is reused.
func DeltaMatrix(matrix [][]float64, window int) [][]float64 {
	if len(matrix) == 0 || len(matrix[0]) == 0 || window <= 0 {
		return nil
	}

	cols := len(matrix[0])
	delta := make([][]float64, len(matrix))
	denominator := 0
	for n := 1; n <= window; n++ {
		denominator += n * n
	}
	denominator *= 2

	for frame := range matrix {
		if len(matrix[frame]) != cols {
			return nil
		}

		delta[frame] = make([]float64, cols)
		for column := 0; column < cols; column++ {
			var value float64
			for n := 1; n <= window; n++ {
				previous := frame - n
				if previous < 0 {
					previous = 0
				}

				next := frame + n
				if next >= len(matrix) {
					next = len(matrix) - 1
				}

				value += float64(n) * (matrix[next][column] - matrix[previous][column])
			}
			delta[frame][column] = value / float64(denominator)
		}
	}

	return delta
}
