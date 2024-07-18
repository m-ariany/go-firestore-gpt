package model

import "time"

type Product struct {
	Id                    *string         `firestore:"id,omitempty"`
	Name                  *string         `firestore:"name,omitempty"`
	Description           *string         `firestore:"description,omitempty"`
	SentimentAnalized     *bool           `firestore:"sentimentAnalized,omitempty"`
	RelatedVideosAnalized *bool           `firestore:"relatedVideosAnalized,omitempty"`
	QAs                   []ProductQA     `firestore:"-"` // it is not a field but a collection
	Reviews               []ProductReview `firestore:"-"` // it is not a field but a collection
	CreatedAt             time.Time       `firestore:"createdAt,omitempty"`
	UpdatedAt             time.Time       `firestore:"updatedAt,omitempty"`
}

type ProductQA struct {
	Question  *string   `firestore:"question,omitempty"`
	Answer    *string   `firestore:"answer,omitempty"`
	CreatedAt time.Time `firestore:"createdAt,omitempty"`
}

type ProductReview struct {
	Rating    *int      `firestore:"rating,omitempty"`
	Comment   *string   `firestore:"comment,omitempty"`
	CreatedAt time.Time `firestore:"createdAt,omitempty"`
}
