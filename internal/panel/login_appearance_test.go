package panel

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/kejilion/kejilion-panel/internal/desktopwallpapers"
	"github.com/kejilion/kejilion-panel/internal/scenepacks"
	"github.com/kejilion/kejilion-panel/internal/store"
)

func savedLoginAppearance(t *testing.T, s *Server, wallpaper string) {
	t.Helper()
	_, version := s.store.Appearance()
	if err := s.store.ReplaceAppearance(version, store.Appearance{
		Theme: "dark", Wallpaper: wallpaper, ClassicLevel: "off",
		Colors: &store.AppearanceColors{Brand: "#356fc0", Neutral: "#34465c", Signature: "#23a6bd"},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestLoginBootstrapUsesLatestSavedAppearanceWithoutPrivateMetadata(t *testing.T) {
	s, _ := newTestServer(t)
	read := func() *loginAppearance {
		t.Helper()
		response := performRequest(s, http.MethodGet, "/api/v1/auth/bootstrap", nil, nil)
		var payload struct {
			Appearance *loginAppearance `json:"appearance"`
		}
		if response.Code != http.StatusOK || response.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("bootstrap: %d %v", response.Code, response.Header())
		}
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		for _, private := range []string{"resourceVersion", "classicLevel", "configured", "imageURL", "thumbURL"} {
			if strings.Contains(response.Body.String(), private) {
				t.Fatalf("private metadata: %s", response.Body.String())
			}
		}
		return payload.Appearance
	}
	if read() != nil {
		t.Fatal("unconfigured installation should preserve browser preferences")
	}
	savedLoginAppearance(t, s, "horizon")
	first := read()
	if first.Theme != "dark" || first.Colors.Brand != "#356fc0" || first.Wallpaper.URL != "/wallpapers/kpanel-desktop-horizon.webp" || !first.Wallpaper.Bright {
		t.Fatalf("saved theme: %+v", first)
	}
	savedLoginAppearance(t, s, "rift")
	if next := read(); next.Wallpaper.URL != "/wallpapers/kpanel-desktop-rift.webp" || next.Wallpaper.Bright {
		t.Fatalf("refresh retained previous selection: %+v", next)
	}
}

func TestLoginWallpaperOnlyExposesCurrentThumbnailAndRevokesPreviousURL(t *testing.T) {
	s, _ := newTestServer(t)
	thumb := wallpaperJPEG(t, 64, 36)
	prepared, err := desktopwallpapers.Prepare(desktopwallpapers.Metadata{Name: "private library name", FocusX: 700, FocusY: 320}, wallpaperJPEG(t, 320, 180), thumb)
	if err != nil {
		t.Fatal(err)
	}
	item, err := s.desktopWallpapers.Add(prepared)
	if err != nil {
		t.Fatal(err)
	}
	savedLoginAppearance(t, s, "custom:"+item.ID)
	appearance := s.loginAppearance()
	if appearance.Wallpaper.FocusX != 700 || appearance.Wallpaper.FocusY != 320 || strings.Contains(appearance.Wallpaper.URL, item.ID) {
		t.Fatalf("public descriptor: %+v", appearance)
	}
	address := appearance.Wallpaper.URL
	response := performRequest(s, http.MethodGet, address, nil, nil)
	if response.Code != 200 || !bytes.Equal(response.Body.Bytes(), thumb) || response.Header().Get("Cache-Control") != "no-store" || response.Header().Get("Content-Type") != "image/jpeg" {
		t.Fatalf("thumbnail: %d %v", response.Code, response.Header())
	}
	head := performRequest(s, http.MethodHead, address, nil, nil)
	if head.Code != 200 || head.Body.Len() != 0 || head.Header().Get("Content-Length") != response.Header().Get("Content-Length") {
		t.Fatalf("HEAD: %d %v", head.Code, head.Header())
	}
	for _, path := range []string{address + "?image=original", address + "/image", loginWallpaperPath + "/" + item.ID, loginWallpaperPath + "/" + strings.Repeat("0", 64)} {
		if r := performRequest(s, http.MethodGet, path, nil, nil); r.Code != 404 {
			t.Fatalf("unexpected public path %s: %d", path, r.Code)
		}
	}
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		if r := performRequest(s, method, address, nil, nil); r.Code != 405 {
			t.Fatalf("write %s: %d", method, r.Code)
		}
	}
	for _, path := range []string{desktopWallpapersPath, desktopWallpapersPath + "/" + item.ID + "/image", desktopWallpapersPath + "/" + item.ID + "/thumb"} {
		if r := performRequest(s, http.MethodGet, path, nil, nil); r.Code != 401 {
			t.Fatalf("private asset %s: %d", path, r.Code)
		}
	}
	savedLoginAppearance(t, s, "orbit")
	if r := performRequest(s, http.MethodGet, address, nil, nil); r.Code != 404 {
		t.Fatalf("revoked preview: %d", r.Code)
	}
}

func TestLoginScenePreviewNeverDownloadsUninstalledAssets(t *testing.T) {
	s, _ := newTestServer(t)
	s.scenePacks.Close()
	s.scenePacks = scenepacks.Open(t.TempDir(), func(context.Context, string, int64) ([]byte, error) {
		t.Fatal("public request initiated remote fetch")
		return nil, scenepacks.ErrSource
	})
	savedLoginAppearance(t, s, "pack:orbital-station")
	if r := performRequest(s, http.MethodGet, s.loginAppearance().Wallpaper.URL, nil, nil); r.Code != 404 {
		t.Fatalf("uninstalled pack: %d", r.Code)
	}
}

func TestLoginPreviewRespectsSecurityEntrance(t *testing.T) {
	s, tokenPath := newTestServer(t)
	session, csrf := bootstrapCookies(t, s, tokenPath)
	savedLoginAppearance(t, s, "pack:orbital-station")
	_, version := s.store.SecurityEntrance()
	body, _ := json.Marshal(map[string]any{"enabled": true, "path": "panel-secure1", "expectedResourceVersion": version})
	if r := authenticatedSiteRequest(s, session, csrf, http.MethodPut, "/api/v1/settings/security-entry", body, true); r.Code != 200 {
		t.Fatalf("entry setting: %d %s", r.Code, r.Body.String())
	}
	for _, path := range []string{"/api/v1/auth/bootstrap", s.loginAppearance().Wallpaper.URL} {
		if r := performRequest(s, http.MethodGet, path, nil, nil); r.Code != 404 {
			t.Fatalf("entry bypass %s: %d", path, r.Code)
		}
	}
}
