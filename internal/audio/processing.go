package audio

import (
	"errors"
	"fmt"
	"math"

	"github.com/lorsanstand/voice-auth/internal/mathlib"
)

const (
	DefaultSampleRate = 16000
	DefaultFrameSize  = 512
	DefaultHopSize    = 160
	DefaultAlpha      = 0.54
	DefaultBeta       = 0.46
)

// Config controls the voice preprocessing pipeline.
type Config struct {
	TargetSampleRate int
	FrameSize        int
	HopSize          int
	WindowAlpha      float64
	WindowBeta       float64
}

// DefaultConfig returns a configuration for 16 kHz voice processing with
// 32 ms frames and a 10 ms hop.
func DefaultConfig() Config {
	return Config{
		TargetSampleRate: DefaultSampleRate,
		FrameSize:        DefaultFrameSize,
		HopSize:          DefaultHopSize,
		WindowAlpha:      DefaultAlpha,
		WindowBeta:       DefaultBeta,
	}
}

// Result contains the signal at the target sample rate, windowed real-valued
// frames, and the corresponding complex FFT spectra.
type Result struct {
	Samples    []float64
	SampleRate int
	Frames     [][]float64
	Spectra    [][]complex128
}

// ProcessWAV reads a WAV file and runs the complete preprocessing pipeline.
func ProcessWAV(path string, config Config) (Result, error) {
	input, err := ReadWAVFile(path)
	if err != nil {
		return Result{}, err
	}

	if input.Channels < 1 {
		return Result{}, errors.New("WAV contains no channels")
	}

	mono, err := ToMono(input.Samples, input.Channels)
	if err != nil {
		return Result{}, fmt.Errorf("convert WAV to mono: %w", err)
	}

	return Process(mono, input.SampleRate, config)
}

// Process converts a mono signal to the target sample rate, extracts frames,
// applies a Hamming window, and calculates FFT for every frame.
func Process(samples []float64, sampleRate int, config Config) (Result, error) {
	if sampleRate <= 0 {
		return Result{}, errors.New("sample rate must be positive")
	}

	config = withDefaults(config)
	if err := validateConfig(config); err != nil {
		return Result{}, err
	}

	resampled := ResampleLinear(samples, sampleRate, config.TargetSampleRate)
	rawFrames := mathlib.ExtractFrames(resampled, config.FrameSize, config.HopSize)
	window := mathlib.NewHammingWindow(config.FrameSize, config.WindowAlpha, config.WindowBeta)

	frames := make([][]float64, len(rawFrames))
	spectra := make([][]complex128, len(rawFrames))
	for i, rawFrame := range rawFrames {
		frame := append([]float64(nil), rawFrame...)
		window.Windowing(frame)
		frames[i] = frame
		spectra[i] = mathlib.FFT(frame)
	}

	return Result{
		Samples:    resampled,
		SampleRate: config.TargetSampleRate,
		Frames:     frames,
		Spectra:    spectra,
	}, nil
}

// ToMono averages interleaved channels into one normalized signal.
func ToMono(interleaved []float64, channels int) ([]float64, error) {
	if channels <= 0 {
		return nil, errors.New("channel count must be positive")
	}
	if len(interleaved)%channels != 0 {
		return nil, fmt.Errorf("sample count %d is not divisible by channel count %d", len(interleaved), channels)
	}

	if channels == 1 {
		mono := make([]float64, len(interleaved))
		copy(mono, interleaved)
		return mono, nil
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
		resampled := make([]float64, len(samples))
		copy(resampled, samples)
		return resampled
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

func withDefaults(config Config) Config {
	defaults := DefaultConfig()
	if config.TargetSampleRate == 0 {
		config.TargetSampleRate = defaults.TargetSampleRate
	}
	if config.FrameSize == 0 {
		config.FrameSize = defaults.FrameSize
	}
	if config.HopSize == 0 {
		config.HopSize = defaults.HopSize
	}
	if config.WindowAlpha == 0 && config.WindowBeta == 0 {
		config.WindowAlpha = defaults.WindowAlpha
		config.WindowBeta = defaults.WindowBeta
	}
	return config
}

func validateConfig(config Config) error {
	if config.TargetSampleRate <= 0 {
		return errors.New("target sample rate must be positive")
	}
	if config.FrameSize <= 0 {
		return errors.New("frame size must be positive")
	}
	if config.HopSize <= 0 {
		return errors.New("hop size must be positive")
	}
	if config.WindowAlpha < 0 || config.WindowBeta < 0 {
		return errors.New("Hamming window coefficients must not be negative")
	}
	return nil
}
