package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"veedeo/handlers"
	"veedeo/util"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func videoHandler(w http.ResponseWriter, r *http.Request) {
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

		handlers.HandleHLSvideo(w, r)
	} else if strings.HasSuffix(path, ".mpd") || strings.HasSuffix(path, ".webm") {
		if _, err := os.Stat("video-dash/my_video_manifest.mpd"); os.IsNotExist(err) {
			done := make(chan bool)
			go func() {
				util.DASHConverter()
				done <- true
			}()
			<-done
		}

		handlers.HandleDASHvideo(w, r)
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

func inferenceFrameHandler(w http.ResponseWriter, r *http.Request) {
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

	fmt.Printf("R BODY: %s\n", body)

	var points Points
	if err := json.Unmarshal(body, &points); err != nil {
		http.Error(w, "Error parsing JSON body", http.StatusBadRequest)
		return
	}

	fmt.Printf("points: %v\n", points)

	coordsJSON, err := json.Marshal(points)
	if err != nil {
		http.Error(w, "Failed to create JSON", http.StatusInternalServerError)
		return
	}

	fmt.Printf("json: %v\n", coordsJSON)

	pythonURL := "http://localhost:9000/predict" // Python server URL
	resp, err := http.Post(pythonURL, "application/json", bytes.NewBuffer(coordsJSON))
	if err != nil {
		http.Error(w, "Error communicating with Python server", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		http.Error(w, "Error decoding Python server response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func inferenceVideoHandler(w http.ResponseWriter, r *http.Request) {
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

	var points Points
	if err := json.Unmarshal(body, &points); err != nil {
		http.Error(w, "Error parsing JSON body", http.StatusBadRequest)
		return
	}

	coordsJSON, err := json.Marshal(points)
	if err != nil {
		http.Error(w, "Failed to create JSON", http.StatusInternalServerError)
		return
	}

	pythonURL := "http://localhost:9000/predict-frames"
	resp, err := http.Post(pythonURL, "application/json", bytes.NewBuffer(coordsJSON))
	if err != nil {
		http.Error(w, "Error communicating with Python server", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		http.Error(w, "Error decoding Python server response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func main() {
	const port = 8080
	http.HandleFunc("/zawarudo.mp4", handlers.HandleMP4video)
	http.HandleFunc("/zawarudo/*", videoHandler)
	http.HandleFunc("/inference-frame", inferenceFrameHandler)
	http.HandleFunc("/inference-video", inferenceVideoHandler)

	http.Handle("/metrics", promhttp.Handler())

	fmt.Printf("Server running on %v\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", port), nil))
}
