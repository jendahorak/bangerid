package main

import (
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jendahorak/bangerid/internal/handlers"
	"github.com/joho/godotenv"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/spotify"
)

var oauthConfig *oauth2.Config

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
		Scopes:       []string{"user-read-private", "user-read-email", "playlist-read-private", "user-library-read"},
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

	_, err := r.Cookie("spotify_access_token")
	loggedIn := err == nil

	tmpl, err := template.ParseFiles("web/templates/index.html")
	if err != nil {
		slog.Error("template parse error", slog.Any("error", err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := tmpl.Execute(w, map[string]bool{"LoggedIn": loggedIn}); err != nil {
		slog.Error("template execute error", slog.Any("error", err))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

}
