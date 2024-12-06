package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

type VideoCoordinates struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
}

type Points struct {
	Coordinates []VideoCoordinates `json:"coordinates"`
	Labels      []int32            `json:"labels"`
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

	imageFile, imageFileHeader, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}
	defer imageFile.Close()

	segmentationData := r.FormValue("segmentationData")
	if segmentationData == "" {
		http.Error(w, "No segmentationData as JSON data provided", http.StatusBadRequest)
		return
	}

	videoKey := r.FormValue("videoKey")
	key := strings.Split(videoKey, "/")[1]

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	segmentationPart, err := writer.CreateFormField("segmentationData")
	if err != nil {
		http.Error(w, "Error creating form field for JSON", http.StatusInternalServerError)
		return
	}
	_, err = segmentationPart.Write([]byte(segmentationData))
	if err != nil {
		http.Error(w, "Error writing JSON to form field", http.StatusInternalServerError)
		return
	}

	imagePart, err := writer.CreateFormFile("image", imageFileHeader.Filename)
	if err != nil {
		http.Error(w, "Error creating form file for image", http.StatusInternalServerError)
		return
	}
	_, err = io.Copy(imagePart, imageFile)
	if err != nil {
		http.Error(w, "Error writing file to form file", http.StatusInternalServerError)
		return
	}

	fileKeyPart, err := writer.CreateFormField("fileKey")
	if err != nil {
		http.Error(w, "Error creating form field for fileKey", http.StatusInternalServerError)
		return
	}
	_, err = fileKeyPart.Write([]byte(key))
	if err != nil {
		http.Error(w, "Error writing fileKey to form field", http.StatusInternalServerError)
		return
	}
	writer.Close()

	pythonURL := "http://localhost:9000/segment"
	req, err := http.NewRequest("POST", pythonURL, body)
	if err != nil {
		http.Error(w, "Error creating Python server request", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Error communicating with Python server", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	if contentType == "application/json" {
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
	w.Header().Set("Content-Disposition", "attachment; filename=crafted_vvvdeo.mp4")

	_, err = io.Copy(w, resp.Body)
	if err != nil {
		http.Error(w, "Error streaming video file", http.StatusInternalServerError)
		return
	}
}
