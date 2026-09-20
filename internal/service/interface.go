package service

import (
	"context"

	"github.com/lorsanstand/voice-auth/internal/models"
)

type VectorStorage interface {
	Add(ctx context.Context, vector models.VectorDBCreate) error
	Get(ctx context.Context, vector []float32) (models.VectorDB, error)
	GetAll(ctx context.Context) ([]models.VectorDB, error)
	Delete(ctx context.Context, id string) error
}
