package tcp

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/armon/go-proxyproto"
	"github.com/pkg/errors"
)

// Server is a scaffolding for basic TCP server. The structure members are
// configured externally, through JSON deserialization or environment variables
type Server struct {
	StopTimeout int    `default:"1000"`  // Stop timeout in milliseconds
	Protocol    string `default:"tcp"`   // Network protocol
	Address     string `default:":7777"` // Listen address
	Ready       func() `ignored:"true"`  // Function to call on server ready

	wg sync.WaitGroup
	l  net.Listener
}

// Handler is called on every accepted connection
type Handler func(net.Conn)

// Serve starts the server on the configured address
func (srv *Server) Serve(handler Handler) (err error) {
	if srv.l, err = net.Listen(srv.Protocol, srv.Address); err != nil {
		return errors.Wrap(err, "creating listener")
	}
	srv.l = &proxyproto.Listener{Listener: srv.l}
	if srv.Ready != nil {
		srv.Ready()
	}
	conns := make(chan net.Conn)
	go func() {
		defer close(conns)
		for {
			var conn net.Conn
			if conn, err = srv.l.Accept(); err != nil {
				err = errors.Wrap(err, "accepting connection")
				return
			}
			conns <- conn
		}
	}()

	for conn := range conns {
		srv.wg.Add(1)
		go func(conn net.Conn) {
			defer srv.wg.Done()
			handler(conn)
		}(conn)
	}
	return
}

// Close stops the server gracefully
func (srv *Server) Close() (err error) {
	done := make(chan struct{})
	go func() {
		srv.wg.Wait()
		close(done)
	}()
	ctx, cancel := context.WithTimeout(context.TODO(), time.Duration(srv.StopTimeout)*time.Millisecond)
	defer cancel()

	err = srv.l.Close()
	select {
	case <-ctx.Done():
		err = errors.Wrapf(ctx.Err(), "stopping handlers")
	case <-done:
		break
	}
	return
}
