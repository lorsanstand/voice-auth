package audio

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"

	"github.com/go-audio/wav"
)

// WAV contains mono or multi-channel PCM samples normalized to approximately
// [-1, 1].
type WAV struct {
	Samples    []float64
	SampleRate int
	Channels   int
	BitDepth   int
}

// ReadWAVFile opens and decodes a PCM WAV file.
func ReadWAVFile(path string) (WAV, error) {
	file, err := os.Open(path)
	if err != nil {
		return WAV{}, fmt.Errorf("open WAV file: %w", err)
	}
	defer file.Close()

	return ReadWAV(file)
}

// ReadWAV decodes a PCM WAV stream. Samples are returned interleaved when the
// source contains more than one channel.
func ReadWAV(reader io.ReadSeeker) (WAV, error) {
	if reader == nil {
		return WAV{}, errors.New("WAV reader is nil")
	}

	decoder := wav.NewDecoder(reader)
	decoder.ReadInfo()
	if err := decoder.Err(); err != nil {
		return WAV{}, fmt.Errorf("read WAV header: %w", err)
	}
	if decoder.WavAudioFormat != 1 {
		return WAV{}, fmt.Errorf("unsupported WAV format %d: only PCM is supported", decoder.WavAudioFormat)
	}
	if decoder.NumChans == 0 || decoder.SampleRate == 0 || decoder.BitDepth == 0 {
		return WAV{}, errors.New("WAV header contains invalid audio format")
	}

	buffer, err := decoder.FullPCMBuffer()
	if err != nil {
		return WAV{}, fmt.Errorf("decode WAV samples: %w", err)
	}

	samples := make([]float64, len(buffer.Data))
	for i, sample := range buffer.Data {
		samples[i] = normalizePCM(sample, int(decoder.BitDepth))
	}

	return WAV{
		Samples:    samples,
		SampleRate: int(decoder.SampleRate),
		Channels:   int(decoder.NumChans),
		BitDepth:   int(decoder.BitDepth),
	}, nil
}

func normalizePCM(sample, bitDepth int) float64 {
	if bitDepth == 8 {
		return (float64(sample) - 128.0) / 128.0
	}

	scale := math.Ldexp(1, bitDepth-1)
	if scale == 0 {
		return 0
	}

	value := float64(sample) / scale
	return math.Max(-1, math.Min(1, value))
}
