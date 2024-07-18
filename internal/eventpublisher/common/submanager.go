package common

import (
	"sync"

	"go-firestore-gpt/internal/eventpublisher/event"
)

type SubManager struct {
	subscribers    map[event.EventWChannel]struct{}
	subscriptionMu sync.RWMutex
}

func NewSubManager() *SubManager {
	return &SubManager{
		subscribers:    make(map[event.EventWChannel]struct{}),
		subscriptionMu: sync.RWMutex{},
	}
}

func (m *SubManager) Subscribe(subscriber event.EventWChannel) {
	m.subscriptionMu.Lock()
	defer m.subscriptionMu.Unlock()

	if _, ok := m.subscribers[subscriber]; !ok {
		m.subscribers[subscriber] = struct{}{}
	}
}

func (m *SubManager) Unsubscribe(subscriber event.EventWChannel) {
	m.subscriptionMu.Lock()
	defer m.subscriptionMu.Unlock()

	// only act on the subscribed channels
	if _, ok := m.subscribers[subscriber]; !ok {
		return
	}
	delete(m.subscribers, subscriber)
	close(subscriber)
}

func (m *SubManager) UnsubscribeAll() {
	for subscriber := range m.subscribers {
		m.Unsubscribe(subscriber)
	}
}

func (m *SubManager) OnSubscribers(do func(event.EventWChannel)) {
	m.subscriptionMu.RLock()

	// Caution: The 'do' function may modify the 'subscribers' map during iteration.
	// To avoid unexpected bugs caused by channel deletion on the iterating map,
	// we create a separate list of channels for processing.
	var subsCopy []event.EventWChannel
	for subscriber := range m.subscribers {
		subsCopy = append(subsCopy, subscriber)
	}
	m.subscriptionMu.RUnlock()

	for _, subscriber := range subsCopy {
		do(subscriber)
	}
}
