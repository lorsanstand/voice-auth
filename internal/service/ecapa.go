package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	sherpa "github.com/k2-fsa/sherpa-onnx-go/sherpa_onnx"

	"github.com/lorsanstand/voice-auth/internal/mathlib"
	"github.com/lorsanstand/voice-auth/internal/models"
	"github.com/lorsanstand/voice-auth/internal/utils"
)

const minECAPASamples = 16000

type ECAPABiometry struct {
	extractor *sherpa.SpeakerEmbeddingExtractor
	store     VectorStorage
	mu        sync.Mutex
}

func NewECAPABiometry(store VectorStorage, modelPath string) (*ECAPABiometry, error) {
	if store == nil {
		return nil, errors.New("ECAPA storage is required")
	}
	if modelPath == "" {
		return nil, errors.New("ECAPA model path is required")
	}
	if _, err := os.Stat(modelPath); err != nil {
		return nil, fmt.Errorf("ECAPA model %q is unavailable: %w", modelPath, err)
	}

	config := &sherpa.SpeakerEmbeddingExtractorConfig{
		Model:      modelPath,
		NumThreads: 1,
		Provider:   "cpu",
	}
	extractor := sherpa.NewSpeakerEmbeddingExtractor(config)
	if extractor == nil {
		return nil, fmt.Errorf("failed to load ECAPA model %q", modelPath)
	}
	if setter, ok := store.(interface{ SetEmbeddingSize(int) error }); ok {
		if err := setter.SetEmbeddingSize(extractor.Dim()); err != nil {
			sherpa.DeleteSpeakerEmbeddingExtractor(extractor)
			return nil, fmt.Errorf("configure ECAPA embedding size: %w", err)
		}
	}

	return &ECAPABiometry{
		extractor: extractor,
		store:     store,
	}, nil
}

func (e *ECAPABiometry) Close() {
	if e == nil || e.extractor == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	sherpa.DeleteSpeakerEmbeddingExtractor(e.extractor)
	e.extractor = nil
}

func (e *ECAPABiometry) RegisterVoice(ctx context.Context, samples []float64, name string) error {
	vector, err := e.embedding(samples)
	if err != nil {
		return err
	}

	if err := e.store.Add(ctx, models.VectorDBCreate{
		Vector:  vector,
		Content: name,
	}); err != nil {
		return fmt.Errorf("error saving ECAPA data to DB: %w", err)
	}
	return nil
}

func (e *ECAPABiometry) GetVoice(ctx context.Context, samples []float64) (models.VectorDB, error) {
	vector, err := e.embedding(samples)
	if err != nil {
		return models.VectorDB{}, err
	}

	result, err := e.store.Get(ctx, vector)
	if err != nil {
		return models.VectorDB{}, fmt.Errorf("error getting ECAPA data from DB: %w", err)
	}
	return result, nil
}

func (e *ECAPABiometry) GetAllVoice(ctx context.Context) ([]models.VectorDB, error) {
	content, err := e.store.GetAll(ctx)
	if err != nil {
		return []models.VectorDB{}, fmt.Errorf("error getting ECAPA data from DB: %w", err)
	}
	return content, nil
}

func (e *ECAPABiometry) DeleteVoice(ctx context.Context, id string) error {
	if err := e.store.Delete(ctx, id); err != nil {
		return fmt.Errorf("error deleting ECAPA data from DB: %w", err)
	}
	return nil
}

func (e *ECAPABiometry) Compare(samplesA, samplesB []float64) (float32, error) {
	first, err := e.embedding(samplesA)
	if err != nil {
		return 0, err
	}
	second, err := e.embedding(samplesB)
	if err != nil {
		return 0, err
	}

	first64 := make([]float64, len(first))
	second64 := make([]float64, len(second))
	for i := range first {
		first64[i] = float64(first[i])
		second64[i] = float64(second[i])
	}

	similarity, err := mathlib.CosineSimilarity(first64, second64)
	if err != nil {
		return 0, fmt.Errorf("compare ECAPA vectors: %w", err)
	}
	return float32(similarity), nil
}

func (e *ECAPABiometry) embedding(samples []float64) ([]float32, error) {
	if len(samples) < minECAPASamples {
		return nil, ErrSmallSamples
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	if e.extractor == nil {
		return nil, errors.New("ECAPA extractor is closed")
	}

	stream := e.extractor.CreateStream()
	if stream == nil {
		return nil, errors.New("failed to create ECAPA audio stream")
	}
	defer sherpa.DeleteOnlineStream(stream)

	stream.AcceptWaveform(16000, utils.SliceToFloat32(samples))
	stream.InputFinished()
	if !e.extractor.IsReady(stream) {
		return nil, errors.New("ECAPA extractor did not receive enough usable audio")
	}

	embedding := e.extractor.Compute(stream)
	if len(embedding) != e.extractor.Dim() {
		return nil, errors.New("ECAPA extractor returned an invalid embedding")
	}
	return embedding, nil
}
