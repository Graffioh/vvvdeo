package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"veedeo/storage"
)

type Notification struct {
	VideoKey string `json:"videoKey"`
	Status   string `json:"status"`
}

func VideoUploadNotificationFromWorkerHandler(w http.ResponseWriter, r *http.Request) {
	var video_notification Notification

	err := json.NewDecoder(r.Body).Decode(&video_notification)
	if err != nil {
		fmt.Printf("Video notification decode Error: %v\n", err)
		http.Error(w, "Invalid Video upload notification request payload", http.StatusBadRequest)
		return
	}

	fmt.Printf("VideoKey: %s\n", video_notification.VideoKey)
	fmt.Printf("VideoStatus: %s\n", video_notification.Status)

	// convert the video into frames and store them in r2 bucket
	var bucketName = os.Getenv("R2_BUCKET")
	go func() {
		err := storage.ProcessVideo(bucketName, video_notification.VideoKey)
		if err != nil {
			fmt.Fprintf(w, "Error processing the video! %v", err)
			return
		}
	}()

	fmt.Fprint(w, "Video upload notification received.", http.StatusOK)
}

func FrameNotificationFromWorkerHandler(w http.ResponseWriter, r *http.Request) {
	type FrameExtractionMessage struct {
		Message  string `json:"message"`
		Status   string `json:"status"`
		VideoKey string `json:"videoKey"`
	}

	var frames_notification Notification

	err := json.NewDecoder(r.Body).Decode(&frames_notification)
	if err != nil {
		fmt.Printf("Frame notification decode Error: %v\n", err)
		http.Error(w, "Invalid Frame extraction notification request payload", http.StatusBadRequest)
		return
	}

	fmt.Printf("FrameKey: %s\n", frames_notification.VideoKey)
	fmt.Printf("FrameStatus: %s\n", frames_notification.Status)

	videoNameKeyParts := strings.Split(frames_notification.VideoKey, "/")[1]
	videoNameKey := strings.TrimSuffix(videoNameKeyParts, ".zip")
	videoKey := "videos/" + videoNameKey

	// send notification to frontend client
	wsConnection := key_socket_connections[videoNameKey]
	if err = wsConnection.WriteJSON(FrameExtractionMessage{
		Message:  "frame extraction",
		Status:   "completed",
		VideoKey: videoKey,
	}); err != nil {
		log.Printf("ERROR SENDING MESSAGE TO FRONTEND FROM WEBSOCKET! error: %v", err)
		return
	}

	fmt.Fprint(w, "Frame extraction complete.", http.StatusOK)
}
