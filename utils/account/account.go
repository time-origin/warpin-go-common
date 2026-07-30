package account

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// GenerateUsernameFromEmail creates a username from an email address,
// appending a random 8-digit suffix to the email prefix to reduce collisions.
// For example, "test@example.com" might become "test_12345678".
func GenerateUsernameFromEmail(email string) string {
	// 1. Find the email prefix
	prefix := email
	if i := strings.Index(email, "@"); i != -1 {
		prefix = email[:i]
	}

	// 2. Generate a random 8-digit number
	// Use a new random source seeded with the current time to ensure non-deterministic results.
	source := rand.NewSource(time.Now().UnixNano())
	random := rand.New(source)
	suffix := random.Intn(90000000) + 10000000 // Range: 10,000,000 to 99,999,999

	// 3. Combine prefix and suffix
	return fmt.Sprintf("%s_%d", prefix, suffix)
}
