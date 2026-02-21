package order

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRoutes_HeartbeatPing(t *testing.T) {
	ordR := &OrderReceiver{}
	h := ordR.routes()

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rw := httptest.NewRecorder()

	h.ServeHTTP(rw, req)

	if rw.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rw.Code)
	}
}
