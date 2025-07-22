package persistence

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/DataDog/datadog-go/v5/statsd"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
	"github.com/pkg/errors"
)

var (
	StatsdClientNotInitialized = errors.New("statsd client not initialized")

	statsdClient *statsd.Client
	statsdMu     sync.Mutex
)

func InitStatsd(port, namespace string) (*statsd.Client, error) {
	statsdMu.Lock()
	defer statsdMu.Unlock()

	if statsdClient == nil {
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			return nil, errors.Wrap(err, "failed to init AWS config")
		} else {
			ip, err := getInstancePrivateIP(cfg)
			if err != nil {
				return nil, errors.Wrap(err, "failed to find instance ip")
			}
			c, err := statsd.New(fmt.Sprintf("%s:%s", ip, port),
				// prefix every metric with the app name
				statsd.WithNamespace(namespace),
			)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to init statsd client targeting ip %s", ip)
			}
			statsdClient = c
		}
	}
	return statsdClient, nil
}

func Statsd() *statsd.Client {
	return statsdClient
}

func MaybeSendStatsdTiming(name string, value time.Duration, tags []string, rate float64) error {
	client := Statsd()
	if client == nil {
		return StatsdClientNotInitialized
	}
	return client.Timing(name, value, tags, rate)
}

func MaybeSendStatsdGauge(name string, value float64, tags []string, rate float64) error {
	client := Statsd()
	if client == nil {
		return StatsdClientNotInitialized
	}
	return client.Gauge(name, value, tags, rate)
}

func getInstancePrivateIP(cfg aws.Config) (string, error) {
	imdsClient := imds.NewFromConfig(cfg)
	result, err := imdsClient.GetInstanceIdentityDocument(context.TODO(), &imds.GetInstanceIdentityDocumentInput{})
	if err != nil {
		return "", err
	}
	return result.PrivateIP, nil
}
