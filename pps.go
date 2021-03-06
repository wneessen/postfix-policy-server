package pps

import (
	"bufio"
	"context"
	"fmt"
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
	// ctxConnId represents the connection id in the connection context
	ctxConnId CtxKey = iota

	// CtxNoLog lets the user control wether the server should log to
	// STDERR or not
	CtxNoLog
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

// PostfixTextResp is a possible response value that requires additional text
type PostfixTextResp string

// Possible non-optional text responses to the postfix server
const (
	TextRespFilter   PostfixTextResp = "FILTER"
	TextRespPrepend  PostfixTextResp = "PREPEND"
	TextRespRedirect PostfixTextResp = "REDIRECT"
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

	// postfix-policy-server specific values
	PPSConnId string
}

// connection represents an incoming policy server connection
type connection struct {
	conn net.Conn
	rs   *bufio.Scanner
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

// SetPort will override the listening port on an already existing policy server
func (s *Server) SetPort(p string) {
	s.lp = p
}

// SetAddr will override the listening address on an already existing policy server
func (s *Server) SetAddr(a string) {
	s.la = a
}

// Run starts a server based on the Server object
func (s *Server) Run(ctx context.Context, h Handler) error {
	sa := net.JoinHostPort(s.la, s.lp)
	l, err := net.Listen("tcp", sa)
	if err != nil {
		return err
	}
	return s.RunWithListener(ctx, h, l)
}

// RunWithListener starts a server based on the Server object with a given network listener
func (s *Server) RunWithListener(ctx context.Context, h Handler, l net.Listener) error {
	el := log.New(os.Stderr, "[Server] ERROR: ", log.Lmsgprefix|log.LstdFlags|log.Lshortfile)
	noLog := false
	ok, nlv := ctx.Value(CtxNoLog).(bool)
	if ok {
		noLog = nlv
	}

	go func() {
		<-ctx.Done()
		if err := l.Close(); err != nil && !noLog {
			el.Printf("failed to close listener: %s", err)
		}
	}()

	// Accept new connections
	for {
		c, err := l.Accept()
		if err != nil {
			if !noLog {
				el.Printf("failed to accept new connection: %s", err)
			}
			break
		}
		conn := &connection{
			conn: c,
			rs:   bufio.NewScanner(c),
			h:    h,
		}

		connId := xid.New()
		conCtx := context.WithValue(ctx, ctxConnId, connId)
		ec := make(chan error, 1)
		go func() { ec <- connHandler(conCtx, conn) }()
		select {
		case <-conCtx.Done():
			<-ec
			return ctx.Err()
		case err := <-ec:
			return err
		}
	}

	return nil
}

// connHandler processes the incoming policy connection request and hands it to the
// Handle function of the Handler interface
func connHandler(ctx context.Context, c *connection) error {
	connId, ok := ctx.Value(ctxConnId).(xid.ID)
	if !ok {
		return fmt.Errorf("failed to retrieve connection id from context")
	}

	for !c.cc {
		ps := &PolicySet{PPSConnId: connId.String()}
		processMsg(c, ps)
		if ps.Request != "" {
			resp := c.h.Handle(ps)
			if err := c.conn.SetWriteDeadline(time.Now().Add(time.Second)); err != nil {
				c.err = fmt.Errorf("failed to set write deadline on connection: %s", err.Error())
			}
			sResp := fmt.Sprintf("action=%s\n\n", resp)
			if _, err := c.conn.Write([]byte(sResp)); err != nil {
				c.err = fmt.Errorf("failed to write response on connection: %s", err.Error())
			}
		}
	}
	return c.err
}

// processMsg processes the incoming policy message and updates the given PolicySet
func processMsg(c *connection, ps *PolicySet) {
	for c.rs.Scan() {
		l := c.rs.Text()
		if l == "" {
			break
		}
		sl := strings.SplitN(l, "=", 2)
		if f, ok := polSetFuncs[sl[0]]; ok {
			f(ps, sl[1])
		}
	}
	if err := c.rs.Err(); err != nil {
		if _, ok := err.(*net.OpError); ok {
			return
		}
		c.err = err
	}
}

// TextResponseOpt allows you to use a PostfixResp with an optional text as response to the
// Postfix server
func TextResponseOpt(rt PostfixResp, t string) PostfixResp {
	r := PostfixResp(fmt.Sprintf("%s %s", rt, t))
	return r
}

// TextResponseNonOpt allows you to use a PostfixTextResp with a non-optional text as response to the
// Postfix server
func TextResponseNonOpt(rt PostfixTextResp, t string) PostfixResp {
	r := PostfixResp(fmt.Sprintf("%s %s", rt, t))
	return r
}
