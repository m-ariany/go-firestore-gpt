package database

import (
	"context"

	"cloud.google.com/go/firestore"
)

type ChangeEvent struct {
	Change firestore.DocumentChange
	Err    error
}

type DataBatch struct {
	DocRef *firestore.DocumentRef
	Data   interface{}
}

// FIXME: this interface is very much firestore dependant. It should be decoupled from the underlying db technology
type Client interface {
	NotifyOnChanges(ctx context.Context, it *firestore.QuerySnapshotIterator, kind firestore.DocumentChangeKind) <-chan ChangeEvent
	GetDoc(ctx context.Context, docRef *firestore.DocumentRef) (*firestore.DocumentSnapshot, error)
	IterDocs(ctx context.Context, coll *firestore.CollectionRef, fn func(*firestore.DocumentSnapshot))
	UpdateDoc(ctx context.Context, docRef *firestore.DocumentRef, updates []firestore.Update, preconds ...firestore.Precondition) (_ *firestore.WriteResult, err error)
	SetDoc(ctx context.Context, docRef *firestore.DocumentRef, data interface{}, opts ...firestore.SetOption) (_ *firestore.WriteResult, err error)
	SetDocs(ctx context.Context, data []DataBatch) (_ []*firestore.WriteResult, err error)
	Collection(path string) *firestore.CollectionRef
	DeleteDoc(ctx context.Context, docRef *firestore.DocumentRef) (_ *firestore.WriteResult, err error)
	DeleteColl(ctx context.Context, collRef *firestore.CollectionRef)
}
