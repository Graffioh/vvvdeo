package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"veedeo/util"
)

func VideoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	path := r.URL.Path

	if strings.HasSuffix(path, ".m3u8") || strings.HasSuffix(path, ".ts") {
		if _, err := os.Stat("video-hls/master.m3u8"); os.IsNotExist(err) {
			done := make(chan bool)
			go func() {
				util.HLSConverter()
				done <- true
			}()
			<-done
		}

		handleHLSvideo(w, r)
	} else if strings.HasSuffix(path, ".mpd") || strings.HasSuffix(path, ".webm") {
		if _, err := os.Stat("video-dash/my_video_manifest.mpd"); os.IsNotExist(err) {
			done := make(chan bool)
			go func() {
				util.DASHConverter()
				done <- true
			}()
			<-done
		}

		handleDASHvideo(w, r)
	} else if strings.HasSuffix(path, ".mp4") {
		handleMP4video(w, r)
	} else {
		http.NotFound(w, r)
	}
}

func min(a int64, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func handleMP4video(w http.ResponseWriter, r *http.Request) {
	videoPath := "./jojorun.mp4"
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
	// w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")

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

func handleHLSvideo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	fmt.Println(r.URL.Path)
	fileName := strings.Split(r.URL.Path, "/")[2]
	fmt.Println(fileName)
	if strings.HasSuffix(r.URL.Path, ".m3u8") {
		playlistData, err := os.ReadFile("../video-hls/" + fileName)
		if err != nil {
			http.Error(w, "Error reading HLS playlist file", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/x-mpegURL")
		w.Header().Set("Cache-Control", "public, max-age=60, must-revalidate")
		w.WriteHeader(http.StatusOK)
		w.Write(playlistData)
		return
	}

	if strings.HasSuffix(r.URL.Path, ".ts") {
		segmentPath := "../video-hls/" + fileName
		w.Header().Set("Cache-Control", "public, max-age=86400, immutable")
		http.ServeFile(w, r, segmentPath)
		return
	}

	http.Error(w, "Invalid HLS request", http.StatusBadRequest)
}

func handleDASHvideo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	fileName := strings.Split(r.URL.Path, "/")[2]
	if strings.HasSuffix(r.URL.Path, ".mpd") {
		manifestData, err := os.ReadFile("../video-dash/" + fileName)
		if err != nil {
			http.Error(w, "Error reading DASH manifest file", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/dash+xml")
		w.Header().Set("Cache-Control", "public, max-age=60, must-revalidate")
		w.WriteHeader(http.StatusOK)
		w.Write(manifestData)
		return
	}

	if strings.HasSuffix(r.URL.Path, ".webm") {
		segmentPath := "../video-dash/" + fileName
		w.Header().Set("Cache-Control", "public, max-age=86400, immutable")
		http.ServeFile(w, r, segmentPath)
		return
	}

	http.Error(w, "Invalid DASH request", http.StatusBadRequest)
}
