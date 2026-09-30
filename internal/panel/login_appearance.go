package panel

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/kejilion/kejilion-panel/internal/desktopwallpapers"
	"github.com/kejilion/kejilion-panel/internal/store"
)

const loginWallpaperPath = "/api/v1/auth/wallpaper"

// Login appearance is public branding. It deliberately omits private asset IDs,
// names, library contents and the authenticated settings resource version.
type loginAppearance struct {
	Theme     string                  `json:"theme"`
	Colors    *store.AppearanceColors `json:"colors"`
	Wallpaper loginWallpaper          `json:"wallpaper"`
}

type loginWallpaper struct {
	URL    string `json:"url"`
	FocusX int    `json:"focusX"`
	FocusY int    `json:"focusY"`
	Bright bool   `json:"bright"`
}

func (s *Server) loginAppearance() *loginAppearance {
	value, version := s.store.Appearance()
	if value == nil || store.ValidateAppearance(*value) != nil {
		return nil // Older installations keep their browser preferences until imported.
	}
	wallpaper := loginWallpaper{URL: "/wallpapers/kpanel-desktop.webp", FocusX: 500, FocusY: 500}
	switch value.Wallpaper {
	case "orbit", "horizon", "rift", "prism":
		wallpaper.URL = "/wallpapers/kpanel-desktop-" + value.Wallpaper + ".webp"
		wallpaper.Bright = value.Wallpaper == "horizon"
	default:
		if id := strings.TrimPrefix(value.Wallpaper, "custom:"); id != value.Wallpaper && s.desktopWallpapers != nil {
			items, _ := s.desktopWallpapers.List()
			for _, item := range items {
				if item.ID == id {
					wallpaper.URL = loginWallpaperPath + "/" + strings.TrimPrefix(version, "sha256:")
					wallpaper.FocusX, wallpaper.FocusY = item.FocusX, item.FocusY
					wallpaper.Bright = item.Luminance >= 50
					break
				}
			}
		} else if strings.HasPrefix(value.Wallpaper, "pack:") {
			wallpaper.URL = loginWallpaperPath + "/" + strings.TrimPrefix(version, "sha256:")
		}
	}
	return &loginAppearance{Theme: value.Theme, Colors: value.Colors, Wallpaper: wallpaper}
}

// Only the currently selected artwork's bounded preview can be read before sign-in.
// No caller-supplied asset ID, original image, scene code or remote download is accepted.
func (s *Server) handleLoginWallpaper(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.URL.RawPath != "" || r.URL.RawQuery != "" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	value, version := s.store.Appearance()
	if value == nil || r.URL.Path != loginWallpaperPath+"/"+strings.TrimPrefix(version, "sha256:") {
		http.NotFound(w, r)
		return
	}
	if id := strings.TrimPrefix(value.Wallpaper, "custom:"); id != value.Wallpaper && s.desktopWallpapers != nil {
		file, size, contentType, err := s.desktopWallpapers.OpenFile(id, "thumb")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer file.Close()
		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodGet {
			_, _ = io.Copy(w, file)
		}
		return
	}
	if id := strings.TrimPrefix(value.Wallpaper, "pack:"); id != value.Wallpaper && s.scenePacks != nil {
		if !s.scenePackStream(w, r) {
			return
		}
		defer func() { <-s.scenePackStreams }()
		body, err := s.scenePacks.InstalledPreview(id, desktopwallpapers.MaxThumbBytes)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/webp")
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodGet {
			_, _ = w.Write(body)
		}
		return
	}
	http.NotFound(w, r)
}
