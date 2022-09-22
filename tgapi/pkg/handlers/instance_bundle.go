package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kurl-testgrid/tgapi/pkg/crypto"
	"github.com/replicatedhq/kurl-testgrid/tgapi/pkg/logger"
	"github.com/replicatedhq/kurl-testgrid/tgapi/pkg/persistence"
)

func InstanceBundle(passpharse string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		bucket := os.Getenv("SUPPORT_BUNDLE_BUCKET")
		if bucket == "" {
			w.WriteHeader(http.StatusNotImplemented)
			return
		}

		instanceID := mux.Vars(r)["instanceId"]

		filename, err := instanceBundleEncryptToDisk(r.Context(), r.Body, passpharse)
		if err != nil {
			logger.Error(errors.Errorf("Failed save encrypted file for instance %s: %v", instanceID, err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer os.RemoveAll(filename)

		key := fmt.Sprintf("%s-%d/bundle.tgz", instanceID, time.Now().Unix())
		// we dont need the request context anymore once the file has been stored on disk
		if err := instanceBundleUploadToS3(context.Background(), filename, bucket, key); err != nil {
			logger.Error(errors.Errorf("Failed to upload encrypted file go s3 for instance %s: %v", instanceID, err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		bundleURL := fmt.Sprintf("https://%s.s3.amazonaws.com/%s", bucket, key)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(bundleURL + "\n"))
		return
	}
}

func instanceBundleEncryptToDisk(ctx context.Context, source io.Reader, passphrase string) (string, error) {
	f, err := os.CreateTemp("", "testgrid-bundle")
	if err != nil {
		return "", errors.Wrap(err, "create temp file")
	}
	defer f.Close()

	encrypter, err := crypto.Encrypt(passphrase, f)
	if err != nil {
		_ = os.RemoveAll(f.Name())
		return "", errors.Wrap(err, "create new encrypter")
	}
	defer encrypter.Close()

	_, err = io.Copy(encrypter, source)
	if err != nil {
		_ = os.RemoveAll(f.Name())
		return "", errors.Wrap(err, "copy source to file")
	}
	return f.Name(), nil
}

func instanceBundleUploadToS3(ctx context.Context, filename, bucket, key string) error {
	f, err := os.Open(filename)
	if err != nil {
		return errors.Wrap(err, "open file")
	}
	defer f.Close()

	input := &s3manager.UploadInput{
		Body:   aws.ReadSeekCloser(f),
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	s3Uploader := persistence.GetS3Uploader()
	_, err = s3Uploader.UploadWithContext(ctx, input)
	return errors.Wrap(err, "upload to s3")
}
