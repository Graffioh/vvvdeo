package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"veedeo/storage"
)

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
	fmt.Println(videoPath)
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

	log.Printf("range header: %v", rangeHeader)

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

func VideoHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	if isMP4Format(path) {
		handleMP4video(w, r)
	} else {
		http.NotFound(w, r)
	}
}

func VideoUploadNotificationHandler(w http.ResponseWriter, r *http.Request) {
	type Notification struct {
		VideoKey    string `json:"videoKey"`
		VideoStatus string `json:"videoStatus"`
	}
	var notification Notification

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Could not read body", http.StatusBadRequest)
		return
	}
	fmt.Printf("Raw Body: %s\n", string(bodyBytes))

	err = json.Unmarshal(bodyBytes, &notification)
	if err != nil {
		fmt.Printf("Decode Error: %v\n", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	fmt.Printf("VideoKey: %s\n", notification.VideoKey)
	fmt.Printf("VideoStatus: %s\n", notification.VideoStatus)

	var bucketName = os.Getenv("R2_BUCKET")

	go func() {
		err := storage.ProcessVideo(bucketName, notification.VideoKey)
		if err != nil {
			fmt.Fprintf(w, "Error processing the video! %v", err)
			return
		}
	}()

	fmt.Fprint(w, "Video notification sent", http.StatusOK)
}
