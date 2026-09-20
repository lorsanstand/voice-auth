package mathlib

import (
	"fmt"
	"math"
)

func CosineSimilarity(first, second []float64) (float64, error) {
	if len(first) == 0 || len(first) != len(second) {
		return 0, fmt.Errorf("vectors must have the same non-zero size")
	}

	var dot, firstNorm, secondNorm float64
	for i := range first {
		dot += first[i] * second[i]
		firstNorm += first[i] * first[i]
		secondNorm += second[i] * second[i]
	}
	if firstNorm == 0 || secondNorm == 0 {
		return 0, fmt.Errorf("vectors must not be zero")
	}

	return dot / (math.Sqrt(firstNorm) * math.Sqrt(secondNorm)), nil
}

// NormalizedMeanStdMatrix returns one vector made from the normalized means
// and standard deviations of columns starting at firstColumn.
func NormalizedMeanStdMatrix(matrix [][]float64, firstColumn int) []float64 {
	if len(matrix) == 0 || len(matrix[0]) == 0 || firstColumn < 0 || firstColumn >= len(matrix[0]) {
		return nil
	}

	cols := len(matrix[0]) - firstColumn
	means := make([]float64, cols)

	for _, row := range matrix {
		if len(row) != len(matrix[0]) {
			return nil
		}
		for i := 0; i < cols; i++ {
			means[i] += row[firstColumn+i]
		}
	}

	rows := float64(len(matrix))
	for i := range means {
		means[i] /= rows
	}

	stds := make([]float64, cols)
	for _, row := range matrix {
		for i := 0; i < cols; i++ {
			diff := row[firstColumn+i] - means[i]
			stds[i] += diff * diff
		}
	}

	for i := range stds {
		stds[i] = math.Sqrt(stds[i] / rows)
	}

	normalizeL2(means)
	normalizeL2(stds)

	vector := make([]float64, 0, 2*cols)
	vector = append(vector, means...)
	vector = append(vector, stds...)
	return vector
}

func normalizeL2(vector []float64) {
	var sum float64
	for _, value := range vector {
		sum += value * value
	}

	if sum == 0 {
		return
	}

	norm := math.Sqrt(sum)
	for i := range vector {
		vector[i] /= norm
	}
}
