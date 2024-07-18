package eventpublisher

import (
	"go-firestore-gpt/internal/eventpublisher/event"
)

type Publisher interface {
	Subscribe(event.EventWChannel)
	Unsubscribe(event.EventWChannel)
}
