package postfix_policy_server

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Connection represents an incoming policy server connection
type Connection struct {
	conn net.Conn
	rb   *bufio.Reader
	wb   *bufio.Writer
	h    Handler
	err  error
	cc   bool
}

// Server defines a new policy server with corresponding settings
type Server struct {
	lp string
	la string
}

// Handler interface for handling incoming policy requests and returning the
// corresponding action
type Handler interface {
	Handle() string
}

// ServerOpt is an override function for the New() method
type ServerOpt func(*Server)

// New returns a new server object
func New(options ...ServerOpt) Server {
	s := Server{
		lp: "12346",
		la: "0.0.0.0",
	}
	for _, o := range options {
		if o == nil {
			continue
		}
		o(&s)
	}

	return s
}

// WithPort overrides the default listening port for the policy server
func WithPort(p string) ServerOpt {
	return func(s *Server) {
		s.lp = p
	}
}

// WithAddr overrides the default listening address for the policy server
func WithAddr(a string) ServerOpt {
	return func(s *Server) {
		s.la = a
	}
}

// Run starts a server based on the Server object
func (s *Server) Run(ctx context.Context, h Handler) error {
	el := log.New(os.Stderr, "[Server] ERROR: ", log.Lmsgprefix|log.LstdFlags)
	sa := net.JoinHostPort(s.la, s.lp)
	l, err := net.Listen("tcp", sa)
	if err != nil {
		return err
	}
	defer func() {
		if err := l.Close(); err != nil {
			el.Printf("failed to close listener: %s", err)
		}
	}()

	// Accept new connections
	for {
		c, err := l.Accept()
		if err != nil {
			el.Printf("failed to accept new connection: %s", err)
		}
		conn := &Connection{
			conn: c,
			rb:   bufio.NewReader(c),
			wb:   bufio.NewWriter(c),
			h:    h,
		}

		connId, err := uuid.NewUUID()
		if err != nil {
			el.Printf("failed to generate UUID: %s", err)
		}
		conCtx := context.WithValue(ctx, "id", connId.String())
		go connHandler(conCtx, conn)
	}
}

// connHandler processes the incoming policy connection request and hands it to the
// Handle function of the Handler interface
func connHandler(ctx context.Context, c *Connection) {
	connId := ctx.Value("id").(string)
	cl := log.New(os.Stderr, fmt.Sprintf("[%s] ERROR: ", connId), log.Lmsgprefix|log.LstdFlags)
	cl.Printf("new connection from %s", c.conn.RemoteAddr())

	done := make(chan bool)
	defer close(done)

	// Make sure to close the connection when our context is done
	go func() {
		select {
		case <-ctx.Done():
		case <-done:
			if c.err != nil {
				cl.Printf("closing connection due to an unexpected error: ", c.err)
			}
		}

		cl.Print("closing connection...")
		if err := c.conn.Close(); err != nil {
			cl.Printf("failed to close connection: %s", err)
		}
		c.cc = true
	}()

	for !c.cc {
		kvMap := make(map[string]string, 0)
		for {
			l, err := c.rb.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					done <- true
					break
				}
				if _, ok := err.(*net.OpError); ok {
					break
				}
				c.err = err
				done <- true
			}
			l = strings.TrimRight(l, "\n\n")
			if l == "" {
				break
			}
			sl := strings.Split(l, "=")
			if len(sl) != 2 {
				continue
			}
			kvMap[sl[0]] = sl[1]
		}

		if (len(kvMap)) > 0 {
			cl.Printf("%+v", kvMap)
			if err := c.conn.SetWriteDeadline(time.Now().Add(time.Second)); err != nil {
				c.err = fmt.Errorf("failed to set write deadline on connection: %s", err.Error())
				done <- true
			}
			if _, err := c.conn.Write([]byte("action=DUNNO\n\n")); err != nil {
				c.err = fmt.Errorf("failed to write respone on connection: %s", err.Error())
				done <- true
			}
		}
		//foo := c.h.Handle()
	}
}
