package product

import (
	"context"
	"time"

	"go-firestore-gpt/internal/repository/filter"
	"go-firestore-gpt/internal/repository/ops"
	productRepo "go-firestore-gpt/internal/repository/product"
)

type Factory interface {
	OnProductReviewSentimentAnalysis() ProductPublisher
	OnProductVideoAnalysis() ProductPublisher
}

type factory struct {
	repo productRepo.IRepository
}

func ProductPublisherFactory(productRepo productRepo.IRepository) Factory {
	return &factory{
		repo: productRepo,
	}
}

func (f *factory) OnProductVideoAnalysis() ProductPublisher {
	return new(func(ctx context.Context) <-chan productRepo.ProductEvent {
		return f.repo.NotifyOnAdded(ctx,
			[]filter.Where{{Path: productRepo.RelatedVideosAnalized, Op: ops.Equal, Value: false}})
	})
}

func (f *factory) OnProductReviewSentimentAnalysis() ProductPublisher {
	return new(func(ctx context.Context) <-chan productRepo.ProductEvent {
		return f.repo.NotifyOnAdded(ctx,
			[]filter.Where{{Path: productRepo.SentimentAnalizedFieldPath, Op: ops.Equal, Value: false}})
	})
}

func (f *factory) OnProductXXX() ProductPublisher {
	return new(func(ctx context.Context) <-chan productRepo.ProductEvent {
		return f.repo.NotifyOnAdded(ctx,
			[]filter.Where{{Path: productRepo.UpdatedAtFieldPath, Op: ops.Greater, Value: time.Now()}})
	})
}
