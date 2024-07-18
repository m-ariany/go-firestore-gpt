package relevantvideos

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go-firestore-gpt/internal/database"
	"go-firestore-gpt/internal/model"
	"go-firestore-gpt/internal/repository/filter"
	"go-firestore-gpt/internal/repository/helper"
	"go-firestore-gpt/internal/repository/ops"
	"go-firestore-gpt/internal/utils"

	"cloud.google.com/go/firestore"
	"github.com/rs/zerolog/log"
)

type RelevantVideosRepository struct {
	db database.Client
}

var _ IRepository = RelevantVideosRepository{}

func New(db database.Client) RelevantVideosRepository {
	return RelevantVideosRepository{
		db: db,
	}
}

func (r RelevantVideosRepository) CreateIfNotExist(ctx context.Context, data model.RelevantVideos) error {

	rv, err := r.getById(ctx, *data.ProductId)
	if rv != nil {
		return nil
	}

	if err != nil {
		return err
	}

	return r.create(ctx, data)
}

func (r RelevantVideosRepository) Update(ctx context.Context, data model.RelevantVideos) error {
	if data.ProductId == nil {
		return fmt.Errorf("failed to update, RelevantVideo.ProductId is nil")
	}

	docRef := r.db.Collection(relevantVideosNode).Doc(*data.ProductId)
	updates := []firestore.Update{}

	err := r.createVideos(ctx, docRef.ID, data.Videos)
	if err != nil {
		return err
	}

	data.UpdatedAt = time.Now().UTC()
	if data.Ready != nil {
		updates = append(updates, firestore.Update{
			Path:  ReadyFieldPath,
			Value: *data.Ready,
		})
	}

	_, err = r.db.UpdateDoc(ctx, docRef, updates)
	if err != nil {
		return fmt.Errorf("update relevant videos: %w, id: %s", err, *data.ProductId)
	}
	return nil
}

func (r RelevantVideosRepository) NotifyOnAdded(ctx context.Context) <-chan RelevantVideosEvent {
	query := r.db.Collection(relevantVideosNode).Query
	where := []filter.Where{{Path: ReadyFieldPath, Op: ops.Equal, Value: false}}
	return r.notifyOnChanges(ctx, query, where, firestore.DocumentAdded)
}

func (r RelevantVideosRepository) create(ctx context.Context, data model.RelevantVideos) error {

	data.CreatedAt = time.Now().UTC()
	data.UpdatedAt = data.CreatedAt
	docRef := r.db.Collection(relevantVideosNode).Doc(*data.ProductId)
	_, err := r.db.SetDoc(ctx, docRef, data)

	if err != nil {
		err = fmt.Errorf("create relevant videos: %w, id: %s", err, docRef.ID)
		return err
	}

	err = r.createVideos(ctx, docRef.ID, data.Videos)

	return err
}

func (r RelevantVideosRepository) createVideos(ctx context.Context, id string, videos []model.Video) error {

	if len(videos) == 0 {
		return nil
	}

	relevantVideoDoc := r.db.Collection(relevantVideosNode).Doc(id)

	batchData := []database.DataBatch{}
	for _, video := range videos {
		docId := utils.Hash(video.Url)
		docRef := relevantVideoDoc.Collection(videosNode).Doc(docId)
		video.Id = &docId
		video.CreatedAt = time.Now().UTC()

		batchData = append(batchData, database.DataBatch{
			DocRef: docRef,
			Data:   video,
		})
	}

	_, err := r.db.SetDocs(ctx, batchData)

	if err != nil {
		err = fmt.Errorf("create relevant video docs: %w", err)
	}

	return err
}

func (r RelevantVideosRepository) getById(ctx context.Context, id string) (rv *model.RelevantVideos, err error) {

	query := r.db.Collection(relevantVideosNode).Query.Where(ProductIdFieldPath, ops.Equal, id)
	docs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return nil, fmt.Errorf("get relevant videos: %w, id: %s", err, id)
	}

	for _, doc := range docs {
		if !doc.Exists() {
			continue
		}

		// rv must not be nil
		rv = &model.RelevantVideos{}
		if e := doc.DataTo(rv); e != nil {
			return nil, fmt.Errorf("get relevant videos: %w, id: %s", err, id)
		}
		return rv, nil
	}

	return nil, nil
}

func (r RelevantVideosRepository) notifyOnChanges(ctx context.Context, query firestore.Query, where []filter.Where, kind firestore.DocumentChangeKind) <-chan RelevantVideosEvent {

	ch := make(chan RelevantVideosEvent)
	var writeFailureCount, writeFailureThreshold = 0, 3

	go func() {
		defer close(ch)

		helper.NotifyOnChanges(ctx, r.db, query, where, kind, func(dc firestore.DocumentChange, err error) error {

			if writeFailureCount > writeFailureThreshold {
				return fmt.Errorf("write failure threshould reached")
			}

			rv := model.RelevantVideos{}
			if err != nil && !(errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) {
				log.Error().Err(err).Msg("relevant videos repo: failed to read events")
				helper.NonblockingWrite[RelevantVideosEvent](ctx, channelWriteTimeout, ch, RelevantVideosEvent{RelevantVideos: rv, Err: err})
				return err
			}

			err = dc.Doc.DataTo(&rv)
			if err != nil {
				log.Error().Err(err).Msg("relevant videos repo: failed to convert doc to relevant video")
				return nil
			}

			err = helper.NonblockingWrite[RelevantVideosEvent](ctx, channelWriteTimeout, ch, RelevantVideosEvent{RelevantVideos: rv, Err: err})
			if err != nil {
				writeFailureCount++
			}

			return nil
		})

	}()

	return ch
}
