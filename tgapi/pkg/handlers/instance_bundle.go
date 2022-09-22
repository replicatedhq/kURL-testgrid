package handlers

import (
	"context"
	"fmt"
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

		encrypted, err := crypto.StreamEncrypt(passpharse, r.Body)
		if err != nil {
			logger.Error(errors.Errorf("Failed to encrypt bundle for instance %s: %v", instanceID, err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer encrypted.Close()

		key := fmt.Sprintf("%s-%d/bundle.tgz", instanceID, time.Now().Unix())
		input := &s3manager.UploadInput{
			Body:   aws.ReadSeekCloser(encrypted),
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		}

		s3Uploader := persistence.GetS3Uploader()
		_, err = s3Uploader.UploadWithContext(context.Background(), input)
		if err != nil {
			logger.Error(errors.Errorf("Failed to upload bundle to s3 for instance %s: %v", instanceID, err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		bundleURL := fmt.Sprintf("https://%s.s3.amazonaws.com/%s", bucket, key)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(bundleURL + "\n"))
		return
	}
}
