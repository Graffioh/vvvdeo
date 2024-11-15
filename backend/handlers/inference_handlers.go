package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

type VideoCoordinates struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
}

type Points struct {
	Coordinates []VideoCoordinates `json:"coordinates"`
	Labels      []int32            `json:"labels"`
}

func InferenceFrameHandler(w http.ResponseWriter, r *http.Request) {
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

func InferenceVideoHandler(w http.ResponseWriter, r *http.Request) {
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
