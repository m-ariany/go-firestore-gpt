package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/rs/zerolog/log"
	"google.golang.org/api/iterator"
)

type snapEvent struct {
	snap *firestore.QuerySnapshot
	err  error
}

type snapCh chan snapEvent

type FirestoreClient struct {
	*firestore.Client
	writeTimeout time.Duration
}

func New(client *firestore.Client) FirestoreClient {
	return FirestoreClient{
		Client:       client,
		writeTimeout: time.Second * 120,
	}
}

// This function listens to the given SnapshotIterator and put all the events on the ChangeEvent channel.
// The cicuite breaker pattern here defines a error rate tolarance cap. If the listener raises error more than
// the given cap, it stops the listener and closes the ChangeEvent channel.
func (c FirestoreClient) NotifyOnChanges(ctx context.Context, it *firestore.QuerySnapshotIterator, kind firestore.DocumentChangeKind) <-chan ChangeEvent {

	ch := make(chan ChangeEvent)
	errToleranceCap := 20
	errCnt := 0

	go func() {
		defer close(ch)

		eventCh := registerEventListener(ctx, it)
		for event := range eventCh {
			if event.err != nil {
				// The error is not wrapped properly, so errors.Is() does not work
				if strings.Contains(event.err.Error(), "context canceled") || strings.Contains(event.err.Error(), "context deadline exceeded") {
					return
				}

				log.Error().Err(event.err).Msg("error reading events")
				errCnt++
				if errCnt < errToleranceCap {
					continue
				}
				ch <- ChangeEvent{Err: event.err}
				return
			}

			for _, change := range event.snap.Changes {
				if change.Kind == kind {
					if change.Doc == nil {
						continue
					}

					if !change.Doc.Exists() {
						continue
					}
					// FIXME: make this logic more rubust and configurable
					select {
					case ch <- ChangeEvent{Change: change}:
						continue
					case <-time.After(time.Minute):
						log.Error().Msg("timedout to deliver a change to the client")
					}
				}
			}
		}
	}()

	return ch
}

// registerEventListener keeps the listener open until context is cancelled
func registerEventListener(ctx context.Context, it *firestore.QuerySnapshotIterator) <-chan snapEvent {

	threshold := 5
	retry := 0
	c := make(snapCh)
	go func() {
		defer close(c)
		defer it.Stop()

		for {
			snap, err := it.Next()
			if err == iterator.Done {
				return
			}

			select {
			case <-ctx.Done():
				return
			case c <- snapEvent{snap, err}:
				continue
			case <-time.After(time.Second * 10):
				// FIXME: return or continue? Make the timeout configurable
				log.Error().Msg("timedout to deliver a snapshot to the client")
				retry++
				if retry > threshold {
					return
				}
			}
		}
	}()

	return c
}

// Iterate over all the docs of the given coll
func (c FirestoreClient) IterDocs(ctx context.Context, coll *firestore.CollectionRef, fn func(*firestore.DocumentSnapshot)) {
	iter := coll.Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err != nil {
			if err == iterator.Done || strings.Contains(err.Error(), "context canceled") || strings.Contains(err.Error(), "context deadline exceeded") {
				return
			}
			continue
		}

		fn(doc)
	}
}

func (c FirestoreClient) GetDoc(ctx context.Context, docRef *firestore.DocumentRef) (*firestore.DocumentSnapshot, error) {
	ctx, cancel := context.WithTimeout(ctx, c.writeTimeout)
	defer cancel()

	docSnapshot, err := docRef.Get(ctx)
	if err != nil {
		return nil, err
	}

	if !docSnapshot.Exists() {
		return nil, fmt.Errorf("doc snapshot does not exist")
	}

	return docSnapshot, nil
}

func (c FirestoreClient) UpdateDoc(ctx context.Context, docRef *firestore.DocumentRef, updates []firestore.Update, preconds ...firestore.Precondition) (_ *firestore.WriteResult, err error) {
	ctx, cancel := context.WithTimeout(ctx, c.writeTimeout)
	defer cancel()

	return docRef.Update(ctx, updates, preconds...)
}

func (c FirestoreClient) SetDoc(ctx context.Context, docRef *firestore.DocumentRef, data interface{}, opts ...firestore.SetOption) (_ *firestore.WriteResult, err error) {
	ctx, cancel := context.WithTimeout(ctx, c.writeTimeout)
	defer cancel()

	return docRef.Set(ctx, data, opts...)
}

func (c FirestoreClient) SetDocs(ctx context.Context, data []DataBatch) (_ []*firestore.WriteResult, err error) {
	ctx, cancel := context.WithTimeout(ctx, c.writeTimeout)
	defer cancel()

	batch := c.Client.Batch()
	for _, item := range data {
		batch.Set(item.DocRef, item.Data)
	}

	return batch.Commit(ctx)
}

func (c FirestoreClient) DeleteDoc(ctx context.Context, docRef *firestore.DocumentRef) (_ *firestore.WriteResult, err error) {
	ctx, cancel := context.WithTimeout(ctx, c.writeTimeout)
	defer cancel()
	colls, err := docRef.Collections(ctx).GetAll()
	if err != nil {
		log.Error().Err(err).Msgf("failed to get all collections of the doc %s", docRef.Path)
		return nil, err
	}

	for _, collRef := range colls {
		// must not be concurrent otherwise subcolls will not be cleaned up due to context cancellation
		c.DeleteColl(ctx, collRef)

	}

	if docRef != nil {
		docRef.Delete(ctx)
	}

	return nil, nil
}

func (c FirestoreClient) DeleteColl(ctx context.Context, collRef *firestore.CollectionRef) {
	// Recursively delete all subcollections
	docs := collRef.Documents(ctx)
	for {
		doc, err := docs.Next()
		if err == iterator.Done {
			return
		}
		c.DeleteDoc(ctx, doc.Ref)
	}
}
