package main

import (
	"log"
	"net/http"
	"os"
	"veedeo/handlers"
	"veedeo/storage"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
)

func main() {
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
		AllowedOrigins: []string{"http://localhost:9000", "https://vvvdeo.pages.dev", "https://vvvdeo.com", "http://localhost:5173"},
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

	mux.HandleFunc("/inference-video", handlers.VideoInferenceHandler)
	mux.HandleFunc("/presigned-url/put", storage.PresignedPutURLHandler)
	mux.HandleFunc("/presigned-url/get", storage.PresignedGetURLHandler)
	mux.HandleFunc("/notification/video-upload", handlers.VideoUploadNotificationFromWorkerHandler)
	mux.HandleFunc("/notification/frames-extraction", handlers.FrameNotificationFromWorkerHandler)
	mux.HandleFunc("/ws", handlers.WebSocketHandler)
	//mux.HandleFunc("/ytvideo", handlers.VideoStreamYTHandler)
	mux.HandleFunc("/video/speedup", handlers.VideoSpeedupHandler)
	mux.Handle("/metrics", promhttp.Handler())

	handler := c.Handler(mux)

	log.Println("Server starting on :8080...")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatal(err)
	}
}
