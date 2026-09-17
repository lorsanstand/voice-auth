package mathlib

func ExtractFrames(samples []float64, frameSize, hopSize int) [][]float64 {
	if frameSize <= 0 || hopSize <= 0 || len(samples) == 0 {
		return nil
	}

	padLeft := frameSize / 2
	padded := make([]float64, padLeft+len(samples))
	copy(padded[padLeft:], samples)

	if len(padded) < frameSize {
		padded = append(padded, make([]float64, frameSize-len(padded))...)
	}

	remainder := (len(padded) - frameSize) % hopSize
	if remainder != 0 {
		padded = append(padded, make([]float64, hopSize-remainder)...)
	}

	frames := make([][]float64, 0, 1+(len(padded)-frameSize)/hopSize)
	for start := 0; start+frameSize <= len(padded); start += hopSize {
		frame := make([]float64, frameSize)
		copy(frame, padded[start:start+frameSize])
		frames = append(frames, frame)
	}

	return frames
}
