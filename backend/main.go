package main

import (
	"fmt"
	"log"
	"net/http"
	"veedeo/handlers"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	const port = 8080
	http.HandleFunc("/zawarudo/*", handlers.VideoHandler)
	http.HandleFunc("/inference-frame", handlers.InferenceFrameHandler)
	http.HandleFunc("/inference-video", handlers.InferenceVideoHandler)

	http.Handle("/metrics", promhttp.Handler())

	fmt.Printf("Server running on %v\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", port), nil))
}
