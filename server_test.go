package tcp

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/pkg/errors"
)

func TestServer(t *testing.T) {
	var counter int
	ready := make(chan struct{})
	srv := &Server{
		StopTimeout: 1000,
		Protocol:    "tcp",
		Address:     "127.1.2.3:45678",
		Ready:       func() { close(ready) },
	}

	var serverError error
	go func() {
		serverError = errors.Cause(srv.Serve(
			func(conn net.Conn) {
				t.Log("client connected")
				counter++
				time.Sleep(3000 * time.Millisecond)
				counter++
			}))
	}()

	<-ready
	client, err := net.Dial(srv.Protocol, srv.Address)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	defer client.Close()
	time.Sleep(500 * time.Millisecond)
	if err = errors.Cause(srv.Close()); err != context.DeadlineExceeded {
		t.Errorf("unexpected error closing: %+v", err)
	}
	if _, ok := serverError.(net.Error); !ok {
		t.Errorf("unexpected error serving: %+v", err)
	}
	if counter != 1 {
		t.Errorf("unexpected counter value: %d", counter)
	}
}
