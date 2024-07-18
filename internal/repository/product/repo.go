package product

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"go-firestore-gpt/internal/database"
	ierr "go-firestore-gpt/internal/errors"
	"go-firestore-gpt/internal/model"
	"go-firestore-gpt/internal/repository/filter"
	"go-firestore-gpt/internal/repository/helper"
	"go-firestore-gpt/internal/utils"

	"cloud.google.com/go/firestore"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ProductRepository struct {
	db database.Client
}

var _ IRepository = ProductRepository{}

func New(db database.Client) ProductRepository {
	return ProductRepository{
		db: db,
	}
}

func (r ProductRepository) GetById(ctx context.Context, id string) (product *model.Product, err error) {

	docRef := r.db.Collection(productNode).Doc(id)
	docSnap, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ierr.NotFound
		}
		return nil, fmt.Errorf("get product: %w, id: %s", err, id)
	}

	if !docSnap.Exists() {
		return nil, nil
	}

	product = &model.Product{}
	if err = docSnap.DataTo(product); err != nil { // continue iteration to get the lastest version of the doc
		return nil, fmt.Errorf("get product: %w, id: %s", err, id)
	}

	err = r.setProductReviewAndQAs(ctx, docRef, product)
	return product, err
}

func (r ProductRepository) Create(ctx context.Context, data model.Product) error {

	p, err := r.GetById(ctx, *data.Id)
	if p != nil {
		return fmt.Errorf("create product: %w, id: %s", err, *data.Id)
	}

	if err != nil && err != ierr.NotFound {
		return fmt.Errorf("create product: %w, id: %s", err, *data.Id)
	}

	data.CreatedAt = time.Now().UTC()
	data.UpdatedAt = data.CreatedAt
	data.RelatedVideosAnalized = utils.BoolToPointer(false)
	data.SentimentAnalized = utils.BoolToPointer(false)
	docRef := r.db.Collection(productNode).Doc(*data.Id)
	_, err = r.db.SetDoc(ctx, docRef, data)

	if err != nil {
		return fmt.Errorf("create product: %w, id: %s", err, *data.Id)
	}

	if err := r.addProductReviews(ctx, data); err != nil {
		return fmt.Errorf("create product: %w, id: %s", err, *data.Id)
	}

	if err := r.addProductQAs(ctx, data); err != nil {
		return fmt.Errorf("create product: %w, id: %s", err, *data.Id)
	}

	return nil
}

func (r ProductRepository) Delete(ctx context.Context, id string) error {

	docRef := r.db.Collection(productNode).Doc(id)

	if docRef == nil {
		return nil
	}

	if _, err := r.db.DeleteDoc(ctx, docRef); err != nil {
		return fmt.Errorf("delete product: %w, id: %s", err, id)
	}

	return nil
}

func (r ProductRepository) addProductReviews(ctx context.Context, data model.Product) error {

	dataBatch := []database.DataBatch{}
	for _, review := range data.Reviews {
		docRef := r.db.Collection(productNode).Doc(*data.Id).Collection(reviewNode).NewDoc()
		review.CreatedAt = time.Now().UTC()
		dataBatch = append(dataBatch, database.DataBatch{
			DocRef: docRef,
			Data:   review,
		})
	}

	if _, err := r.db.SetDocs(ctx, dataBatch); err != nil {
		return fmt.Errorf("add product review: %w", err)
	}

	return nil
}

func (r ProductRepository) addProductQAs(ctx context.Context, data model.Product) error {

	for _, qa := range data.QAs {
		docRef := r.db.Collection(productNode).Doc(*data.Id).Collection(qasNode).NewDoc()
		qa.CreatedAt = time.Now().UTC()
		if _, err := r.db.SetDoc(ctx, docRef, qa); err != nil {
			return fmt.Errorf("add product qas: %w", err)
		}
	}

	return nil
}

func (r ProductRepository) Update(ctx context.Context, id string, data model.Product) error {
	docRef := r.db.Collection(productNode).Doc(id)
	updates := []firestore.Update{}

	updates = append(updates, firestore.Update{
		Path:  UpdatedAtFieldPath,
		Value: time.Now().UTC(),
	})

	if data.SentimentAnalized != nil {
		updates = append(updates, firestore.Update{
			Path:  SentimentAnalizedFieldPath,
			Value: *data.SentimentAnalized,
		})
	}

	if data.RelatedVideosAnalized != nil {
		updates = append(updates, firestore.Update{
			Path:  RelatedVideosAnalized,
			Value: *data.RelatedVideosAnalized,
		})
	}

	_, err := r.db.UpdateDoc(ctx, docRef, updates)
	if err != nil {
		return fmt.Errorf("update product: %w, id: %s", err, id)
	}
	return nil
}

