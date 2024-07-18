package model

import "time"

type ReviewSentiments struct {
	ProductId  *string     `firestore:"productId,omitempty"`
	Sentiments []Sentiment `firestore:"-"` // it is not a field but a collection
	CreatedAt  time.Time   `firestore:"createdAt,omitempty"`
	UpdatedAt  time.Time   `firestore:"updatedAt,omitempty"`
}

type Sentiment struct {
	Label     string    `firestore:"label,omitempty"`
	Score     int       `firestore:"score,omitempty"`
	CreatedAt time.Time `firestore:"createdAt,omitempty"`
}
