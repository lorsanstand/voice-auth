package audio

import (
	"errors"
	"fmt"
	"math"
)

const TargetSampleRate = 16000

// ProcessWAV reads a PCM WAV file and returns its decoded samples without
// framing, windowing, or FFT processing. Multi-channel input is mixed down to
// mono and resampled to 16 kHz.
func ProcessWAV(path string) (WAV, error) {
	input, err := ReadWAVFile(path)
	if err != nil {
		return WAV{}, err
	}

	samples, err := ToMono(input.Samples, input.Channels)
	if err != nil {
		return WAV{}, fmt.Errorf("convert WAV to mono: %w", err)
	}

	input.Samples = ResampleLinear(samples, input.SampleRate, TargetSampleRate)
	input.SampleRate = TargetSampleRate
	input.Channels = 1
	return input, nil
}

// ToMono averages interleaved channel samples into one normalized signal.
func ToMono(interleaved []float64, channels int) ([]float64, error) {
	if channels <= 0 {
		return nil, errors.New("channel count must be positive")
	}
	if len(interleaved)%channels != 0 {
		return nil, fmt.Errorf("sample count %d is not divisible by channel count %d", len(interleaved), channels)
	}

	if channels == 1 {
		return interleaved, nil
	}

	mono := make([]float64, len(interleaved)/channels)
	for frame := range mono {
		var sum float64
		for channel := 0; channel < channels; channel++ {
			sum += interleaved[frame*channels+channel]
		}
		mono[frame] = sum / float64(channels)
	}

	return mono, nil
}

// ResampleLinear changes the sample rate using linear interpolation.
func ResampleLinear(samples []float64, sourceRate, targetRate int) []float64 {
	if len(samples) == 0 || sourceRate <= 0 || targetRate <= 0 {
		return nil
	}
	if sourceRate == targetRate {
		return samples
	}

	outputLength := int(math.Round(float64(len(samples)) * float64(targetRate) / float64(sourceRate)))
	if outputLength < 1 {
		outputLength = 1
	}

	resampled := make([]float64, outputLength)
	ratio := float64(sourceRate) / float64(targetRate)
	for i := range resampled {
		position := float64(i) * ratio
		left := int(math.Floor(position))
		if left >= len(samples)-1 {
			resampled[i] = samples[len(samples)-1]
			continue
		}

		fraction := position - float64(left)
		resampled[i] = samples[left] + fraction*(samples[left+1]-samples[left])
	}

	return resampled
}
