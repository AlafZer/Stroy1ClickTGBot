package order

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

type samplePayload struct {
	A int `json:"a"`
}

func TestReadJSON_OK(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"a":123}`))

	var p samplePayload
	if err := readJSON(w, r, &p); err != nil {
		t.Fatalf("readJSON: %v", err)
	}
	if p.A != 123 {
		t.Fatalf("expected A=123, got %d", p.A)
	}
}

func TestReadJSON_RejectsMultipleJSONValues(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"a":1}{"a":2}`))

	var p samplePayload
	if err := readJSON(w, r, &p); err == nil {
		t.Fatal("expected error for multiple JSON values")
	}
}

func TestReadJSON_TooLarge(t *testing.T) {
	// Make body > 1MB.
	big := bytes.Repeat([]byte{'a'}, 1048576+10)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(big))

	var p any
	if err := readJSON(w, r, &p); err == nil {
		t.Fatal("expected error for too large body")
	}
}

func TestWriteJSON_SetsHeadersAndStatus(t *testing.T) {
	ordR := &OrderReceiver{}
	rw := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	h := http.Header{}
	h.Set("X-Test", "1")

	payload := map[string]any{"ok": true}
	if err := ordR.writeJSON(rw, http.StatusCreated, payload, h); err != nil {
		t.Fatalf("writeJSON: %v", err)
	}

	res := rw.Result()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected content-type application/json, got %q", ct)
	}
	if res.Header.Get("X-Test") != "1" {
		t.Fatalf("expected custom header")
	}
	_ = req
}

func TestErrorJSON_DefaultStatus(t *testing.T) {
	ordR := &OrderReceiver{}
	rw := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	err := ordR.errorJSON(rw, errTest("boom"))
	if err != nil {
		t.Fatalf("errorJSON returned error: %v", err)
	}
	if rw.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rw.Code)
	}
	_ = req
}

type errTest string

func (e errTest) Error() string { return string(e) }
