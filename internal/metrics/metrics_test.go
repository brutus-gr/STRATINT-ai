package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPCollectorRecordsMetrics(t *testing.T) {
	collector, err := NewHTTPCollector()
	if err != nil {
		t.Fatalf("NewHTTPCollector returned error: %v", err)
	}

	handlerInvoked := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerInvoked = true
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("ok"))
	})

	instrumented := collector.InstrumentHandler(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	instrumented.ServeHTTP(rr, req)

	if !handlerInvoked {
		t.Fatal("expected handler to be invoked")
	}

	if rr.Code != http.StatusAccepted {
		t.Fatalf("unexpected status code: %d", rr.Code)
	}

	metricsRR := httptest.NewRecorder()
	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	collector.Handler().ServeHTTP(metricsRR, metricsReq)

	if metricsRR.Code != http.StatusOK {
		t.Fatalf("expected metrics handler to return 200, got %d", metricsRR.Code)
	}

	body := metricsRR.Body.String()
	if !strings.Contains(body, `osintmcp_http_requests_total{method="GET",path="/test",status="202"} 1`) {
		t.Fatalf("requests_total metric not recorded, body=%q", body)
	}

	if !strings.Contains(body, `osintmcp_http_request_duration_seconds_count{method="GET",path="/test",status="202"} 1`) {
		t.Fatalf("request_duration_seconds_count metric not recorded, body=%q", body)
	}
}
