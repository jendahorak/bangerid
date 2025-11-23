package main

import (
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jendahorak/bangerid/internal/handlers"
	spotifyClient "github.com/jendahorak/bangerid/internal/spotify"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/spotify"
)

var (
	oauthConfig *oauth2.Config
	tracksCache []spotifyClient.Track // Simple global cache for single user
)

// loggingMiddleware wraps an HTTP handler and logs each request
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Custom ResponseWriter to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		// Log: method, path, status, duration, remote address
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.statusCode,
			"duration", time.Since(start),
			"remote", r.RemoteAddr,
		)
	})
}

// responseWriter wraps http.ResponseWriter to capture the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		slog.Warn("Warning: .env file not found, using system environment variables")
	}

	// Initialize OAuth config after env vars are loaded
	oauthConfig = &oauth2.Config{
		ClientID:     os.Getenv("CLIENT_ID"),
		ClientSecret: os.Getenv("CLIENT_SECRET"),
		RedirectURL:  os.Getenv("REDIRECT_URL"),
		Scopes:       []string{"user-read-private", "user-read-email", "playlist-read-private", "user-library-read", "streaming"},
		Endpoint:     spotify.Endpoint,
	}

	// Validate that required env vars are set
	if oauthConfig.ClientID == "" || oauthConfig.ClientSecret == "" {
		slog.Error("missing required env vars", slog.String("vars", "CLIENT_ID, CLIENT_SECRET"))
		os.Exit(1)
	}

	// Serve static files (CSS, JS) from /static/ directory
	fs := http.FileServer(http.Dir("web/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Home page - serves the main HTML template
	http.HandleFunc("/", homeHandler)

	// OAuth routes
	http.HandleFunc("/login", handlers.LoginHandler(oauthConfig))
	http.HandleFunc("/spotify-auth", handlers.CallbackHandler(oauthConfig))

	// Grid endpoint - renders the track grid
	http.HandleFunc("/grid", handlers.RequireAuth(oauthConfig)(gridHandler))

	// Playback endpoint
	http.HandleFunc("/play", handlers.RequireAuth(oauthConfig)(playHandler))

	http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:   "spotify_access_token",
			Value:  "",
			MaxAge: -1,
		})
		http.SetCookie(w, &http.Cookie{
			Name:   "spotify_refresh_token",
			Value:  "",
			MaxAge: -1,
		})
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
	// Start the server with logging middleware
	port := ":3000"
	slog.Info("server starting", slog.String("url", "http://localhost"+port))
	slog.Info("authenticate", slog.String("url", "http://localhost"+port+"/login"))

	// Wrap all routes with logging middleware
	if err := http.ListenAndServe(port, loggingMiddleware(http.DefaultServeMux)); err != nil {
		slog.Error("server failed", slog.Any("error", err))
		os.Exit(1)
	}
}

// homeHandler serves the main index.html template
func homeHandler(w http.ResponseWriter, r *http.Request) {
	// Only serve index.html on the root path
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Retrieve the token cookie to pass to the frontend
	cookie, err := r.Cookie("spotify_access_token")
	var token string
	loggedIn := err == nil
	if loggedIn {
		token = cookie.Value
	}

	tmpl, err := template.ParseFiles("web/templates/index.html")
	if err != nil {
		slog.Error("template parse error", slog.Any("error", err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	data := struct {
		LoggedIn bool
		Token    string
	}{
		LoggedIn: loggedIn,
		Token:    token,
	}

	if err := tmpl.Execute(w, data); err != nil {
		slog.Error("template execute error", slog.Any("error", err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// gridHandler renders the track grid as HTML
func gridHandler(w http.ResponseWriter, r *http.Request) {
	// If cache is empty, fetch tracks
	if len(tracksCache) == 0 {
		accessToken := r.Context().Value(handlers.AccessTokenKey).(string)

		slog.Info("cache empty, fetching tracks from Spotify")
		tracks, err := spotifyClient.FetchLikedTracks(accessToken)
		if err != nil {
			slog.Error("failed to fetch tracks", slog.Any("error", err))
			http.Error(w, "Failed to load tracks", http.StatusInternalServerError)
			return
		}

		tracksCache = tracks
		slog.Info("cached tracks", slog.Int("count", len(tracksCache)))
	}

	// Render the grid template
	tmpl, err := template.ParseFiles("web/templates/grid.html")
	if err != nil {
		slog.Error("template parse error", slog.Any("error", err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, tracksCache); err != nil {
		slog.Error("template execute error", slog.Any("error", err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

// playHandler triggers playback on the client's device
func playHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	accessToken := r.Context().Value(handlers.AccessTokenKey).(string)
	trackID := r.URL.Query().Get("track_id")
	deviceID := r.PostFormValue("device_id")

	if trackID == "" || deviceID == "" {
		slog.Warn("missing track_id or device_id", "track_id", trackID, "device_id", deviceID)
		http.Error(w, "Missing track_id or device_id", http.StatusBadRequest)
		return
	}

	slog.Info("starting playback", "track", trackID, "device", deviceID)

	if err := spotifyClient.PlayTrack(accessToken, deviceID, trackID); err != nil {
		slog.Error("playback failed", slog.Any("error", err))
		http.Error(w, "Failed to start playback", http.StatusInternalServerError)
		return
	}

	// Return 204 No Content so HTMX does nothing (no swap)
	w.WriteHeader(http.StatusNoContent)
}

