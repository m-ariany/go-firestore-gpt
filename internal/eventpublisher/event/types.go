package event

type (
	EventHandler func()

	EventType int

	Event struct {
		Message interface{}
		Err     error
	}

	EventChannel  chan Event
	EventWChannel chan<- Event
)

const (
	DbDocAdded EventType = iota
	DbDocChanged
	DbDocDeleted
)
