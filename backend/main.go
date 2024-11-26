package main

import (
	"log"
	"net/http"
	"veedeo/handlers"
	"veedeo/storage"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file!")
	}

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:8082", "http://localhost:9000", "http://127.0.0.1:8082"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"X-CSRF-Token",
			"X-Requested-With",
		},
		AllowCredentials: true,
	})

	mux := http.NewServeMux()

	mux.HandleFunc("/zawarudo/*", handlers.VideoHandler)
	mux.HandleFunc("/inference-video", handlers.InferenceVideoHandler)
	mux.HandleFunc("/presigned-put-url", storage.PresignedPutURLHandler)
	mux.HandleFunc("/presigned-get-url", storage.PresignedGetURLHandler)
	mux.HandleFunc("/video-upload-complete", handlers.VideoUploadNotificationHandler)
	mux.HandleFunc("/frames-extraction-complete", handlers.FrameExtractionNotificationHandler)
	mux.HandleFunc("/ws", handlers.WebSocketHandler)
	mux.Handle("/metrics", promhttp.Handler())

	handler := c.Handler(mux)

	log.Println("Server starting on :8080...")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatal(err)
	}
}
