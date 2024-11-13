package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func min(a int64, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func videoHandlerMP4(w http.ResponseWriter, r *http.Request) {
	videoPath := "./sam2-try/jojorun.mp4"
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

func videoHandlerHLS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	fmt.Println(r.URL.Path)
	fileName := strings.Split(r.URL.Path, "/")[2]
	fmt.Println(fileName)
	if strings.HasSuffix(r.URL.Path, ".m3u8") {
		playlistData, err := os.ReadFile("./video-hls/" + fileName)
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
		segmentPath := "./video-hls/" + fileName
		w.Header().Set("Cache-Control", "public, max-age=86400, immutable")
		http.ServeFile(w, r, segmentPath)
		return
	}

	http.Error(w, "Invalid HLS request", http.StatusBadRequest)
}

func videoHandlerDASH(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	fileName := strings.Split(r.URL.Path, "/")[2]
	if strings.HasSuffix(r.URL.Path, ".mpd") {
		manifestData, err := os.ReadFile("./video-dash/" + fileName)
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
		segmentPath := "./video-dash/" + fileName
		w.Header().Set("Cache-Control", "public, max-age=86400, immutable")
		http.ServeFile(w, r, segmentPath)
		return
	}

	http.Error(w, "Invalid DASH request", http.StatusBadRequest)
}

func videoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	path := r.URL.Path

	if strings.HasSuffix(path, ".m3u8") || strings.HasSuffix(path, ".ts") {
		if _, err := os.Stat("video-hls/master.m3u8"); os.IsNotExist(err) {
			done := make(chan bool)
			go func() {
				HLSConverter()
				done <- true
			}()
			<-done
		}

		videoHandlerHLS(w, r)
	} else if strings.HasSuffix(path, ".mpd") || strings.HasSuffix(path, ".webm") {
		if _, err := os.Stat("video-dash/my_video_manifest.mpd"); os.IsNotExist(err) {
			done := make(chan bool)
			go func() {
				DASHConverter()
				done <- true
			}()
			<-done
		}

		videoHandlerDASH(w, r)
	} else {
		http.NotFound(w, r)
	}
}

type VideoCoordinates struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
}

type Points struct {
	Coordinates []VideoCoordinates `json:"coordinates"`
	Labels      []int32            `json:"labels"`
}

func HLSConverter() {
	fmt.Println("Starting HLS conversion...")
	cmd := exec.Command("./convert-to-hls.sh")

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		log.Fatal("convert-to-hls.sh cmd execution error:", err)
	}
	fmt.Println("HLS conversion complete.")
}

func DASHConverter() {
	fmt.Println("Starting MPEG-DASH conversion...")
	cmd := exec.Command("./convert-to-dash.sh", "all")

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		log.Fatal("convert-to-dash.sh cmd execution error:", err)
	}
	fmt.Println("MPEG-DASH conversion complete.")
}

func inferenceHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// Print the body
	fmt.Printf("R BODY: %s\n", body)

	// Optionally, you can unmarshal the body into a struct if it's JSON
	var points Points
	if err := json.Unmarshal(body, &points); err != nil {
		http.Error(w, "Error parsing JSON body", http.StatusBadRequest)
		return
	}

	fmt.Printf("points: %v\n", points)

	// Convert Go struct to JSON for the Python API
	coordsJSON, err := json.Marshal(points)
	if err != nil {
		http.Error(w, "Failed to create JSON", http.StatusInternalServerError)
		return
	}

	fmt.Printf("json: %v\n", coordsJSON)

	// Send the request to the Python server
	pythonURL := "http://localhost:9000/predict" // Python server URL
	resp, err := http.Post(pythonURL, "application/json", bytes.NewBuffer(coordsJSON))
	if err != nil {
		http.Error(w, "Error communicating with Python server", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Parse response from Python server
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		http.Error(w, "Error decoding Python server response", http.StatusInternalServerError)
		return
	}

	// Respond to the web app with the Python server's result
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func inferenceFramesHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("AAAAAAAAAAAAAAA")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// Print the body
	fmt.Printf("R BODY: %s\n", body)

	// Optionally, you can unmarshal the body into a struct if it's JSON
	var points Points
	if err := json.Unmarshal(body, &points); err != nil {
		http.Error(w, "Error parsing JSON body", http.StatusBadRequest)
		return
	}

	fmt.Printf("points: %v\n", points)

	// Convert Go struct to JSON for the Python API
	coordsJSON, err := json.Marshal(points)
	if err != nil {
		http.Error(w, "Failed to create JSON", http.StatusInternalServerError)
		return
	}

	fmt.Printf("json: %v\n", coordsJSON)

	// Send the request to the Python server
	pythonURL := "http://localhost:9000/predict-frames" // Python server URL
	resp, err := http.Post(pythonURL, "application/json", bytes.NewBuffer(coordsJSON))
	if err != nil {
		http.Error(w, "Error communicating with Python server", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Parse response from Python server
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		http.Error(w, "Error decoding Python server response", http.StatusInternalServerError)
		return
	}

	// Respond to the web app with the Python server's result
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func segmentedFrameHandler(w http.ResponseWriter, r *http.Request) {
	// Define the directory to search for files.
	dir := "./sam2-try/static/"

	// Find the latest file in the directory.
	latestFile, err := getLatestFile(dir)
	fmt.Println("LATEST FILE: " + latestFile)
	if err != nil {
		http.Error(w, "Error finding latest file", http.StatusInternalServerError)
		return
	}

	// Serve the latest file.
	http.ServeFile(w, r, latestFile)
}

// getLatestFile finds the most recently added file in a directory.
func getLatestFile(dir string) (string, error) {
	var latestFile string
	var latestModTime time.Time

	// Walk through the directory to find the most recently modified file.
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Check if the path is a file and if it's newer than the current latest.
		if !info.IsDir() && info.ModTime().After(latestModTime) {
			latestFile = path
			latestModTime = info.ModTime()
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	// If no file was found, return an error.
	if latestFile == "" {
		return "", os.ErrNotExist
	}

	return latestFile, nil
}

func main() {
	const port = 8080
	http.HandleFunc("/zawarudo.mp4", videoHandlerMP4)
	http.HandleFunc("/zawarudo/*", videoHandler)
	http.HandleFunc("/inference", inferenceHandler)
	http.HandleFunc("/inference-frames", inferenceFramesHandler)
	http.HandleFunc("/segmented-frame", segmentedFrameHandler)

	http.Handle("/metrics", promhttp.Handler())

	fmt.Printf("Server running on %v\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", port), nil))
}
