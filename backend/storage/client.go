package storage

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var (
	s3Client     *s3.Client
	initS3Client sync.Once
)

func GetS3Client() *s3.Client {
	initS3Client.Do(func() {
		accountId := os.Getenv("CLOUDFLARE_ACCOUNT_ID")
		accessKeyId := os.Getenv("R2_ACCESS_KEY")
		accessKeySecret := os.Getenv("R2_SECRET_ACCESS_KEY")

		cfg, err := config.LoadDefaultConfig(context.TODO(),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyId, accessKeySecret, "")),
			config.WithRegion("auto"),
		)
		if err != nil {
			log.Fatalf("Failed to load S3 configuration: %v", err)
		}

		s3Client = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountId))
		})
	})
	return s3Client
}
