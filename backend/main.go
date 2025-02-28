package main

import (
	"log"
	"net/http"
	"os"
	"veedeo/events"
	"veedeo/storage"
	"veedeo/video"
	"veedeo/websocket"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
)

func main() {
	h := setupServerHandler()

	log.Println("Server starting on :8080...")
	if err := http.ListenAndServe(":8080", h); err != nil {
		log.Fatal(err)
	}
}

func setupServerHandler() http.Handler {
	env := os.Getenv("APP_ENV")
	if env != "PROD" {
		err := godotenv.Load()
		if err != nil {
			log.Println("Error loading .env file!")
		}
	} else {
		log.Println("Running in production mode, skipping .env file")
	}

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:9000", "https://vvvdeo.pages.dev", "https://vvvdeo.com", "http://localhost:5173", "http://localhost:5174", "https://api.vvvdeo.com"},
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

	// mux.HandleFunc("/video/inference", video.VideoInferenceHandler)
	mux.HandleFunc("/video/local-inference", video.VideoLocalInferenceHandler)
	mux.HandleFunc("/presigned-url/put", storage.PresignedPutURLHandler)
	mux.HandleFunc("/presigned-url/get", storage.PresignedGetURLHandler)
	// mux.HandleFunc("/notification/video-upload", notification.VideoUploadNotificationFromWorkerHandler)
	// mux.HandleFunc("/notification/frames-extraction", notification.FrameNotificationFromWorkerHandler)
	mux.HandleFunc("/ws", websocket.WebSocketHandler)
	mux.HandleFunc("/video/speedup", video.VideoSpeedupHandler)
	mux.HandleFunc("/ffmpeg-events", events.FfmpegEventsHandler)
	mux.Handle("/metrics", promhttp.Handler())

	return c.Handler(mux)
}
