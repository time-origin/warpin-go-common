package stringutil

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"time"
)

const (
	// SecretAlphabet defines the set of characters to be used in generated secrets.
	SecretAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
)

// GenerateSecret creates a cryptographically secure random string of a given length.
func GenerateSecret(length int) (string, error) {
	secretBytes := make([]byte, length)
	alphabetLength := big.NewInt(int64(len(SecretAlphabet)))

	for i := 0; i < length; i++ {
		randomIndex, err := rand.Int(rand.Reader, alphabetLength)
		if err != nil {
			return "", fmt.Errorf("failed to generate random index: %w", err)
		}
		secretBytes[i] = SecretAlphabet[randomIndex.Int64()]
	}

	return string(secretBytes), nil
}

// GenerateTinyVerseSecretKey creates a secret key with the specific format "tv-sk-YYYY$RANDOM_PART".
func GenerateTinyVerseSecretKey() (string, error) {
	// 1. Define the length of the random part.
	const randomPartLength = 28

	// 2. Generate the random part using the existing utility.
	randomPart, err := GenerateSecret(randomPartLength)
	if err != nil {
		return "", fmt.Errorf("failed to generate random part of the key: %w", err)
	}

	// 3. Get the current year.
	year := time.Now().UTC().Year()

	// 4. Assemble the final key in the specified format.
	finalKey := fmt.Sprintf("tv-sk-%d$%s", year, randomPart)

	return finalKey, nil
}

// PrintNewTinyVerseSecretKey is a helper function that generates a new key
// in the "tv-sk" format and prints it directly to the console.
func PrintNewTinyVerseSecretKey() {
	secret, err := GenerateTinyVerseSecretKey()
	if err != nil {
		log.Fatalf("FATAL: Could not generate TinyVerse secret key: %v", err)
	}
	fmt.Println("--- Generated TinyVerse Secret Key ---")
	fmt.Println(secret)
	fmt.Println("------------------------------------")
}
