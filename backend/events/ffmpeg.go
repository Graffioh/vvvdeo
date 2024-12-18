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
	w.Header().Set("Cache-Control", "no-cache")

	fmt.Println("1")

	for msg := range ch {
		fmt.Println("2")
		fmt.Fprintf(w, "data: %s\n\n", msg)
		w.(http.Flusher).Flush()
	}
}
