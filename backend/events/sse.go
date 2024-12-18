package events

import (
	"sync"
)

type SSEManager struct {
	subscribers map[string]chan string
	mutex       sync.Mutex
}

func NewSSEManager() *SSEManager {
	return &SSEManager{
		subscribers: make(map[string]chan string),
	}
}

func (s *SSEManager) Subscribe(id string) chan string {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	ch := make(chan string, 20)
	s.subscribers[id] = ch

	return ch
}

func (s *SSEManager) Unsubscribe(id string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if ch, ok := s.subscribers[id]; ok {
		close(ch)
		delete(s.subscribers, id)
	}
}

func (s *SSEManager) Update(message string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for id, ch := range s.subscribers {
		select {
		case ch <- message:
		default:
			close(ch)
			delete(s.subscribers, id)
		}
	}
}
