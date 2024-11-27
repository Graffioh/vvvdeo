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

/*
func min(a int64, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func getLastUploadedVideo(w http.ResponseWriter) string {
	videoDir := "../sam2seg/vid"

	files, err := os.ReadDir(videoDir)
	if err != nil {
		log.Printf("Error reading video directory: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return ""
	}

	var mp4Files []os.DirEntry
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".mp4" {
			mp4Files = append(mp4Files, file)
		}
	}

	if len(mp4Files) == 0 {
		http.Error(w, "No MP4 files found", http.StatusNotFound)
		return ""
	}

	sort.Slice(mp4Files, func(i, j int) bool {
		infoI, _ := mp4Files[i].Info()
		infoJ, _ := mp4Files[j].Info()
		return infoI.ModTime().After(infoJ.ModTime())
	})

	return filepath.Join(videoDir, mp4Files[0].Name())
}

func handleMP4video(w http.ResponseWriter, r *http.Request) {
	videoPath := getLastUploadedVideo(w)
	videoData, err := os.Open(videoPath)
	if err != nil {
		log.Printf("Error opening video file: %v", err)

		if os.IsNotExist(err) {
			http.Error(w, "Video file not found", http.StatusNotFound)
			return
		}

		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer videoData.Close()

	videoInfo, err := os.Stat(videoPath)
	if err != nil {
		log.Printf("Error getting video stats: %v", err)

		if os.IsNotExist(err) {
			http.Error(w, "Video file not found", http.StatusNotFound)
			return
		}

		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	videoSize := videoInfo.Size()

	var start, end int64
	const chunk_size = 1 * 1024 * 1024 // 1MB

	rangeHeader := r.Header.Get("Range")
	if rangeHeader == "" {
		start = 0
		end = videoSize - 1
	} else {
		start, err = strconv.ParseInt(strings.Split(strings.Split(rangeHeader, "=")[1], "-")[0], 10, 64)
		if err != nil {
			log.Printf("Error converting start bytes from string to int64: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		end = min(start+chunk_size-1, videoSize-1)
	}

	remainingBytes := end - start + 1

	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", remainingBytes))
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, videoSize))

	w.WriteHeader(http.StatusPartialContent)

	_, err = videoData.Seek(start, 0)
	if err != nil {
		log.Printf("Error seeking video file: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	_, err = io.CopyN(w, videoData, remainingBytes)
	if err != nil {
		log.Printf("Error copying video data to response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

func isMP4Format(path string) bool {
	return strings.HasSuffix(path, ".mp4")
}

func PlayVideoHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	if isMP4Format(path) {
		handleMP4video(w, r)
	} else {
		http.NotFound(w, r)
	}
	}*/

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

	var bucketName = os.Getenv("R2_BUCKET")

	// convert the video into frames and store them in r2 bucket
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

	videoNameKey := strings.Split(frames_notification.VideoKey, "/")[1]
	videoKey := "videos/" + videoNameKey

	// send notification to frontend client
	wsConnection := key_socket_connections[videoNameKey]
	if err = wsConnection.WriteJSON(FrameExtractionMessage{
		Message:  "frame extraction",
		Status:   "completed",
		VideoKey: videoKey,
	}); err != nil {
		log.Println(err)
		return
	}

	fmt.Fprint(w, "Frame extraction complete.", http.StatusOK)
}
