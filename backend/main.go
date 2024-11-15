package main

import (
	"fmt"
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
	} else if strings.HasSuffix(path, ".mp4") {
		handlers.HandleMP4video(w, r)
	} else {
		http.NotFound(w, r)
	}
}

func main() {
	const port = 8080
	http.HandleFunc("/zawarudo/*", videoHandler)
	http.HandleFunc("/inference-frame", handlers.InferenceFrameHandler)
	http.HandleFunc("/inference-video", handlers.InferenceVideoHandler)

	http.Handle("/metrics", promhttp.Handler())

	fmt.Printf("Server running on %v\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", port), nil))
}
