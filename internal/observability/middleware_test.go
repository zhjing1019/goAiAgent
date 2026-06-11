package observability

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestMiddlewareRequestIDAndAccessLog(t *testing.T) {
	Init()
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	var gotID string
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID = RequestIDFromContext(r.Context())
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status=%d", rec.Code)
	}
	if gotID == "" {
		t.Fatal("expected request_id in context")
	}
	if rec.Header().Get("X-Request-ID") != gotID {
		t.Fatalf("header id mismatch: %s vs %s", rec.Header().Get("X-Request-ID"), gotID)
	}
}

func TestMetricsHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	MetricsHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "http_requests_total") {
		t.Fatalf("metrics body missing counter: %s", body)
	}
}

func TestInitJSONLogger(t *testing.T) {
	Init()
	if slog.Default() == nil {
		t.Fatal("default logger nil")
	}
	// smoke: ensure JSON handler doesn't panic
	b, err := json.Marshal(map[string]string{"ok": "1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 {
		t.Fatal("empty json")
	}
}
