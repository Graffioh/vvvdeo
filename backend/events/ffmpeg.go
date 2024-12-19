package events

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

var SseManager = NewSSEManager()

func FfmpegEventsHandler(w http.ResponseWriter, r *http.Request) {
	id := uuid.New().String()
	ch := SseManager.Subscribe(id)
	defer SseManager.Unsubscribe(id)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")

	for msg := range ch {
		content := fmt.Sprintf("data: %s\n\n", msg)
		w.Write([]byte(content))
		w.(http.Flusher).Flush()
	}
}
