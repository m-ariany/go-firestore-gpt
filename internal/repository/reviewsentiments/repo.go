package reviewsentiments

import (
	"context"
	"fmt"
	"time"

	"go-firestore-gpt/internal/database"
	"go-firestore-gpt/internal/model"
	"go-firestore-gpt/internal/utils"
)

type ReviewSentimentsRepository struct {
	db database.Client
}

var _ IRepository = ReviewSentimentsRepository{}

func New(db database.Client) ReviewSentimentsRepository {
	return ReviewSentimentsRepository{
		db: db,
	}
}

func (r ReviewSentimentsRepository) Create(ctx context.Context, data model.ReviewSentiments) error {

	data.CreatedAt = time.Now().UTC()
	data.UpdatedAt = data.CreatedAt
	docRef := r.db.Collection(reviewSentimentsNode).Doc(*data.ProductId)
	_, err := r.db.SetDoc(ctx, docRef, data)

	if err != nil {
		err = fmt.Errorf("create review sentiments: %w, id: %s", err, docRef.ID)
		return err
	}

	err = r.createSentiments(ctx, docRef.ID, data.Sentiments)

	return err
}

func (r ReviewSentimentsRepository) createSentiments(ctx context.Context, id string, sentiments []model.Sentiment) error {

	if len(sentiments) == 0 {
		return nil
	}

	reviewSentimentsDoc := r.db.Collection(reviewSentimentsNode).Doc(id)

	batchData := []database.DataBatch{}
	for _, sentiment := range sentiments {
		// Next update should be done on the same doc. So, we use hash(the sentiment.Label) as the doc id
		docId := utils.Hash(sentiment.Label)
		docRef := reviewSentimentsDoc.Collection(sentimentsNode).Doc(docId)
		sentiment.CreatedAt = time.Now().UTC()

		batchData = append(batchData, database.DataBatch{
			DocRef: docRef,
			Data:   sentiment,
		})
	}

	_, err := r.db.SetDocs(ctx, batchData)

	if err != nil {
		err = fmt.Errorf("create review sentiment docs: %w", err)
	}

	return err
}

func (r ReviewSentimentsRepository) GetById(ctx context.Context, id string) (rv *model.ReviewSentiments, err error) {

	docRef := r.db.Collection(reviewSentimentsNode).Doc(id)
	docSnap, err := docRef.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("get review sentiments: %w, id: %s", err, id)
	}

	if !docSnap.Exists() {
		return nil, nil
	}

	rv = &model.ReviewSentiments{}
	if e := docSnap.DataTo(rv); e != nil {
		return nil, fmt.Errorf("get review sentiments: %w, id: %s", err, id)
	}
	return rv, nil
}
