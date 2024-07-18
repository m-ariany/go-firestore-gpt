package product

import (
	"context"

	"go-firestore-gpt/internal/model"
	"go-firestore-gpt/internal/repository/filter"
)

type IRepository interface {
	GetById(ctx context.Context, id string) (*model.Product, error)
	Create(ctx context.Context, data model.Product) error
	Update(ctx context.Context, id string, data model.Product) error
	NotifyOnAdded(ctx context.Context, where []filter.Where) <-chan ProductEvent
}
