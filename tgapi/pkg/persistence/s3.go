package persistence

import (
	"context"
	"sync"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var s3Client *s3.Client
var s3UploadManager *transfermanager.Client
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

func GetS3Uploader() *transfermanager.Client {
	s3Mu.Lock()
	defer s3Mu.Unlock()

	if s3UploadManager == nil {
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			panic(err)
		}
		s3UploadManager = transfermanager.New(s3.NewFromConfig(cfg))
	}
	return s3UploadManager
}
