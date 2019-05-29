package pubsub

import (
	"cloud.google.com/go/pubsub"
	"fmt"
	"os"
)

// Option is the function signature required to be considered an pubsub.Option.
type Option func(*Transport) error

const (
	DefaultProjectEnvKey      = "GOOGLE_CLOUD_PROJECT"
	DefaultTopicEnvKey        = "PUBSUB_TOPIC"
	DefaultSubscriptionEnvKey = "PUBSUB_SUBSCRIPTION"
)

// WithEncoding sets the encoding for pubsub transport.
func WithEncoding(encoding Encoding) Option {
	return func(t *Transport) error {
		t.Encoding = encoding
		return nil
	}
}

// WithClient sets the pubsub client for pubsub transport. Use this for explicit
// auth setup. Otherwise the env var 'GOOGLE_APPLICATION_CREDENTIALS' is used.
// See https://cloud.google.com/docs/authentication/production for more details.
func WithClient(client *pubsub.Client) Option {
	return func(t *Transport) error {
		t.client = client
		return nil
	}
}

// WithProjectID sets the project ID for pubsub transport.
func WithProjectID(projectID string) Option {
	return func(t *Transport) error {
		t.projectID = projectID
		return nil
	}
}

// WithProjectIDFromEnv sets the project ID for pubsub transport from a
// given environment variable name.
func WithProjectIDFromEnv(key string) Option {
	return func(t *Transport) error {
		v := os.Getenv(key)
		if v == "" {
			return fmt.Errorf("unable to load project id, %s environment variable not set", key)
		}
		t.projectID = v
		return nil
	}
}

// WithProjectIDFromDefaultEnv sets the project ID for pubsub transport from
// the environment variable named 'GOOGLE_CLOUD_PROJECT'.
func WithProjectIDFromDefaultEnv(key string) Option {
	return WithProjectIDFromEnv(DefaultProjectEnvKey)
}

// WithTopicID sets the topic ID for pubsub transport.
func WithTopicID(topicID string) Option {
	return func(t *Transport) error {
		t.topicID = topicID
		return nil
	}
}

// WithTopicIDFromEnv sets the topic ID for pubsub transport from a given
// environment variable name.
func WithTopicIDFromEnv(key string) Option {
	return func(t *Transport) error {
		v := os.Getenv(key)
		if v == "" {
			return fmt.Errorf("unable to load topic id, %s environment variable not set", key)
		}
		t.topicID = v
		return nil
	}
}

// WithTopicIDFromDefaultEnv sets the topic ID for pubsub transport from the
// environment variable named 'PUBSUB_TOPIC'.
func WithTopicIDFromDefaultEnv(key string) Option {
	return WithTopicIDFromEnv(DefaultTopicEnvKey)
}

// WithSubscriptionID sets the subscription ID for pubsub transport.
func WithSubscriptionID(subscriptionID string) Option {
	return func(t *Transport) error {
		t.subscriptionID = subscriptionID
		return nil
	}
}

// WithSubscriptionIDFromEnv sets the subscription ID for pubsub transport from
// a given environment variable name.
func WithSubscriptionIDFromEnv(key string) Option {
	return func(t *Transport) error {
		v := os.Getenv(key)
		if v == "" {
			return fmt.Errorf("unable to load subscription id, %s environment variable not set", key)
		}
		t.subscriptionID = v
		return nil
	}
}

// WithSubscriptionIDFromDefaultEnv sets the subscription ID for pubsub
// transport from the environment variable named 'PUBSUB_SUBSCRIPTION'.
func WithSubscriptionIDFromDefaultEnv(key string) Option {
	return WithSubscriptionIDFromEnv(DefaultSubscriptionEnvKey)
}
