package common

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go-firestore-gpt/internal/eventpublisher/event"
)

var ErrWriteFailure = fmt.Errorf("write failure threshold exceeded")

type PublisherWithFailureThreshold struct {
	writeTimeout          time.Duration
	writeFailureThreshold int
	failureCount          map[event.EventWChannel]int
	failureMu             sync.Mutex
}

func NewPublisherWithFailureThreshold(writeTimeout time.Duration, writeFailureThreshold int) *PublisherWithFailureThreshold {
	return &PublisherWithFailureThreshold{
		writeTimeout:          writeTimeout,
		writeFailureThreshold: writeFailureThreshold,
		failureCount:          make(map[event.EventWChannel]int),
		failureMu:             sync.Mutex{},
	}
}

func (p *PublisherWithFailureThreshold) Publish(ctx context.Context, subscriber event.EventWChannel, e event.Event) (err error) {

	defer func() {
		// Since the subscriber channel may be closed after some failures,
		// it may happen that another execution of this func tries to write
		// on a closed subscriber and it causes a panic that should be recovered silently.
		if p := recover(); p != nil {
			err = ErrWriteFailure
		}
	}()

	ctx, cancel := context.WithTimeout(ctx, p.writeTimeout)
	defer cancel()

	select {
	case subscriber <- e:
		return nil
	case <-ctx.Done():
		p.failureMu.Lock()
		count := p.failureCount[subscriber] + 1
		p.failureCount[subscriber] = count
		p.failureMu.Unlock()

		if count >= p.writeFailureThreshold {
			err = ErrWriteFailure
			return
		}
		return nil
	}
}
