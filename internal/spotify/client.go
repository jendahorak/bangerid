package spotify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Track represents a simplified Spotify track for our grid
type Track struct {
	ID         string
	Name       string
	Artist     string
	AlbumImage string
}

type LinkedFrom struct {
	ID  string `json:"id"`
	URI string `json:"uri"`
}

// SavedTracksResponse matches Spotify's API response structure
type SavedTracksResponse struct {
	Items []struct {
		AddedAt string `json:"added_at"`
		Track   struct {
			ID         string      `json:"id"`
			URI        string      `json:"uri"`
			Name       string      `json:"name"`
			LinkedFrom *LinkedFrom `json:"linked_from"`
			Artists    []struct {
				Name string `json:"name"`
			} `json:"artists"`
			Album struct {
				Images []struct {
					URL    string `json:"url"`
					Height int    `json:"height"`
					Width  int    `json:"width"`
				} `json:"images"`
			} `json:"album"`
		} `json:"track"`
	} `json:"items"`
	Next   *string `json:"next"`  // URL to next page, null if last page
	Total  int     `json:"total"` // Total number of liked tracks
	Limit  int     `json:"limit"`
	Offset int     `json:"offset"`
}

// FetchLikedTracks retrieves all of the user's saved/liked tracks from Spotify
func FetchLikedTracks(accessToken string) ([]Track, error) {
	var allTracks []Track
	url := "https://api.spotify.com/v1/me/tracks?limit=50&market=from_token"

	// Create HTTP client
	client := &http.Client{}

	for url != "" {
		// Create request
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		// Add authorization header with the access token
		req.Header.Set("Authorization", "Bearer "+accessToken)

		// Make the request
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch tracks: %w", err)
		}
		defer resp.Body.Close()

		// Check for errors
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("spotify API error %d: %s", resp.StatusCode, string(body))
		}

		// Parse JSON response
		var response SavedTracksResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		// Extract simplified track data
		for _, item := range response.Items {
			stableURI := item.Track.URI

			if item.Track.LinkedFrom != nil && item.Track.LinkedFrom.URI != "" {
				stableURI = item.Track.LinkedFrom.URI
			}

			track := Track{
				ID:   stableURI,
				Name: item.Track.Name,
			}

			// Get first artist name
			if len(item.Track.Artists) > 0 {
				track.Artist = item.Track.Artists[0].Name
			}

			// Get smallest album image (usually the last one in the array)
			// Images are ordered: [0]=largest, [last]=smallest (typically 64x64)
			images := item.Track.Album.Images
			if len(images) > 0 {
				// Try to find exact 64x64 match first
				found := false
				for _, img := range images {
					if img.Height == 64 && img.Width == 64 {
						track.AlbumImage = img.URL
						found = true
						break
					}
				}
				// Fallback to last image (usually smallest) or first (if only one exists)
				if !found {
					track.AlbumImage = images[len(images)-1].URL
				}
			} else {
				// Log missing images to debug console
				fmt.Printf("Warning: Track '%s' (ID: %s) has no album images - SKIPPING\n", track.Name, track.ID)
				continue // Skip this track entirely
			}

			allTracks = append(allTracks, track)
		}

		// Check if there's a next page
		if response.Next != nil {
			url = *response.Next
		} else {
			url = "" // Exit loop
		}
	}

	return allTracks, nil
}

// PlayTrack starts playback of a specific track on a specific device
func PlayTrack(accessToken, deviceID, trackURI string) error {
	url := fmt.Sprintf("https://api.spotify.com/v1/me/player/play?device_id=%s", deviceID)

	// Create the body: {"uris": ["spotify:track:track_uri"]}
	bodyData := map[string][]string{
		"uris": {trackURI},
	}
	jsonBody, err := json.Marshal(bodyData)
	if err != nil {
		return fmt.Errorf("failed to marshal play request: %w", err)
	}

	// Create request
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to perform play request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("spotify play error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
