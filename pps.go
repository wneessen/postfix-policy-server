package pps

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rs/xid"
)

// DefaultAddr is the default address the server is listening on
const DefaultAddr = "0.0.0.0"

// DefaultPort is the default port the server is listening on
const DefaultPort = "10005"

// CtxKey represents the different key ids for values added to contexts
type CtxKey int

const (
	// CtxConnId represents the connection id in the connection context
	CtxConnId CtxKey = iota
)

// PostfixResp is a possible response value for the policy request
type PostfixResp string

// Possible responses to the postfix server
// See: http://www.postfix.org/access.5.html
const (
	RespOk            PostfixResp = "OK"
	RespReject        PostfixResp = "REJECT"
	RespDefer         PostfixResp = "DEFER"
	RespDeferIfReject PostfixResp = "DEFER_IF_REJECT"
	RespDeferIfPermit PostfixResp = "DEFER_IF_PERMIT"
	RespDiscard       PostfixResp = "DISCARD"
	RespDunno         PostfixResp = "DUNNO"
	RespHold          PostfixResp = "HOLD"
	RespInfo          PostfixResp = "INFO"
	RespWarn          PostfixResp = "WARN"
)

// polSetFuncs is a map of polSetFunc that assigns a given value to a PolicySet
// See http://www.postfix.org/SMTPD_POLICY_README.html for all supported values
var polSetFuncs = map[string]polSetFunc{
	"request":        func(ps *PolicySet, v string) { ps.Request = v },
	"protocol_state": func(ps *PolicySet, v string) { ps.ProtocolState = v },
	"protocol_name":  func(ps *PolicySet, v string) { ps.ProtocolName = v },
	"helo_name":      func(ps *PolicySet, v string) { ps.HELOName = v },
	"queue_id":       func(ps *PolicySet, v string) { ps.QueueId = v },
	"sender":         func(ps *PolicySet, v string) { ps.Sender = v },
	"recipient":      func(ps *PolicySet, v string) { ps.Recipient = v },
	"recipient_count": func(ps *PolicySet, v string) {
		rc, err := strconv.ParseUint(v, 10, 64)
		if err == nil {
			ps.RecipientCount = rc
		}
	},
	"client_address": func(ps *PolicySet, v string) {
		ca := net.ParseIP(v)
		ps.ClientAddress = ca
	},
	"client_name":         func(ps *PolicySet, v string) { ps.ClientName = v },
	"reverse_client_name": func(ps *PolicySet, v string) { ps.ReverseClientName = v },
	"instance":            func(ps *PolicySet, v string) { ps.Instance = v },
	"sasl_method":         func(ps *PolicySet, v string) { ps.SASLMethod = v },
	"sasl_username":       func(ps *PolicySet, v string) { ps.SASLUsername = v },
	"sasl_sender":         func(ps *PolicySet, v string) { ps.SASLSender = v },
	"size": func(ps *PolicySet, v string) {
		s, err := strconv.ParseUint(v, 10, 64)
		if err == nil {
			ps.Size = s
		}
	},
	"ccert_subject":       func(ps *PolicySet, v string) { ps.CCertSubject = v },
	"ccert_issuer":        func(ps *PolicySet, v string) { ps.CCertIssuer = v },
	"ccert_fingerprint":   func(ps *PolicySet, v string) { ps.CCertFingerprint = v },
	"encryption_protocol": func(ps *PolicySet, v string) { ps.EncryptionProtocol = v },
	"encryption_cipher":   func(ps *PolicySet, v string) { ps.EncryptionCipher = v },
	"encryption_keysize": func(ps *PolicySet, v string) {
		ks, err := strconv.ParseUint(v, 10, 64)
		if err == nil {
			ps.EncryptionKeysize = ks
		}
	},
	"etrn_domain":              func(ps *PolicySet, v string) { ps.ETRNDomain = v },
	"stress":                   func(ps *PolicySet, v string) { ps.Stress = v == "yes" },
	"ccert_pubkey_fingerprint": func(ps *PolicySet, v string) { ps.CCertPubkeyFingerprint = v },
	"client_port": func(ps *PolicySet, v string) {
		cp, err := strconv.ParseUint(v, 10, 64)
		if err == nil {
			ps.ClientPort = cp
		}
	},
	"policy_context": func(ps *PolicySet, v string) { ps.PolicyContext = v },
	"server_address": func(ps *PolicySet, v string) {
		sa := net.ParseIP(v)
		ps.ServerAddress = sa
	},
	"server_port": func(ps *PolicySet, v string) {
		sp, err := strconv.ParseUint(v, 10, 64)
		if err == nil {
			ps.ServerPort = sp
		}
	},
}

