package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

// In-memory storage for state tokens. In production, use Redis or signed JWTs.
// The state only needs to live for ~60 seconds (the OAuth round-trip time).
var (
	stateMu    sync.Mutex
	stateStore = make(map[string]time.Time)
)

// generateState creates a cryptographically secure random string for CSRF protection.
func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// cleanupExpiredStates removes states older than 2 minutes to prevent memory leaks.
func cleanupExpiredStates() {
	stateMu.Lock()
	defer stateMu.Unlock()

	cutoff := time.Now().Add(-2 * time.Minute)
	for state, createdAt := range stateStore {
		if createdAt.Before(cutoff) {
			delete(stateStore, state)
		}
	}
}

// LoginHandler redirects the user to Spotify's authorization page.
// This is where the OAuth flow begins.
func LoginHandler(oauthConfig *oauth2.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Generate a random state token to protect against CSRF attacks
		state, err := generateState()
		if err != nil {
			http.Error(w, "Failed to generate state", http.StatusInternalServerError)
			return
		}

		// Store the state with its creation time so we can validate it in the callback
		stateMu.Lock()
		stateStore[state] = time.Now()
		stateMu.Unlock()

		// Clean up old states to prevent memory leaks
		go cleanupExpiredStates()

		// Build the Spotify authorization URL with our parameters
		// AuthCodeURL adds client_id, redirect_uri, scope, and state to the URL
		authURL := oauthConfig.AuthCodeURL(state)

		// Redirect the user's browser to Spotify's login page
		http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
	}
}

// CallbackHandler receives the authorization code from Spotify and exchanges it for tokens.
// This is the redirect_uri endpoint that Spotify sends the user back to.
func CallbackHandler(oauthConfig *oauth2.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract the state and code from the query parameters
		state := r.URL.Query().Get("state")
		code := r.URL.Query().Get("code")
		errorParam := r.URL.Query().Get("error")

		// If Spotify returned an error (e.g., user denied permission)
		if errorParam != "" {
			http.Error(w, fmt.Sprintf("Spotify authorization failed: %s", errorParam), http.StatusBadRequest)
			return
		}

		// Verify the state token matches what we stored (CSRF protection)
		stateMu.Lock()
		createdAt, exists := stateStore[state]
		if exists {
			delete(stateStore, state) // Use the state only once
		}
		stateMu.Unlock()

		if !exists {
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			return
		}

		// Ensure the state isn't too old (should be used within 2 minutes)
		if time.Since(createdAt) > 2*time.Minute {
			http.Error(w, "State token expired", http.StatusBadRequest)
			return
		}

		// Exchange the authorization code for an access token
		// This makes a POST request to Spotify's /api/token endpoint
		token, err := oauthConfig.Exchange(r.Context(), code)
		if err != nil {
			http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
			return
		}

		// Store the access token in a secure HTTP-only cookie
		// This prevents JavaScript from accessing it (XSS protection)
		http.SetCookie(w, &http.Cookie{
			Name:     "spotify_access_token",
			Value:    token.AccessToken,
			Path:     "/",
			HttpOnly: true,                 // Prevent JavaScript access
			Secure:   false,                // Set to false for local dev (no HTTPS on localhost)
			SameSite: http.SameSiteLaxMode, // CSRF protection
			Expires:  token.Expiry,         // Cookie expires when token expires (~1 hour)
		})

		// Store the refresh token in a separate cookie
		// The refresh token is used to get new access tokens when they expire
		if token.RefreshToken != "" {
			http.SetCookie(w, &http.Cookie{
				Name:     "spotify_refresh_token",
				Value:    token.RefreshToken,
				Path:     "/",
				HttpOnly: true,
				Secure:   false,                 // Set to false for local dev
				SameSite: http.SameSiteLaxMode,
				MaxAge:   60 * 60 * 24 * 30, // 30 days
			})
		}

		// Redirect to your application's main page
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	}
}
