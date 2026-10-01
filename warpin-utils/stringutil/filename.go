package stringutil

import (
	"fmt"
	"path/filepath"

	"github.com/google/uuid"
)

// GenerateUniqueFilename creates a new filename using a 36-character UUID
// and the extension from the original filename.
// Example: "my-photo.jpg" -> "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx.jpg"
func GenerateUniqueFilename(originalFilename string) string {
	// Get the extension from the original filename.
	extension := filepath.Ext(originalFilename)

	// Generate a new UUID (which is 36 characters long with hyphens).
	id := uuid.New().String()

	// Combine the UUID and the extension.
	return fmt.Sprintf("%s%s", id, extension)
}
