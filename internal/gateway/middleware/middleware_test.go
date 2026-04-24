package middleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRecover(t *testing.T) {
	logger := slog.Default()

	// Create a handler that panics
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	wrapped := Recover(logger)(panicHandler)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestRequestID(t *testing.T) {
	middleware := RequestID()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check that X-Request-ID header is set
		requestID := r.Context().Value(RequestIDKey)
		w.WriteHeader(http.StatusOK)
		if requestID != nil {
			w.Write([]byte(requestID.(string)))
		}
	})

	wrapped := middleware(handler)

	// Test with no existing request ID
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotEmpty(t, rec.Header().Get("X-Request-ID"))

	// Test with existing request ID
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.Header.Set("X-Request-ID", "existing-id")
	rec2 := httptest.NewRecorder()

	wrapped.ServeHTTP(rec2, req2)

	require.Equal(t, http.StatusOK, rec2.Code)
	require.Equal(t, "existing-id", rec2.Header().Get("X-Request-ID"))
}

func TestLogging(t *testing.T) {
	logger := slog.Default()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := Logging(logger)(handler)

	req := httptest.NewRequest("GET", "/test/path", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestCORS(t *testing.T) {
	middleware := CORS([]string{"*"})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware(handler)

	// Test preflight request
	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Code)
	require.Equal(t, "http://example.com", rec.Header().Get("Access-Control-Allow-Origin"))

	// Test regular request with allowed origin
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.Header.Set("Origin", "http://example.com")
	rec2 := httptest.NewRecorder()

	wrapped.ServeHTTP(rec2, req2)

	require.Equal(t, http.StatusOK, rec2.Code)
	require.Equal(t, "http://example.com", rec2.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_SpecificOrigin(t *testing.T) {
	middleware := CORS([]string{"http://allowed.com"})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := middleware(handler)

	// Test with allowed origin
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "http://allowed.com")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "http://allowed.com", rec.Header().Get("Access-Control-Allow-Origin"))

	// Test with disallowed origin
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.Header.Set("Origin", "http://disallowed.com")
	rec2 := httptest.NewRecorder()

	wrapped.ServeHTTP(rec2, req2)

	require.Equal(t, http.StatusOK, rec2.Code) // Still 200, just no CORS header
	require.Empty(t, rec2.Header().Get("Access-Control-Allow-Origin"))
}

func TestMetrics(t *testing.T) {
	collector := NewMetricsCollector()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := Metrics(collector)(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, collector.requestCount)
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rec, statusCode: http.StatusOK}

	rw.WriteHeader(http.StatusNotFound)

	require.Equal(t, http.StatusNotFound, rw.statusCode)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestMetricsCollector_RecordRequest(t *testing.T) {
	collector := NewMetricsCollector()

	collector.RecordRequest("GET", "/test", 200, 100*1000000) // 100ms

	require.Equal(t, 1, collector.requestCount["GET /test"])
	require.Len(t, collector.latencies["GET /test"], 1)
}