// PolicySet is a set information provided by the postfix policyd request
type PolicySet struct {
	// Postfix version 2.1 and later
	Request           string
	ProtocolState     string
	ProtocolName      string
	HELOName          string
	QueueId           string
	Sender            string
	Recipient         string
	RecipientCount    uint64
	ClientAddress     net.IP
	ClientName        string
	ReverseClientName string
	Instance          string

	// Postfix version 2.2 and later
	SASLMethod       string
	SASLUsername     string
	SASLSender       string
	Size             uint64
	CCertSubject     string
	CCertIssuer      string
	CCertFingerprint string

	// Postfix version 2.3 and later
	EncryptionProtocol string
	EncryptionCipher   string
	EncryptionKeysize  uint64
	ETRNDomain         string

	// Postfix version 2.5 and later
	Stress bool

	// Postfix version 2.9 and later
	CCertPubkeyFingerprint string

	// Postfix version 3.0 and later
	ClientPort uint64

	// Postfix version 3.1 and later
	PolicyContext string

	// Postfix version 3.2 and later
	ServerAddress net.IP
	ServerPort    uint64
}

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

// polSetFunc is a function alias that tries to fit a given value into a PolicySet
type polSetFunc func(*PolicySet, string)

// ServerOpt is an override function for the New() method
type ServerOpt func(*Server)

// Handler interface for handling incoming policy requests and returning the
// corresponding action
type Handler interface {
	Handle(*PolicySet) PostfixResp
}

// New returns a new server object
func New(options ...ServerOpt) Server {
	s := Server{
		lp: DefaultPort,
		la: DefaultAddr,
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
	go func() {
		<-ctx.Done()
		if err := l.Close(); err != nil {
			el.Printf("failed to close listener: %s", err)
		}
	}()

	// Accept new connections
	for {
		c, err := l.Accept()
		if err != nil {
			el.Printf("failed to accept new connection: %s", err)
			break
		}
		conn := &Connection{
			conn: c,
			rb:   bufio.NewReader(c),
			wb:   bufio.NewWriter(c),
			h:    h,
		}

		connId := xid.New()
		conCtx := context.WithValue(ctx, CtxConnId, connId)
		go connHandler(conCtx, conn)
	}

	return nil
}

// connHandler processes the incoming policy connection request and hands it to the
// Handle function of the Handler interface
func connHandler(ctx context.Context, c *Connection) {
	connId, ok := ctx.Value(CtxConnId).(xid.ID)
	if !ok {
		log.Print("failed to retrieve connection id from context.")
		return
	}
	cl := log.New(os.Stderr, fmt.Sprintf("[%s] ERROR: ", connId.String()),
		log.Lmsgprefix|log.LstdFlags)

	// Channel to close connection in case of an error
	cc := make(chan bool)
	defer close(cc)

	// Make sure to close the connection when our context is cc
	go func() {
		select {
		case <-ctx.Done():
		case <-cc:
			if c.err != nil {
				cl.Printf("closing connection due to an unexpected error: %s", c.err)
			}
		}
		if err := c.conn.Close(); err != nil {
			cl.Printf("failed to close connection: %s", err)
		}
		c.cc = true
	}()

	for !c.cc {
		ps := &PolicySet{}
		for {
			l, err := c.rb.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					cc <- true
					break
				}
				if _, ok := err.(*net.OpError); ok {
					break
				}
				c.err = err
				cc <- true
			}
			l = strings.TrimRight(l, "\n")
			if l == "" {
				break
			}
			sl := strings.Split(l, "=")
			if len(sl) != 2 {
				continue
			}
			if f, ok := polSetFuncs[sl[0]]; ok {
				f(ps, sl[1])
			}
		}

		if ps.Request != "" {
			resp := c.h.Handle(ps)
			if err := c.conn.SetWriteDeadline(time.Now().Add(time.Second)); err != nil {
				c.err = fmt.Errorf("failed to set write deadline on connection: %s", err.Error())
				cc <- true
			}
			sResp := fmt.Sprintf("action=%s\n\n", resp)
			if _, err := c.conn.Write([]byte(sResp)); err != nil {
				c.err = fmt.Errorf("failed to write response on connection: %s", err.Error())
				cc <- true
			}
		}
	}
}