func (r ProductRepository) NotifyOnAdded(ctx context.Context, where []filter.Where) <-chan ProductEvent {
	query := r.db.Collection(productNode).Query
	return r.notifyOnChanges(ctx, query, where, firestore.DocumentAdded)
}

func (r ProductRepository) notifyOnChanges(ctx context.Context, query firestore.Query, where []filter.Where, kind firestore.DocumentChangeKind) <-chan ProductEvent {

	ch := make(chan ProductEvent)
	var writeFailureCount, writeFailureThreshold int32 = 0, 3

	go func() {
		defer close(ch)

		helper.NotifyOnChanges(ctx, r.db, query, where, kind, func(dc firestore.DocumentChange, err error) error {

			if writeFailureCount > writeFailureThreshold {
				return fmt.Errorf("write failure threshould reached")
			}

			product := model.Product{}

			if err != nil && !(errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) {
				log.Error().Err(err).Msg("product repo: failed to read product events")
				helper.NonblockingWrite[ProductEvent](ctx, channelWriteTimeout, ch, ProductEvent{Product: product, Err: err})
				return err
			}

			err = dc.Doc.DataTo(&product)
			if err != nil {
				log.Error().Err(err).Msg("product repo: failed to convert doc to product")
				return nil
			}

			//dc.Doc.Ref
			docRef := r.db.Collection(productNode).Doc(*product.Id)

			// Reading reviews and QAs is a blocking operation, so we do it in a goroutine. Otherwise we might block the sender
			go func() {
				// err can be only ctx.Err()
				err = r.setProductReviewAndQAs(ctx, docRef, &product)
				if err := helper.NonblockingWrite[ProductEvent](ctx, channelWriteTimeout, ch, ProductEvent{Product: product, Err: err}); err != nil {
					atomic.AddInt32(&writeFailureCount, 1)
				}
			}()

			return nil
		})

	}()
	return ch
}

func (r ProductRepository) setProductReviewAndQAs(ctx context.Context, productRef *firestore.DocumentRef, product *model.Product) error {

	reviewsCh := r.productReviews(ctx, productRef)
	qasCh := r.productQAs(ctx, productRef)

	// no synchronization needed to increament finished, since select already synchronizes the access to it.
	finished := 0
	for finished != 2 {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case data, ok := <-reviewsCh:
			if !ok {
				continue
			}
			product.Reviews = data
			finished++

		case data, ok := <-qasCh:
			if !ok {
				continue
			}
			product.QAs = data
			finished++
		}
	}

	return nil
}

func (r ProductRepository) productQAs(ctx context.Context, productRef *firestore.DocumentRef) <-chan []model.ProductQA {
	ch := make(chan []model.ProductQA)

	go func() {
		defer close(ch)

		qas := make([]model.ProductQA, 0)

		colRef := r.db.Collection(productNode).Doc(productRef.ID).Collection(qasNode)
		r.db.IterDocs(ctx, colRef, func(ds *firestore.DocumentSnapshot) {
			qa := model.ProductQA{}
			if err := ds.DataTo(&qa); err != nil {
				return
			}
			qas = append(qas, qa)
		})

		select {
		case <-ctx.Done():
		case ch <- qas:
		}

	}()

	return ch
}

func (r ProductRepository) productReviews(ctx context.Context, productRef *firestore.DocumentRef) <-chan []model.ProductReview {
	ch := make(chan []model.ProductReview)

	go func() {
		defer close(ch)

		rws := make([]model.ProductReview, 0)

		colRef := r.db.Collection(productNode).Doc(productRef.ID).Collection(reviewNode)
		r.db.IterDocs(ctx, colRef, func(ds *firestore.DocumentSnapshot) {
			rw := model.ProductReview{}
			if err := ds.DataTo(&rw); err != nil {
				return
			}
			rws = append(rws, rw)
		})

		select {
		case <-ctx.Done():
		case ch <- rws:
		}

	}()

	return ch
}
