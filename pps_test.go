package pps

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"testing"
	"time"
)

// Empty struct to test the Handler interface
type Hi struct{}

// Handle is the function required by the Handler Interface
func (h Hi) Handle(*PolicySet) PostfixResp {
	return RespDunno
}

const exampleReq = `request=smtpd_access_policy
protocol_state=RCPT
protocol_name=SMTP
client_address=127.0.0.1
client_name=localhost
client_port=45140
reverse_client_name=localhost
server_address=127.0.0.1
server_port=25
helo_name=example.com
sender=tester@example.com
recipient=tester@localhost.tld
recipient_count=0
queue_id=
instance=1234.5678910a.bcdef.0
size=0
etrn_domain=
stress=
sasl_method=
sasl_username=
sasl_sender=
ccert_subject=
ccert_issuer=
ccert_fingerprint=
ccert_pubkey_fingerprint=
encryption_protocol=
encryption_cipher=
encryption_keysize=0
policy_context=

`

// TestNew tests the New() method
func TestNew(t *testing.T) {
	s := New()
	if s.lp != DefaultPort {
		t.Errorf("policy server ceation failed: configured port mismatch => Expected: %s, got: %s",
			DefaultPort, s.lp)
	}
	if s.la != DefaultAddr {
		t.Errorf("policy server ceation failed: configured listen address mismatch => Expected: %s, got: %s",
			DefaultAddr, s.la)
	}
}

// TestNewWithAddr tests the New() method with the WithAddr() option
func TestNewWithAddr(t *testing.T) {
	a := "1.2.3.4"
	s := New(WithAddr(a))
	if s.la != a {
		t.Errorf("policy server ceation failed: configured listen address mismatch => Expected: %s, got: %s",
			a, s.la)
	}
}

// TestNewWithPort tests the New() method with the WithPort() option
func TestNewWithPort(t *testing.T) {
	p := "1234"
	s := New(WithPort(p))
	if s.lp != p {
		t.Errorf("policy server ceation failed: configured listen address mismatch => Expected: %s, got: %s",
			p, s.lp)
	}
}

// TestNewWithEmptyOpt tests the New() method with a nil-option
func TestNewWithEmptyOpt(t *testing.T) {
	emptyOpt := func(p *string) ServerOpt { return nil }
	s := New(emptyOpt(nil))
	if s.lp != DefaultPort {
		t.Errorf("policy server ceation failed: configured listen address mismatch => Expected: %s, got: %s",
			DefaultPort, s.lp)
	}
}

// TestRun starts a new server listening for connections
func TestRun(t *testing.T) {
	testTable := []struct {
		testName   string
		listenAddr string
		listenPort string
		shouldFail bool
	}{
		{`Successfully start with defaults`, DefaultAddr, DefaultPort, false},
		{`Fail to on invalid IP`, "256.256.256.256", DefaultPort, true},
		{`Fail to on invalid port`, DefaultAddr, "1", true},
	}

	for _, tc := range testTable {
		t.Run(tc.testName, func(t *testing.T) {
			s := New(WithAddr(tc.listenAddr), WithPort(tc.listenPort))
			ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*200)
			defer cancel()

			h := Hi{}
			err := s.Run(ctx, h)
			if err != nil && !tc.shouldFail {
				t.Errorf("could not run server: %s", err)
			}
		})
	}
}

// TestRunDial starts a new server listening for connections and tries to connect to it
func TestRunDial(t *testing.T) {
	s := New()
	sctx, scancel := context.WithCancel(context.Background())
	defer scancel()

	h := Hi{}
	go func() {
		if err := s.Run(sctx, h); err != nil {
			t.Errorf("could not run server: %s", err)
		}
	}()

	// Wait a brief moment for the server to start
	time.Sleep(time.Millisecond * 200)

	d := net.Dialer{}
	cctx, ccancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer ccancel()
	conn, err := d.DialContext(cctx, "tcp",
		fmt.Sprintf("%s:%s", s.la, s.lp))
	if err != nil {
		t.Errorf("failed to connect to running server: %s", err)
		return
	}
	if err := conn.Close(); err != nil {
		t.Errorf("failed to close client connection: %s", err)
	}

	// Wait a brief moment for the connection to close
	time.Sleep(time.Millisecond * 500)
}

// TestRunDialWithRequest starts a new server listening for connections and tries to connect to it
// and sends example data
func TestRunDialWithRequest(t *testing.T) {
	s := New()
	sctx, scancel := context.WithCancel(context.Background())
	defer scancel()

	h := Hi{}
	go func() {
		if err := s.Run(sctx, h); err != nil {
			t.Errorf("could not run server: %s", err)
		}
	}()

	// Wait a brief moment for the server to start
	time.Sleep(time.Millisecond * 200)

	d := net.Dialer{}
	cctx, ccancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer ccancel()
	conn, err := d.DialContext(cctx, "tcp",
		fmt.Sprintf("%s:%s", s.la, s.lp))
	if err != nil {
		t.Errorf("failed to connect to running server: %s", err)
		return
	}
	defer func() { _ = conn.Close() }()
	rb := bufio.NewReader(conn)
	_, err = conn.Write([]byte(exampleReq))
	if err != nil {
		t.Errorf("failed to send request to server: %s", err)
	}
	resp, err := rb.ReadString('\n')
	if err != nil {
		t.Errorf("failed to read response from server: %s", err)
	}
	exresp := fmt.Sprintf("action=%s\n", RespDunno)
	if resp != exresp {
		t.Errorf("unexpected server response => expected: %s, got: %s", exresp, resp)
	}
}