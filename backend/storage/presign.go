package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

type PresignedURLResponse struct {
	Key          string `json:"key"`
	PresignedURL string `json:"presignedUrl"`
}

func PresignedPutURLHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var bucketName = os.Getenv("R2_BUCKET")

	client := GetS3Client()
	presignClient := s3.NewPresignClient(client)

	uuid := uuid.New()
	key := fmt.Sprintf("videos/%s", "video-"+uuid.String()+"-"+os.Getenv("APP_ENV"))

	presignResult, err := presignClient.PresignPutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	})

	if err != nil {
		panic("Couldn't get presigned URL for PutObject")
	}

	response := PresignedURLResponse{
		Key:          key,
		PresignedURL: presignResult.URL,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func PresignedGetURLHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	var bucketName = os.Getenv("R2_BUCKET")

	client := GetS3Client()
	presignClient := s3.NewPresignClient(client)

	key := r.URL.Query().Get("key")

	presignResult, err := presignClient.PresignGetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(key),
	})

	if err != nil {
		panic("Couldn't get presigned URL for GetObject")
	}

	response := PresignedURLResponse{
		Key:          key,
		PresignedURL: presignResult.URL,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
