package monitoring

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoadUsesBearerTokenAndReadsMetrics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch request.URL.Path {
		case "/api/cpu/current":
			_, _ = w.Write([]byte(`{"percent":12.5}`))
		case "/api/memory/current":
			_, _ = w.Write([]byte(`{"usedPercent":50,"used":1024,"total":2048}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	t.Setenv("SENTINEL_URL", server.URL)
	t.Setenv("SENTINEL_TOKEN", "secret")

	metrics := Load()
	if !metrics.Available || metrics.CPU != 12.5 || metrics.RAM != 50 || metrics.RAMUsed != 1024 || metrics.RAMTotal != 2048 {
		t.Fatalf("unexpected metrics: %#v", metrics)
	}
}
