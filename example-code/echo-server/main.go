package main

/*
	This code example implements a simple echo server using the postfix-policy-server framework.
	It creates a new policy server that listens for incoming policy requests from postfix on
	the default port (10005) and returns a JSON representation of the received PolicySet{} values
	to STDOUT

	To integrate this test server with your postfix configuration, you simply have to add
	"check_policy_service inet:127.0.0.1:10005" to the "smtpd_recipient_restrictions" of
	your postfix' main.cf and reload postfix.

	Example:

		smtpd_recipient_restrictions =
			[...]
			reject_unauth_destination
			check_policy_service inet:127.0.0.1:10005
			[...]
*/

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/wneessen/postfix-policy-server"
	"log"
)

// Hi is an empty struct to work as the Handler interface
type Hi struct{}

// Handle is the test handler for the test server as required by the Handler interface
func (h Hi) Handle(ps *pps.PolicySet) pps.PostfixResp {
	log.Println("received new policy set...")
	jps, err := json.Marshal(ps)
	if err != nil {
		log.Printf("failed to marshal policy set data: %s", err)
		return pps.RespWarn
	}
	fmt.Println(string(jps))
	return pps.RespDunno
}

// main starts the server
func main() {
	s := pps.New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h := Hi{}
	log.Println("Starting policy echo server...")
	if err := s.Run(ctx, h); err != nil {
		log.Fatalf("could not run server: %s", err)
	}
}
