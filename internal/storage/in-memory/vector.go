package in_memory

import (
	"context"
	"errors"
	"uuid"

	"github.com/lorsanstand/voice-auth/internal/models"
	"github.com/philippgille/chromem-go"
)

type VectorStorage struct {
	db            *chromem.Collection
	embeddingSize int
}

var ErrEmbeddingSizeUnknown = errors.New("embedding size is unknown")

func NewVectorStorage(db *chromem.DB) (*VectorStorage, error) {
	col, err := db.GetOrCreateCollection("voice", nil, nil)
	return &VectorStorage{db: col}, err
}

func (v *VectorStorage) Add(ctx context.Context, content models.VectorDBCreate) error {
	if len(content.Vector) == 0 {
		return errors.New("embedding is empty")
	}

	if v.embeddingSize == 0 {
		v.embeddingSize = len(content.Vector)
	}
	if len(content.Vector) != v.embeddingSize {
		return errors.New("embedding size does not match stored vectors")
	}

	return v.db.AddDocument(ctx, chromem.Document{
		ID:        uuid.New().String(),
		Content:   content.Content,
		Embedding: content.Vector,
		Metadata:  content.Metadata,
	})
}

func (v *VectorStorage) Get(ctx context.Context, vector []float32) (models.VectorDB, error) {
	result, err := v.db.QueryEmbedding(ctx, vector, 1, nil, nil)
	if err != nil {
		return models.VectorDB{}, err
	}

	if len(result) == 0 {
		return models.VectorDB{}, errors.New("vector not found")
	}

	res := result[0]

	return models.VectorDB{
		ID:         res.ID,
		Vector:     res.Embedding,
		Content:    res.Content,
		Metadata:   res.Metadata,
		Similarity: res.Similarity,
	}, nil
}

func (v *VectorStorage) GetAll(ctx context.Context) ([]models.VectorDB, error) {
	count := v.db.Count()
	if count == 0 {
		return []models.VectorDB{}, nil
	}
	if v.embeddingSize == 0 {
		return nil, ErrEmbeddingSizeUnknown
	}

	probe := make([]float32, v.embeddingSize)
	probe[0] = 1
	result, err := v.db.QueryEmbedding(ctx, probe, count, nil, nil)
	if err != nil {
		return nil, err
	}

	content := make([]models.VectorDB, 0, len(result))
	for _, res := range result {
		content = append(content, models.VectorDB{
			ID:       res.ID,
			Vector:   res.Embedding,
			Content:  res.Content,
			Metadata: res.Metadata,
		})
	}

	return content, nil
}

func (v *VectorStorage) Delete(ctx context.Context, id string) error {
	return v.db.Delete(ctx, nil, nil, id)
}
