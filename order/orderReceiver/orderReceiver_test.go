package order

import "testing"

func TestNewOrderReceiver_SetsFields(t *testing.T) {
	ordR := New(nil, "localhost:9092")
	if ordR == nil {
		t.Fatal("expected non-nil OrderReceiver")
	}
	if ordR.instance != "localhost:9092" {
		t.Fatalf("expected instance set, got %q", ordR.instance)
	}
	if ordR.store != nil {
		t.Fatal("expected store to be nil when passed nil")
	}
}
