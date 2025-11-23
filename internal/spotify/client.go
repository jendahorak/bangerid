package spotify

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Track represents a simplified Spotify track for our grid
type Track struct {
	ID          string
	Name        string
	Artist      string
	AlbumImage  string
}

// SavedTracksResponse matches Spotify's API response structure
type SavedTracksResponse struct {
	Items []struct {
		AddedAt string `json:"added_at"`
		Track   struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Artists []struct {
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
	Next   *string `json:"next"`   // URL to next page, null if last page
	Total  int     `json:"total"`  // Total number of liked tracks
	Limit  int     `json:"limit"`
	Offset int     `json:"offset"`
}

// FetchLikedTracks retrieves all of the user's saved/liked tracks from Spotify
func FetchLikedTracks(accessToken string) ([]Track, error) {
	var allTracks []Track
	url := "https://api.spotify.com/v1/me/tracks?limit=50"

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
			track := Track{
				ID:   item.Track.ID,
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
