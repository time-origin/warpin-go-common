package gcs

import (
	"context"
	"fmt"
	"io"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

// Storage defines the interface for cloud storage operations.
type Storage interface {
	// Upload uploads a file and specifies if it should be publicly accessible.
	Upload(ctx context.Context, bucket, object string, file io.Reader, isPublic bool) (string, error)
	// GenerateSignedURL creates a temporary URL to access a private object.
	GenerateSignedURL(ctx context.Context, bucket, object string) (string, error)
}

// GoogleStorage holds the client and configuration for Google Cloud Storage.
type GoogleStorage struct {
	client *storage.Client
}

// New creates a new GoogleStorage client.
// If credentialsFile is an empty string, it uses Application Default Credentials.
// Otherwise, it uses the provided service account JSON file.
func New(ctx context.Context, credentialsFile string) (*GoogleStorage, error) {
	var opts []option.ClientOption
	if credentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(credentialsFile))
	}

	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("storage.NewClient: %w", err)
	}
	return &GoogleStorage{client: client}, nil
}

// Close closes the underlying storage client.
func (g *GoogleStorage) Close() error {
	return g.client.Close()
}
