package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/kkdai/youtube/v2"
)

type VideoRequest struct {
	URL string `json:"url"`
}

func VideoStreamYTHandler(w http.ResponseWriter, r *http.Request) {
	yt_client := youtube.Client{}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var video_req VideoRequest
	if err := json.Unmarshal(body, &video_req); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	video, err := yt_client.GetVideo(video_req.URL)
	if err != nil {
		http.Error(w, "Failed to fetch video details", http.StatusInternalServerError)
		return
	}

	fmt.Println(video.Title)

	formats := video.Formats.WithAudioChannels()

	fmt.Printf("%v", formats)

	stream, s_size, err := yt_client.GetStream(video, &formats[0])
	if err != nil {
		http.Error(w, "Failed to get video stream", http.StatusInternalServerError)
		return
	}
	defer stream.Close()

	fmt.Println(s_size)

	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	if _, err := io.Copy(w, stream); err != nil {
		http.Error(w, "Failed to stream video", http.StatusInternalServerError)
		return
	}
}
