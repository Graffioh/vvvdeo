package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func downloadVideo(bucket, key, localPath string) error {
	fmt.Println("Downloading video...")

	client := GetS3Client()
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
	fmt.Println("Extracting frames...")

	err := os.MkdirAll(framesDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create frames directory: %w", err)
	}

	fmt.Println(videoPath)

	cmd := exec.Command("ffmpeg",
		"-i", videoPath,
		"-q:v", "2",
		"-start_number", "0",
		fmt.Sprintf("%s/frame_%%05d.jpg", framesDir),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("FFmpeg error: %w\nOutput: %s", err, string(output))
	}

	return nil
}

func uploadFrames(bucket, framesDir, videoKey string) error {
	fmt.Println("Uploading frames...")

	files, err := os.ReadDir(framesDir)
	if err != nil {
		return fmt.Errorf("failed to read frames directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		framePath := filepath.Join(framesDir, file.Name())
		frameKey := fmt.Sprintf("frames/%s/%s", videoKey, file.Name())

		frameFile, err := os.Open(framePath)
		if err != nil {
			return fmt.Errorf("failed to open frame file: %w", err)
		}
		defer frameFile.Close()

		client := GetS3Client()
		_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(frameKey),
			Body:   frameFile,
		})
		if err != nil {
			return fmt.Errorf("failed to upload frame: %w", err)
		}
	}

	return nil
}

func ProcessVideo(bucket, videoKey string) error {
	fmt.Println("Video processing started...")

	keyPath := strings.Split(videoKey, "/")[1]

	localVideoPath := "./tmp/videos/" + keyPath
	framesDir := "./tmp/frames/" + keyPath

	err := downloadVideo(bucket, videoKey, localVideoPath)
	if err != nil {
		return fmt.Errorf("failed to download video: %w", err)
	}

	updatedVideoPath, err := ensureMp4Extension(localVideoPath)
	if err != nil {
		return fmt.Errorf("failed to ensure MP4 extension: %w", err)
	}

	err = extractFrames(updatedVideoPath, framesDir)
	if err != nil {
		return fmt.Errorf("failed to extract frames: %w", err)
	}

	err = uploadFrames(bucket, framesDir, keyPath)
	if err != nil {
		return fmt.Errorf("failed to upload frames: %w", err)
	}

	fmt.Println("Video processing complete")
	return nil
}
