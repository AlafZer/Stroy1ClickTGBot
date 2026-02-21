package dbController

import (
	"context"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestListenAndServe_WrapsError(t *testing.T) {
	dbCtrl := &DatabaseController{
		token:  "secret",
		server: &http.Server{Addr: "bad address"},
	}

	err := dbCtrl.ListenAndServe(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "error from DatabaseController") {
		t.Fatalf("expected wrapped error, got: %v", err)
	}
}

func TestShutdown_StopsRunningServer(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	srv := &http.Server{Handler: http.NewServeMux()}
	dbCtrl := &DatabaseController{server: srv}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.Serve(ln)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	dbCtrl.Shutdown(ctx)

	select {
	case <-serveErr:
		// ok
	case <-time.After(2 * time.Second):
		t.Fatal("server did not stop")
	}
}
