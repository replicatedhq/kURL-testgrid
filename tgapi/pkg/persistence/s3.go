package persistence

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var s3Client *s3.Client
var s3UploadManager *manager.Uploader
var s3Mu sync.Mutex

func GetS3Client() *s3.Client {
	s3Mu.Lock()
	defer s3Mu.Unlock()

	if s3Client == nil {
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			panic(err)
		}
		s3Client = s3.NewFromConfig(cfg)
	}
	return s3Client
}

func GetS3Uploader() *manager.Uploader {
	s3Mu.Lock()
	defer s3Mu.Unlock()

	if s3UploadManager == nil {
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			panic(err)
		}
		s3UploadManager = manager.NewUploader(s3.NewFromConfig(cfg))
	}
	return s3UploadManager
}
