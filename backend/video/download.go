package video

type VideoRequest struct {
	URL string `json:"url"`
}

/*
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
*/

/*
func VideoStreamYTHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var videoReq VideoRequest
	if err := json.Unmarshal(body, &videoReq); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	if videoReq.URL == "" {
		http.Error(w, "Missing video URL", http.StatusBadRequest)
		return
	}

	cmd := exec.Command("yt-dlp", "-o", "-", "-f", "best", videoReq.URL)

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		log.Printf("StderrPipe error: %v", err)
		http.Error(w, "Failed to initialize stderr stream", http.StatusInternalServerError)
		return
	}

	cmdOutput, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("StdoutPipe error: %v", err)
		http.Error(w, "Failed to initialize video stream", http.StatusInternalServerError)
		return
	}

	log.Println("Pipes initialized successfully.")

	if err := cmd.Start(); err != nil {
		log.Printf("Command start error: %v", err)
		http.Error(w, "Failed to start video stream", http.StatusInternalServerError)
		return
	}

	// Log stderr in a goroutine
	go func() {
		stderrData, _ := io.ReadAll(stderrPipe)
		log.Printf("yt-dlp stderr: %s", string(stderrData))
	}()

	if _, err := io.Copy(w, cmdOutput); err != nil {
		log.Printf("Streaming error: %v", err)
		http.Error(w, "Failed to stream video", http.StatusInternalServerError)
		return
	}

	log.Println("Video stream completed successfully.")

	if err := cmd.Wait(); err != nil {
		log.Printf("Command wait error: %v", err)
		http.Error(w, "Error during video streaming", http.StatusInternalServerError)
		return
	}

	log.Println("Command finished successfully.")
}
*/
