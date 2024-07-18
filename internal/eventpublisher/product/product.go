package product

import (
	"context"
	"time"

	"go-firestore-gpt/internal/eventpublisher"
	"go-firestore-gpt/internal/eventpublisher/common"
	"go-firestore-gpt/internal/eventpublisher/event"
	productRepo "go-firestore-gpt/internal/repository/product"

	"github.com/rs/zerolog/log"
)

const (
	writeTimeout          = time.Second
	writeFailureThreshold = 3
)

type eventFunc func(context.Context) <-chan productRepo.ProductEvent

type ProductPublisher interface {
	eventpublisher.Publisher
	Start(ctx context.Context) error
}

type productPublisher struct {
	eventFn    eventFunc
	submanager common.SubManager
	publisher  common.PublisherWithFailureThreshold
}

func new(fn eventFunc) ProductPublisher {
	return &productPublisher{
		eventFn:    fn,
		submanager: *common.NewSubManager(),
		publisher:  *common.NewPublisherWithFailureThreshold(writeTimeout, writeFailureThreshold),
	}
}

func (p *productPublisher) Subscribe(subscriber event.EventWChannel) {
	p.submanager.Subscribe(subscriber)
}

func (p *productPublisher) Unsubscribe(subscriber event.EventWChannel) {
	p.submanager.Unsubscribe(subscriber)
}

func (p *productPublisher) publish(ctx context.Context, productEvent productRepo.ProductEvent) {
	p.submanager.OnSubscribers(func(subscriber event.EventWChannel) {
		go func() {
			if err := p.publisher.Publish(ctx,
				subscriber,
				event.Event{Message: productEvent.Product, Err: productEvent.Err}); err != nil {
				p.Unsubscribe(subscriber)
			}
		}()
	})
}

func (p *productPublisher) Start(ctx context.Context) error {
	defer p.submanager.UnsubscribeAll()

	eventsCh := p.eventFn(ctx)
	for {
		select {
		case <-ctx.Done():
			log.Error().Err(ctx.Err()).Msg("ProductPublisher stopped")
			return ctx.Err()
		case e, ok := <-eventsCh:
			if !ok {
				return nil
			}
			log.Debug().Msgf("publish productId %s", *e.Product.Id)
			p.publish(ctx, e)
		}
	}
}
