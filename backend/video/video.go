package video

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"veedeo/events"
	"veedeo/storage"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func downloadVideo(bucket, key, localPath string) error {
	client := storage.GetS3Client()
	resp, err := client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	err = os.MkdirAll(filepath.Dir(localPath), os.ModePerm)
	if err != nil {
		fmt.Printf("Error creating directories: %v\n", err)
		return err
	}

	file, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}

func ensureMp4Extension(videoPath string) (string, error) {
	ext := filepath.Ext(videoPath)
	if ext == ".mp4" {
		return videoPath, nil
	}

	newPath := videoPath + ".mp4"
	err := os.Rename(videoPath, newPath)
	if err != nil {
		return "", fmt.Errorf("failed to rename file: %w", err)
	}

	return newPath, nil
}

func extractFrames(videoPath, framesDir string) error {
	err := os.MkdirAll(framesDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create frames directory: %w", err)
	}

	env := os.Getenv("APP_ENV")

	var cmd *exec.Cmd
	if env == "PROD" {
		cmd = exec.Command("ffmpeg",
			"-i", videoPath,
			"-q:v", "2",
			"-start_number", "0",
			fmt.Sprintf("%s/%%05d.jpg", framesDir),
		)
	} else {
		// select a frame every n so the development workflow is faster when inferencing
		n_of_jumped_frames := "select='not(mod(n, 5))'"
		cmd = exec.Command("ffmpeg",
			"-i", videoPath,
			"-vf", n_of_jumped_frames,
			"-vsync", "vfr",
			"-q:v", "2",
			fmt.Sprintf("%s/%%05d.jpg", framesDir),
		)

		/*
			cmd = exec.Command("ffmpeg",
				"-i", videoPath,
				"-q:v", "2",
				"-start_number", "0",
				fmt.Sprintf("%s/%%05d.jpg", framesDir),
			)
		*/
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("FFmpeg error: %w\nOutput: %s", err, string(output))
	}

	fmt.Println("Frame extracted!")
	return nil
}

func zipFramesInMemory(framesDir string) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)
	defer zipWriter.Close()

	err := filepath.Walk(framesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(framesDir, path)
		if err != nil {
			return err
		}

		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", path, err)
		}
		defer file.Close()

		writer, err := zipWriter.Create(relPath)
		if err != nil {
			return fmt.Errorf("failed to create entry for file %s: %w", path, err)
		}

		_, err = io.Copy(writer, file)
		if err != nil {
			return fmt.Errorf("failed to write file %s to zip: %w", path, err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking frames directory: %w", err)
	}

	return buf, nil
}

func uploadFrames(bucket, framesDir, videoKey string) error {
	buf, err := zipFramesInMemory(framesDir)
	if err != nil {
		return fmt.Errorf("failed to zip frames: %w", err)
	}

	zipKey := fmt.Sprintf("frames/%s.zip", videoKey)

	client := storage.GetS3Client()
	contentLength := int64(buf.Len())
	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(zipKey),
		Body:          bytes.NewReader(buf.Bytes()),
		ContentLength: &contentLength,
		ContentType:   aws.String("application/zip"),
	})

	if err != nil {
		return fmt.Errorf("failed to upload frames .zip: %w", err)
	}

	fmt.Println("Frames successfully uploaded!")
	return nil
}

func cleanDirectory(dirPath string) error {
	if err := os.RemoveAll(dirPath); err != nil {
		return err
	}

	fmt.Printf("Removed: %s\n", dirPath)
	return nil
}

func ProcessVideo(bucket, videoKey string) error {
	fmt.Println("Video processing started...")

	keyPath := strings.Split(videoKey, "/")[1]

	// clean videos and frames directories before starting
	if err := cleanDirectory("./tmp/videos"); err != nil {
		fmt.Printf("Error cleaning temporary video directory: %v\n", err)
	} else {
		fmt.Println("Temporary Video directory cleaned successfully.")
	}

	if err := cleanDirectory("./tmp/frames"); err != nil {
		fmt.Printf("Error cleaning temporary frames directory: %v\n", err)
	} else {
		fmt.Println("Temporary Frames directory cleaned successfully.")
	}

	videoPath := "./tmp/videos/" + keyPath
	framesPath := "./tmp/frames/" + keyPath

	err := downloadVideo(bucket, videoKey, videoPath)
	if err != nil {
		return fmt.Errorf("failed to download video: %w", err)
	}

	updatedVideoPath, err := ensureMp4Extension(videoPath)
	if err != nil {
		return fmt.Errorf("failed to ensure MP4 extension: %w", err)
	}

	err = extractFrames(updatedVideoPath, framesPath)
	if err != nil {
		return fmt.Errorf("failed to extract frames: %w", err)
	}

	err = uploadFrames(bucket, framesPath, keyPath)
	if err != nil {
		return fmt.Errorf("failed to upload frames: %w", err)
	}

	fmt.Println("Video processing complete")
	return nil
}

