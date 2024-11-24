package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
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

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file!")
	}
	var accountId = os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	var bucketName = os.Getenv("R2_BUCKET")
	var accessKeyId = os.Getenv("R2_ACCESS_KEY")
	var accessKeySecret = os.Getenv("R2_SECRET_ACCESS_KEY")

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyId, accessKeySecret, "")),
		config.WithRegion("auto"),
	)
	if err != nil {
		log.Fatal(err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountId))
	})

	presignClient := s3.NewPresignClient(client)
	uuid := uuid.New()
	key := fmt.Sprintf("videos/%s", "video-"+uuid.String())

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

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file!")
	}
	var accountId = os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	var bucketName = os.Getenv("R2_BUCKET")
	var accessKeyId = os.Getenv("R2_ACCESS_KEY")
	var accessKeySecret = os.Getenv("R2_SECRET_ACCESS_KEY")

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyId, accessKeySecret, "")),
		config.WithRegion("auto"),
	)
	if err != nil {
		log.Fatal(err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountId))
	})

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
