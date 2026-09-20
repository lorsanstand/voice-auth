package service

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"sync"

	"github.com/lorsanstand/voice-auth/internal/mathlib"
	"github.com/lorsanstand/voice-auth/internal/models"
	"github.com/lorsanstand/voice-auth/internal/utils"
)

const voiceFrameEnergyRatio = 1e-4
const deltaWindow = 2

type AudioBiometry struct {
	melFilter *mathlib.MelFilterBank
	hw        *mathlib.HammingWindow
	mfcc      *mathlib.MFCCExtractor
	fft       *mathlib.FFTPlan
	store     VectorStorage
}

func NewAudioBiometry(storage VectorStorage) *AudioBiometry {
	sampleRate := 16000
	fftSize := 512
	numFilters := 40

	return &AudioBiometry{
		melFilter: mathlib.NewMelFilterBank(numFilters, fftSize, sampleRate, 0, float64(sampleRate)/2),
		hw:        mathlib.NewHammingWindow(fftSize, 0.54, 0.46),
		mfcc:      mathlib.NewMFCCExtractor(numFilters, 13),
		fft:       mathlib.NewFFTPlan(fftSize),
		store:     storage,
	}
}

func (a *AudioBiometry) RegisterVoice(ctx context.Context, samples [][]float64, name string) error {
	if len(samples) < 15 {
		return ErrSmallSamples
	}

	v := utils.SliceToFloat32(a.ToVector(samples))

	vectorDB := models.VectorDBCreate{
		Vector:  v,
		Content: name,
	}

	if err := a.store.Add(ctx, vectorDB); err != nil {
		return fmt.Errorf("error saving data to DB: %w", err)
	}
	return nil
}

func (a *AudioBiometry) GetVoice(ctx context.Context, samples [][]float64) (models.VectorDB, error) {
	if len(samples) < 15 {
		return models.VectorDB{}, ErrSmallSamples
	}

	v := utils.SliceToFloat32(a.ToVector(samples))

	content, err := a.store.Get(ctx, v)
	if err != nil {
		return models.VectorDB{}, fmt.Errorf("error getting data to DB: %w", err)
	}

	return content, nil
}

func (a *AudioBiometry) GetAllVoice(ctx context.Context) ([]models.VectorDB, error) {
	content, err := a.store.GetAll(ctx)
	if err != nil {
		return []models.VectorDB{}, fmt.Errorf("error getting data to DB: %w", err)
	}

	return content, nil
}

func (a *AudioBiometry) DeleteVoice(ctx context.Context, id string) error {
	err := a.store.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("error deleting data to DB: %w", err)
	}

	return nil
}

func (a *AudioBiometry) ToVector(samples [][]float64) []float64 {
	if len(samples) == 0 || len(samples[0]) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	mfcc := make([][]float64, len(samples))
	workers := runtime.GOMAXPROCS(0)
	if workers > len(samples) {
		workers = len(samples)
	}
	jobs := make(chan int)

	wg.Add(workers)
	for worker := 0; worker < workers; worker++ {
		go func() {
			defer wg.Done()
			windowed := make([]float64, len(samples[0]))
			for i := range jobs {
				if len(samples[i]) != len(windowed) {
					mfcc[i] = nil
					continue
				}
				copy(windowed, samples[i])
				mfcc[i] = a.processMFCC(windowed)
			}
		}()
	}

	for i := range samples {
		jobs <- i
	}
	close(jobs)

	wg.Wait()

	delta := mathlib.DeltaMatrix(mfcc, deltaWindow)
	if len(delta) != len(samples) {
		return nil
	}
	voiced := voicedFrameMask(samples)
	features := make([][]float64, 0, len(samples))

	for i := range samples {
		if !voiced[i] || len(mfcc[i]) != 13 || len(delta[i]) != 13 {
			continue
		}

		feature := make([]float64, 0, 24)
		feature = append(feature, mfcc[i][1:]...)
		feature = append(feature, delta[i][1:]...)
		features = append(features, feature)
	}

	return mathlib.NormalizedMeanStdMatrix(features, 0)
}

func (a *AudioBiometry) ProcessMFCC(sample []float64) []float64 {
	windowed := make([]float64, len(sample))
	copy(windowed, sample)
	return a.processMFCC(windowed)
}

func (a *AudioBiometry) processMFCC(sample []float64) []float64 {
	a.hw.Windowing(sample)
	energies := a.melFilter.ComputeMelEnergies(a.fft.Transform(sample))
	vector := a.mfcc.Compute(energies)
	a.mfcc.ApplyLifter(vector, 22)

	return vector
}

func voicedFrameMask(samples [][]float64) []bool {
	if len(samples) == 0 || len(samples[0]) == 0 {
		return nil
	}

	energies := make([]float64, len(samples))
	var maxEnergy float64
	for i, sample := range samples {
		var sum float64
		for _, value := range sample {
			sum += value * value
		}
		energy := sum / float64(len(sample))
		energies[i] = energy
		maxEnergy = math.Max(maxEnergy, energy)
	}

	if maxEnergy == 0 {
		return make([]bool, len(samples))
	}

	threshold := maxEnergy * voiceFrameEnergyRatio
	voiced := make([]bool, len(samples))
	for i := range samples {
		if energies[i] >= threshold {
			voiced[i] = true
		}
	}

	return voiced
}
