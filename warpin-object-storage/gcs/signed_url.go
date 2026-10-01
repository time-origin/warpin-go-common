package gcs

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/storage"
)

// GenerateSignedURL creates a signed URL for a private object.
// The URL will be valid for 15 minutes and allows reading the object (GET).
// Note: This implementation uses the top-level storage.SignedURL function.
// The service account used to initialize the client must have the "iam.serviceAccountTokenCreator" role.
func (g *GoogleStorage) GenerateSignedURL(ctx context.Context, bucket, object string) (string, error) {
	// SignedURLOptions allows you to configure the signed URL.
	// The credentials used are implicitly taken from the client's initialization.
	opts := &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  "GET",
		Expires: time.Now().Add(15 * time.Minute),
	}

	// Correct usage: Call the top-level SignedURL function from the storage package.
	// It does not take a context as a parameter.
	u, err := storage.SignedURL(bucket, object, opts)
	if err != nil {
		return "", fmt.Errorf("storage.SignedURL: %w", err)
	}

	return u, nil
}
