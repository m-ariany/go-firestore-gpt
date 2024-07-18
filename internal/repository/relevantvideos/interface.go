package relevantvideos

import (
	"context"

	"go-firestore-gpt/internal/model"
)

type IRepository interface {
	CreateIfNotExist(ctx context.Context, data model.RelevantVideos) error
	Update(ctx context.Context, data model.RelevantVideos) error
	NotifyOnAdded(ctx context.Context) <-chan RelevantVideosEvent
}
