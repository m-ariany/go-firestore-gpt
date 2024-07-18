package reviewsentiments

import (
	"context"

	"go-firestore-gpt/internal/model"
)

type IRepository interface {
	Create(ctx context.Context, data model.ReviewSentiments) error
	GetById(ctx context.Context, id string) (*model.ReviewSentiments, error)
}
