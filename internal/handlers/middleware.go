package handlers

import (
	"context"
	"log"
	"net/http"
	"time"

	"golang.org/x/oauth2"
)

type contextKey string

const AccessTokenKey contextKey = "access_token"

// RequireAuth is a middleware that ensures the user has a valid access token.
// If the token is expired but a refresh token exists, it automatically refreshes.
// If no valid token can be obtained, it redirects to /login.
func RequireAuth(oauthConfig *oauth2.Config) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Try to get the access token from cookies
			accessCookie, err := r.Cookie("spotify_access_token")
			if err != nil {
				// No access token - redirect to login
				log.Println("No access token found, redirecting to login")
				http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
				return
			}

			// Check if the access token is expired or will expire soon (within 5 minutes)
			expiryCookie, err := r.Cookie("spotify_token_expiry")
			needsRefresh := false

			if err != nil {
				// No expiry info - assume it might need refresh
				needsRefresh = true
			} else {
				expiry, err := time.Parse(time.RFC3339, expiryCookie.Value)
				if err != nil || time.Until(expiry) < 5*time.Minute {
					needsRefresh = true
				}
			}

			// If token needs refresh, try to refresh it
			if needsRefresh {
				refreshCookie, err := r.Cookie("spotify_refresh_token")
				if err != nil {
					// No refresh token - redirect to login
					log.Println("Token expired and no refresh token, redirecting to login")
					http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
					return
				}

				// Use the refresh token to get a new access token
				token := &oauth2.Token{
					RefreshToken: refreshCookie.Value,
				}

				// TokenSource automatically refreshes the token
				tokenSource := oauthConfig.TokenSource(r.Context(), token)
				newToken, err := tokenSource.Token()
				if err != nil {
					log.Printf("Failed to refresh token: %v", err)
					http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
					return
				}

				// Update the access token cookie with the new token
				http.SetCookie(w, &http.Cookie{
					Name:     "spotify_access_token",
					Value:    newToken.AccessToken,
					Path:     "/",
					HttpOnly: true,
					Secure:   false, // Set to false for local dev
					SameSite: http.SameSiteLaxMode,
					Expires:  newToken.Expiry,
				})

				// Store the expiry time so we can check it next time
				http.SetCookie(w, &http.Cookie{
					Name:     "spotify_token_expiry",
					Value:    newToken.Expiry.Format(time.RFC3339),
					Path:     "/",
					HttpOnly: true,
					Secure:   false,
					SameSite: http.SameSiteLaxMode,
					Expires:  newToken.Expiry,
				})

				// Update the refresh token if Spotify sent a new one
				if newToken.RefreshToken != "" {
					http.SetCookie(w, &http.Cookie{
						Name:     "spotify_refresh_token",
						Value:    newToken.RefreshToken,
						Path:     "/",
						HttpOnly: true,
						Secure:   false,
						SameSite: http.SameSiteLaxMode,
						MaxAge:   60 * 60 * 24 * 30, // 30 days
					})
				}

				log.Println("Token refreshed successfully")
				accessCookie.Value = newToken.AccessToken
			}

			// Add the valid access token to the request context
			// Handlers can retrieve it with: token := r.Context().Value(handlers.AccessTokenKey).(string)
			ctx := context.WithValue(r.Context(), AccessTokenKey, accessCookie.Value)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
	}
}
