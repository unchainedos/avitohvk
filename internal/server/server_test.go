package server

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func discardLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }

func bareServer() *Server {
	return &Server{
		internalServer: &http.Server{},
		channelErr:     make(chan error, chanBufferSize),
	}
}

func TestSetReadTimeout(t *testing.T) {
	t.Parallel()

	srv := bareServer()
	SetReadTimeout(5 * time.Second)(srv)
	if srv.internalServer.ReadTimeout != 5*time.Second {
		t.Errorf("ReadTimeout = %v, want %v", srv.internalServer.ReadTimeout, 5*time.Second)
	}
}

func TestSetWriteTimeout(t *testing.T) {
	t.Parallel()

	srv := bareServer()
	SetWriteTimeout(7 * time.Second)(srv)
	if srv.internalServer.WriteTimeout != 7*time.Second {
		t.Errorf("WriteTimeout = %v, want %v", srv.internalServer.WriteTimeout, 7*time.Second)
	}
}

func TestSetAddr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		addr string
		want string
	}{
		{name: "custom address is used as-is", addr: "127.0.0.1:9999", want: "127.0.0.1:9999"},
		{name: "empty address falls back to the default", addr: "", want: defaultAddr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := bareServer()
			SetAddr(tt.addr)(srv)
			if srv.internalServer.Addr != tt.want {
				t.Errorf("Addr = %q, want %q", srv.internalServer.Addr, tt.want)
			}
		})
	}
}

func TestSetShutdownTimeout(t *testing.T) {
	t.Parallel()

	srv := bareServer()
	SetShutdownTimeout(90 * time.Second)(srv)
	if srv.shutdownTimeout != 90*time.Second {
		t.Errorf("shutdownTimeout = %v, want %v", srv.shutdownTimeout, 90*time.Second)
	}
}

func TestNewServer_Defaults(t *testing.T) {
	t.Parallel()

	srv := NewServer(http.NewServeMux())
	defer func() { _ = srv.internalServer.Close() }()

	if srv.internalServer.ReadTimeout != defaultReadTimeout {
		t.Errorf("ReadTimeout = %v, want %v", srv.internalServer.ReadTimeout, defaultReadTimeout)
	}
	if srv.internalServer.WriteTimeout != defaultWriteTimeout {
		t.Errorf("WriteTimeout = %v, want %v", srv.internalServer.WriteTimeout, defaultWriteTimeout)
	}
	if srv.internalServer.Addr != defaultAddr {
		t.Errorf("Addr = %q, want %q", srv.internalServer.Addr, defaultAddr)
	}
	if srv.shutdownTimeout != defaultShutdownTimeout {
		t.Errorf("shutdownTimeout = %v, want %v", srv.shutdownTimeout, defaultShutdownTimeout)
	}
	if cap(srv.channelErr) != chanBufferSize {
		t.Errorf("channelErr capacity = %d, want %d", cap(srv.channelErr), chanBufferSize)
	}
}

func TestNewServer_AppliesOptions(t *testing.T) {
	t.Parallel()

	srv := NewServer(http.NewServeMux(),
		SetAddr("127.0.0.1:0"),
		SetReadTimeout(1*time.Second),
		SetWriteTimeout(2*time.Second),
		SetShutdownTimeout(3*time.Second),
	)
	defer func() { _ = srv.internalServer.Close() }()

	if srv.internalServer.Addr != "127.0.0.1:0" {
		t.Errorf("Addr = %q, want %q", srv.internalServer.Addr, "127.0.0.1:0")
	}
	if srv.internalServer.ReadTimeout != time.Second {
		t.Errorf("ReadTimeout = %v, want %v", srv.internalServer.ReadTimeout, time.Second)
	}
	if srv.internalServer.WriteTimeout != 2*time.Second {
		t.Errorf("WriteTimeout = %v, want %v", srv.internalServer.WriteTimeout, 2*time.Second)
	}
	if srv.shutdownTimeout != 3*time.Second {
		t.Errorf("shutdownTimeout = %v, want %v", srv.shutdownTimeout, 3*time.Second)
	}
}

func TestServer_Start_SurfacesListenErrorOnChannelErr(t *testing.T) {
	t.Parallel()

	srv := NewServer(http.NewServeMux(), SetAddr("invalid-address-with-no-port"))

	select {
	case err, ok := <-srv.channelErr:
		if !ok {
			t.Fatalf("channelErr closed with no value, want a listen error")
		}
		if err == nil {
			t.Errorf("err = nil, want a non-nil listen error")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for the listen error")
	}

	select {
	case _, ok := <-srv.channelErr:
		if ok {
			t.Errorf("channelErr yielded a second value, want it closed after the first")
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("channelErr was not closed after the error")
	}
}

func TestServer_FullShutdownTimeout_Success(t *testing.T) {
	t.Parallel()

	srv := NewServer(http.NewServeMux(), SetAddr("127.0.0.1:0"), SetShutdownTimeout(5*time.Second))

	if err := srv.FullShutdownTimeout(discardLogger()); err != nil {
		t.Errorf("FullShutdownTimeout: %v", err)
	}
}

func TestServer_FullShutdownTimeout_ReturnsErrorWhenDeadlineExceeded(t *testing.T) {
	t.Parallel()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve a free port: %v", err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("release reserved port: %v", err)
	}

	block := make(chan struct{})
	handler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		<-block
	})

	srv := NewServer(handler, SetAddr(addr), SetShutdownTimeout(50*time.Millisecond))
	time.Sleep(50 * time.Millisecond)

	reqDone := make(chan struct{})
	go func() {
		defer close(reqDone)
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get("http://" + addr + "/")
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
	time.Sleep(50 * time.Millisecond)

	shutdownErr := srv.FullShutdownTimeout(discardLogger())
	close(block)
	<-reqDone

	if shutdownErr == nil {
		t.Fatalf("FullShutdownTimeout succeeded, want a deadline-exceeded error")
	}
	if !errors.Is(shutdownErr, context.DeadlineExceeded) {
		t.Errorf("err = %v, want context.DeadlineExceeded wrapped", shutdownErr)
	}
}

func TestServer_GracefulShutdown_OnTimeout(t *testing.T) {
	t.Parallel()

	srv := bareServer()
	srv.shutdownTimeout = 10 * time.Millisecond

	done := make(chan struct{})
	go func() {
		srv.GracefulShutdown(discardLogger())
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("GracefulShutdown did not return after its timeout elapsed")
	}
}

func TestServer_GracefulShutdown_OnChannelError_LogsTheRealError(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	srv := bareServer()
	srv.shutdownTimeout = time.Minute
	realErr := errors.New("listen boom")
	srv.channelErr <- realErr
	close(srv.channelErr)

	done := make(chan struct{})
	go func() {
		srv.GracefulShutdown(logger)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("GracefulShutdown did not return after a channelErr signal")
	}

	logged := buf.String()
	if !strings.Contains(logged, realErr.Error()) {
		t.Errorf("log = %q, want it to contain the real error %q", logged, realErr.Error())
	}
}
