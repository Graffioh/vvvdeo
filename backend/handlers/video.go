package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

func VideoSpeedupHandler(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 500*1024*1024)

	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, "Error parsing form or file too large.", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("videoFile")
	if err != nil {
		http.Error(w, "Error retrieving video file.", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if filepath.Ext(header.Filename) != ".mp4" {
		http.Error(w, "Invalid file type. Only .mp4 allowed.", http.StatusBadRequest)
		return
	}

	tempDir, err := os.MkdirTemp("", "videouploads")
	if err != nil {
		http.Error(w, "Failed to create temporary directory", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tempDir)

	tempFile, err := os.CreateTemp(tempDir, "video-*.mp4")
	if err != nil {
		http.Error(w, "Failed to create temporary file", http.StatusInternalServerError)
		return
	}
	defer tempFile.Close()

	_, err = io.Copy(tempFile, file)
	if err != nil {
		http.Error(w, "Failed to save video file", http.StatusInternalServerError)
		return
	}

	startTime := r.FormValue("startTime")
	endTime := r.FormValue("endTime")

	beforePart := filepath.Join(tempDir, "before.mp4")
	afterPart := filepath.Join(tempDir, "after.mp4")
	speedupPart := filepath.Join(tempDir, "speedup.mp4")
	finalFile := filepath.Join(tempDir, "final.mp4")

	cmd1 := exec.Command("ffmpeg", "-y", "-to", startTime, "-i", tempFile.Name(), "-filter_complex", "[0:v]setpts=PTS-STARTPTS[v];[0:a]aresample=async=1:first_pts=0[a]", "-map", "[v]", "-map", "[a]", "-f", "mp4", beforePart)
	output1, err := cmd1.CombinedOutput()
	if err != nil {
		fmt.Println("FFmpeg Error (cut before):", err)
		fmt.Println("FFmpeg Output:", string(output1))
		http.Error(w, "Failed to cut video before segment", http.StatusInternalServerError)
		return
	}

	cmd2 := exec.Command("ffmpeg", "-y", "-ss", endTime, "-i", tempFile.Name(), "-filter_complex", "[0:v]setpts=PTS-STARTPTS[v];[0:a]aresample=async=1:first_pts=0[a]", "-map", "[v]", "-map", "[a]", "-f", "mp4", afterPart)
	output2, err := cmd2.CombinedOutput()
	if err != nil {
		fmt.Println("FFmpeg Error (cut after):", err)
		fmt.Println("FFmpeg Output:", string(output2))
		http.Error(w, "Failed to cut video after segment", http.StatusInternalServerError)
		return
	}

	cmd3 := exec.Command("ffmpeg", "-y", "-ss", startTime, "-to", endTime, "-i", tempFile.Name(),
		"-filter_complex", "[0:v]setpts=PTS-STARTPTS,setpts=0.5*PTS[v];[0:a]atempo=2.0[a]", "-map", "[v]", "-map", "[a]", "-f", "mp4", speedupPart)
	output3, err := cmd3.CombinedOutput()
	if err != nil {
		fmt.Println("FFmpeg Error (speedup):", err)
		fmt.Println("FFmpeg Output:", string(output3))
		http.Error(w, "Failed to speed up video segment", http.StatusInternalServerError)
		return
	}

	concatFile := filepath.Join(tempDir, "concat.txt")
	concatContent := fmt.Sprintf("file '%s'\nfile '%s'\nfile '%s'\n", beforePart, speedupPart, afterPart)

	err = os.WriteFile(concatFile, []byte(concatContent), 0644)
	if err != nil {
		http.Error(w, "Failed to prepare concatenation list", http.StatusInternalServerError)
		return
	}

	cmd4 := exec.Command("ffmpeg", "-f", "concat", "-safe", "0", "-i", concatFile, "-c", "copy", finalFile)
	output4, err := cmd4.CombinedOutput()
	if err != nil {
		fmt.Println("FFmpeg Error (concatenation):", err)
		fmt.Println("FFmpeg Output:", string(output4))
		http.Error(w, "Failed to concatenate video", http.StatusInternalServerError)
		return
	}

	fmt.Println("FFmpeg command:", cmd4.String())
	fmt.Println("FFmpeg output:", string(output4))
	fmt.Println("Error:", err)

	outFile, err := os.Open(finalFile)
	if err != nil {
		http.Error(w, "Failed to open output file", http.StatusInternalServerError)
		return
	}
	defer outFile.Close()

	w.Header().Set("Content-Type", "video/mp4")
	w.Header().Set("Content-Disposition", "attachment; filename=processed-video.mp4")

	_, err = io.Copy(w, outFile)
	if err != nil {
		http.Error(w, "Failed to send processed video", http.StatusInternalServerError)
		return
	}
}
