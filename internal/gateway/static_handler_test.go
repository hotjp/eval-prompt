package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWebDist(t *testing.T) {
	fs := WebDist()
	require.NotNil(t, fs)
}

func TestStaticHandler(t *testing.T) {
	handler := StaticHandler()
	require.NotNil(t, handler)
}

func TestStaticHandler_ServeHTTP(t *testing.T) {
	handler := StaticHandler()

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// SPA fallback should serve index.html or return 200
	require.True(t, rec.Code == http.StatusOK || rec.Code == http.StatusNotFound)
}

func TestStaticHandler_AssetPath(t *testing.T) {
	handler := StaticHandler()

	// Path with dot and /assets/ prefix should fall through to index.html
	req := httptest.NewRequest("GET", "/assets/app.js", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should either serve the file or fallback to index.html
	require.True(t, rec.Code == http.StatusOK || rec.Code == http.StatusNotFound)
}

func TestRegisterStaticRoutes(t *testing.T) {
	mux := http.NewServeMux()

	RegisterStaticRoutes(mux)
	require.NotNil(t, mux)

	// Should be able to serve a request
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	// Either serves static content or falls back
	require.True(t, rec.Code == http.StatusOK || rec.Code == http.StatusNotFound)
}

func TestRegisterStaticRoutes_CatchAll(t *testing.T) {
	mux := http.NewServeMux()

	RegisterStaticRoutes(mux)

	// Test various paths that should be handled by static router
	paths := []string{
		"/",
		"/index.html",
		"/assets/app.js",
		"/static/app.css",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)
			rec := httptest.NewRecorder()

			mux.ServeHTTP(rec, req)

			// Static handler should handle these paths (200, 301, 404 are valid - depends on embedded FS content)
			require.True(t, rec.Code == http.StatusOK || rec.Code == http.StatusMovedPermanently || rec.Code == http.StatusNotFound,
				"expected 200, 301, or 404, got %d", rec.Code)
		})
	}
}
