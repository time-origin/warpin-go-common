package gcs

import (
	"context"
	"fmt"
	"io"
	"time"

	"cloud.google.com/go/storage"
)

// Upload uploads an object to Google Cloud Storage.
// If isPublic is true, the object will be publicly readable.
func (g *GoogleStorage) Upload(ctx context.Context, bucket, object string, file io.Reader, isPublic bool) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*50)
	defer cancel()

	o := g.client.Bucket(bucket).Object(object)

	// Optional: set a generation-match precondition to avoid potential race
	// conditions and data corruptions.
	o = o.If(storage.Conditions{DoesNotExist: true})

	// Upload an object with storage.Writer.
	wc := o.NewWriter(ctx)

	// If we want the object to be public, we set the PredefinedACL.
	if isPublic {
		wc.PredefinedACL = "publicRead"
	}

	if _, err := io.Copy(wc, file); err != nil {
		return "", fmt.Errorf("io.Copy: %w", err)
	}
	if err := wc.Close(); err != nil {
		return "", fmt.Errorf("Writer.Close: %w", err)
	}

	// Return the public URL of the object.
	// This URL is only accessible if the object is public.
	return fmt.Sprintf("https://storage.googleapis.com/%s/%s", bucket, object), nil
}
