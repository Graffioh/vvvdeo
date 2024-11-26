package storage

import (
	"archive/zip"
	"bytes"
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
	fmt.Println("Uploading frames...")

	buf, err := zipFramesInMemory(framesDir)
	if err != nil {
		return fmt.Errorf("failed to zip frames: %w", err)
	}

	zipKey := fmt.Sprintf("frames/%s/frames.zip", videoKey)

	client := GetS3Client()
	contentLength := int64(buf.Len())
	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String(zipKey),
		Body:          bytes.NewReader(buf.Bytes()),
		ContentLength: &contentLength,
	})
	if err != nil {
		return fmt.Errorf("failed to upload zip file: %w", err)
	}

	fmt.Println("Frames successfully uploaded!")
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
