package gamesource

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/phergul/fiach/internal/steam"
)

const (
	steamArtworkRoutePrefix = "/artwork/steam/"
	steamArtworkRouteRoot   = "/artwork/steam"
	steamArtworkCache       = "private, no-cache"
)

func NewSteamArtworkMiddleware(steamSource *SteamSource) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.URL.Path != steamArtworkRouteRoot && !strings.HasPrefix(req.URL.Path, steamArtworkRoutePrefix) {
				next.ServeHTTP(rw, req)
				return
			}

			serveSteamArtwork(rw, req, steamSource)
		})
	}
}

func serveSteamArtwork(rw http.ResponseWriter, req *http.Request, steamSource *SteamSource) {
	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		rw.Header().Set("Allow", http.MethodGet+", "+http.MethodHead)
		http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	appID, imageType, ok := parseSteamArtworkPath(req.URL.Path)
	if !ok {
		http.NotFound(rw, req)
		return
	}

	if steamSource == nil {
		http.Error(rw, "Steam source is not configured", http.StatusInternalServerError)
		return
	}

	artworkRoot, err := steamSource.getArtworkRoot(req.Context())
	if err != nil {
		http.Error(rw, fmt.Sprintf("locate Steam artwork: %v", err), http.StatusInternalServerError)
		return
	}

	imagePath, err := steam.ResolveGameImagePath(artworkRoot, appID, imageType)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	if imagePath == "" {
		http.NotFound(rw, req)
		return
	}

	file, err := os.Open(imagePath)
	if err != nil {
		http.Error(rw, fmt.Sprintf("open Steam artwork: %v", err), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		http.Error(rw, fmt.Sprintf("stat Steam artwork: %v", err), http.StatusInternalServerError)
		return
	}
	if stat.IsDir() {
		http.NotFound(rw, req)
		return
	}

	rw.Header().Set("Cache-Control", steamArtworkCache)
	http.ServeContent(rw, req, stat.Name(), stat.ModTime(), file)
}

func parseSteamArtworkPath(path string) (string, steam.ImageType, bool) {
	if !strings.HasPrefix(path, steamArtworkRoutePrefix) {
		return "", "", false
	}

	route := strings.TrimPrefix(path, steamArtworkRoutePrefix)
	parts := strings.Split(route, "/")
	if len(parts) != 2 || !isSteamAppID(parts[0]) {
		return "", "", false
	}

	imageType := steam.ImageType(parts[1])
	switch imageType {
	case steam.ImageTypeBanner, steam.ImageTypeHero, steam.ImageTypeLogo:
		return parts[0], imageType, true
	default:
		return "", "", false
	}
}

func isSteamAppID(appID string) bool {
	if appID == "" {
		return false
	}

	for _, char := range appID {
		if char < '0' || char > '9' {
			return false
		}
	}

	return true
}
