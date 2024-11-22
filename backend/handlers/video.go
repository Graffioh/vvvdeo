package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"veedeo/util"
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

func UploadVideoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Error parsing multipart form", http.StatusBadRequest)
		return
	}

	file, fileHeader, err := r.FormFile("video")
	videoName := fileHeader.Filename

	vidPath := filepath.Join("../sam2seg/vid", videoName)
	dst, err := os.Create(vidPath)
	if err != nil {
		http.Error(w, "Error creating file", http.StatusInternalServerError)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, "Error saving file in img dir", http.StatusInternalServerError)
	}

	util.ConvertIntoFrames(videoName)
}