func VideoSpeedupHandler(w http.ResponseWriter, r *http.Request) {
	// get video form
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

	// setup temp directories
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

	beforePart := filepath.Join(tempDir, "before.mp4")
	afterPart := filepath.Join(tempDir, "after.mp4")
	speedupPart := filepath.Join(tempDir, "speedup.mp4")
	finalFile := filepath.Join(tempDir, "final.mp4")

	// get start time - end time - speedup factor from the request
	startTime := r.FormValue("startTime")
	endTime := r.FormValue("endTime")
	speedupFactorStr := r.FormValue("speedupFactor")
	speedupFactor, err := strconv.ParseFloat(speedupFactorStr, 64)
	if err != nil {
		fmt.Println("Error parsing speedupFactor:", err)
		http.Error(w, "Invalid speedupFactor value. Please provide a valid number.", http.StatusBadRequest)
		return
	}

	// update 1
	events.SseManager.Update("0%")

	// part 1: cut the video before the interested segment
	cmd1 := exec.Command("ffmpeg", "-y", "-to", startTime, "-i", tempFile.Name(), "-filter_complex", "[0:v]setpts=PTS-STARTPTS[v];[0:a]aresample=async=1:first_pts=0[a]", "-map", "[v]", "-map", "[a]", "-f", "mp4", beforePart)
	output1, err := cmd1.CombinedOutput()
	if err != nil {
		fmt.Println("FFmpeg Error (cut before):", err)
		fmt.Println("FFmpeg Output:", string(output1))
		http.Error(w, "Failed to cut video before segment", http.StatusInternalServerError)
		return
	}

	// update 2
	events.SseManager.Update("30%")

	// part 2: cut the video after the interested segment
	cmd2 := exec.Command("ffmpeg", "-y", "-ss", endTime, "-i", tempFile.Name(), "-filter_complex", "[0:v]setpts=PTS-STARTPTS[v];[0:a]aresample=async=1:first_pts=0[a]", "-map", "[v]", "-map", "[a]", "-f", "mp4", afterPart)
	output2, err := cmd2.CombinedOutput()
	if err != nil {
		fmt.Println("FFmpeg Error (cut after):", err)
		fmt.Println("FFmpeg Output:", string(output2))
		http.Error(w, "Failed to cut video after segment", http.StatusInternalServerError)
		return
	}

	// update 3
	events.SseManager.Update("60%")

	// part 3: speed up the trimmed part
	setptsMultiplier := 1 / speedupFactor
	speedupFilter := fmt.Sprintf("[0:v]setpts=PTS-STARTPTS,setpts=%f*PTS[v];[0:a]atempo=%f[a]", setptsMultiplier, speedupFactor)
	cmd3 := exec.Command("ffmpeg", "-y", "-ss", startTime, "-to", endTime, "-i", tempFile.Name(),
		"-filter_complex", speedupFilter, "-map", "[v]", "-map", "[a]", "-f", "mp4", speedupPart)
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

	// update 4
	events.SseManager.Update("80%")

	// part 4: replace the trimmed part in the original video
	cmd4 := exec.Command("ffmpeg", "-f", "concat", "-safe", "0", "-i", concatFile, "-c", "copy", finalFile)
	output4, err := cmd4.CombinedOutput()
	if err != nil {
		fmt.Println("FFmpeg Error (concatenation):", err)
		fmt.Println("FFmpeg Output:", string(output4))
		http.Error(w, "Failed to concatenate video", http.StatusInternalServerError)
		return
	}

	// update final
	events.SseManager.Update("100%")

	// send the video to the frontend
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

type VideoCoordinates struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
}

type Points struct {
	Coordinates []VideoCoordinates `json:"coordinates"`
	Labels      []int32            `json:"labels"`
}

func VideoInferenceHandler(w http.ResponseWriter, r *http.Request) {
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
