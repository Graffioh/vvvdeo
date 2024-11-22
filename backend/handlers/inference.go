package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

type VideoCoordinates struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
}

type Points struct {
	Coordinates []VideoCoordinates `json:"coordinates"`
	Labels      []int32            `json:"labels"`
}

/*
func InferenceFrameHandler(w http.ResponseWriter, r *http.Request) {
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

	pythonURL := "http://localhost:9000/predict"
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
*/

func uploadImage(w http.ResponseWriter, file multipart.File, fileHeader *multipart.FileHeader) {
	imgPath := filepath.Join("../sam2seg/img", fileHeader.Filename)
	dst, err := os.Create(imgPath)
	if err != nil {
		http.Error(w, "Error creating file", http.StatusInternalServerError)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, "Error saving file in img dir", http.StatusInternalServerError)
	}
}

func InferenceVideoHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Error parsing multipart form", http.StatusBadRequest)
		return
	}

	file, fileHeader, err := r.FormFile("image")
	var imageName string = ""
	if file != nil {
		uploadImage(w, file, fileHeader)
		imageName = fileHeader.Filename
		defer file.Close()
	}

	jsonData := r.FormValue("data")
	if jsonData == "" {
		http.Error(w, "No JSON data provided", http.StatusBadRequest)
		return
	}

	var points Points
	if err := json.Unmarshal([]byte(jsonData), &points); err != nil {
		http.Error(w, "Error parsing JSON data", http.StatusBadRequest)
		return
	}

	coordsJSON, err := json.Marshal(points)
	if err != nil {
		http.Error(w, "Failed to create JSON", http.StatusInternalServerError)
		return
	}

	videoName := r.FormValue("video_name")

	pythonURL := fmt.Sprintf("http://localhost:9000/predict-frames?image=%s&video_name=%s", imageName, videoName)
	resp, err := http.Post(pythonURL, "application/json", bytes.NewBuffer(coordsJSON))
	if err != nil {
		http.Error(w, "Error communicating with Python server", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")

	if contentType == "application/json" {
		// Handle error response
		var result map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			http.Error(w, "Error decoding Python server response", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
		return
	}

	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Content-Disposition", "attachment; filename=processed_video.mp4")

	_, err = io.Copy(w, resp.Body)
	if err != nil {
		http.Error(w, "Error streaming video file", http.StatusInternalServerError)
		return
	}
}
