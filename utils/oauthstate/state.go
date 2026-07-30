package oauthstate

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/gorilla/sessions"
	"net/http"
	"strings"
)

const stateKey = "oauth_state"
const sessionName = "auth-session"

// GenerateRandomState generates a secure random state parameter.
func GenerateRandomState() (string, error) {
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	state := base64.URLEncoding.EncodeToString(randomBytes)
	state = trimPadding(state)
	return state, nil
}

// CreateAndStoreState generates a random state string and stores it in the user's session.
func CreateAndStoreState(store sessions.Store, w http.ResponseWriter, r *http.Request) (string, error) {
	state, err := GenerateRandomState()
	if err != nil {
		return "", err
	}
	session, err := store.Get(r, sessionName)
	if err != nil {
		return "", fmt.Errorf("failed to get session: %w", err)
	}
	session.Values[stateKey] = state
	if err := session.Save(r, w); err != nil {
		return "", fmt.Errorf("failed to save session: %w", err)
	}
	return state, nil
}

// ValidateState retrieves the expected state from the session, compares it with the received state, and deletes it.
func ValidateState(store sessions.Store, w http.ResponseWriter, r *http.Request, receivedState string) error {
	session, err := store.Get(r, sessionName)
	if err != nil {
		return fmt.Errorf("session store error or session not found: %w", err)
	}

	expectedStateValue, ok := session.Values[stateKey]
	if !ok {
		return fmt.Errorf("state not found in session or session expired")
	}

	expectedState, ok := expectedStateValue.(string)
	if !ok || expectedState == "" {
		return fmt.Errorf("stored state is invalid type or empty")
	}

	if receivedState != expectedState {
		return fmt.Errorf("state mismatch")
	}

	delete(session.Values, stateKey)

	if err := session.Save(r, w); err != nil {
		return fmt.Errorf("failed to save session after state deletion: %w", err)
	}

	return nil
}

// trimPadding removes the padding characters from a base64 encoded string.
func trimPadding(s string) string {
	return strings.TrimRight(s, "=")
}
