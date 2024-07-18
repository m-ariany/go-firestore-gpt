package helper

import (
	"context"
	"encoding/json"
	"go-firestore-gpt/internal/database"
	"go-firestore-gpt/internal/repository/filter"
	"time"

	"cloud.google.com/go/firestore"
)

func NotifyOnChanges(ctx context.Context, db database.Client, query firestore.Query,
	where []filter.Where, kind firestore.DocumentChangeKind, fn func(firestore.DocumentChange, error) error) {

	for _, w := range where {
		query = query.Where(w.Path, w.Op, w.Value)
	}

	events := db.NotifyOnChanges(ctx, query.Snapshots(ctx), kind)

	for e := range events {
		if e.Err != nil {
			fn(e.Change, e.Err)
			return
		}

		if err := fn(e.Change, nil); err != nil {
			return
		}
	}
}

// drainChannelWithTimeout reads from the eventCh until it is closed, the context is done or the receiveTimeout is reached.
// The eventCh is unlikely to close since it is a firestore listener. So the receiveTimeout and context are the main ways to stop reading.
// The bigger the receiveTimeout the longer it waits for new events, which can lead to slower response time.
func DrainChannelWithTimeout(ctx context.Context, receiveTimeout time.Duration, eventCh <-chan database.ChangeEvent, eventProcessor func(database.ChangeEvent)) {

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(receiveTimeout):
			return
		case e, ok := <-eventCh:
			if !ok {
				return
			}

			if e.Err != nil {
				return
			}

			eventProcessor(e)
		}
	}
}

// NonblockingWrite is a generic function that can write any type of event to any channel type.
// T is the type parameter for the event.
func NonblockingWrite[T any](ctx context.Context, timeout time.Duration, ch chan<- T, event T) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	select {
	case ch <- event:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func Clone(src, dst interface{}) error {
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dst)
}
