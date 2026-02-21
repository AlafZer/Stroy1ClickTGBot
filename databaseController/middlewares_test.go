package dbController

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidToken(t *testing.T) {
	ctrl := &DatabaseController{token: "abc123"}

	if got := ctrl.validToken("abc123"); got != 1 {
		t.Fatalf("expected valid token (1), got %d", got)
	}
	if got := ctrl.validToken("zzz999"); got != 0 {
		t.Fatalf("expected invalid token (0), got %d", got)
	}
	if got := ctrl.validToken("short"); got != 0 {
		t.Fatalf("expected invalid token for different length (0), got %d", got)
	}
}

func TestAuthMiddleware_AllowsValidToken(t *testing.T) {
	ctrl := &DatabaseController{token: "secret"}

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	h := ctrl.auth(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rw := httptest.NewRecorder()

	h.ServeHTTP(rw, req)

	if rw.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rw.Code)
	}
	if !called {
		t.Fatal("expected next handler to be called")
	}
}

func TestAuthMiddleware_RejectsInvalidToken(t *testing.T) {
	ctrl := &DatabaseController{token: "secret"}

	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	h := ctrl.auth(next)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	rw := httptest.NewRecorder()

	h.ServeHTTP(rw, req)

	if rw.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rw.Code)
	}
	if called {
		t.Fatal("did not expect next handler to be called")
	}
}
