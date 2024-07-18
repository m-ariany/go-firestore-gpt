package model

import "time"

type RelevantVideos struct {
	ProductId   *string   `firestore:"productId,omitempty"`
	ProductName *string   `firestore:"productName,omitempty"`
	Videos      []Video   `firestore:"-"` // it is not a field but a collection
	Ready       *bool     `firestore:"ready,omitempty"`
	CreatedAt   time.Time `firestore:"createdAt,omitempty"`
	UpdatedAt   time.Time `firestore:"updatedAt,omitempty"`
}

type Video struct {
	Id        *string   `firestore:"id,omitempty"`
	Url       string    `firestore:"url,omitempty"`
	ThumbUp   int       `firestore:"thumbup,omitempty"`
	ThumbDown int       `firestore:"thumbdown,omitempty"`
	CreatedAt time.Time `firestore:"createdAt,omitempty"`
}
