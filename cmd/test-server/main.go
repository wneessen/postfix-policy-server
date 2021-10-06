package main

import (
	"context"
	"fmt"
	"github.com/wneessen/postfix-policy-server"
	"log"
)

// Hi is an empty struct to work as the Handler interface
type Hi struct{}

// Handle is the test handler for the test server as required by the Handler interface
func (h Hi) Handle(ps *pps.PolicySet) pps.PostfixResp {
	fmt.Printf("All Data: %+v\n", ps)

	return pps.RespReject
}

// main executes the test server
func main() {
	s := pps.New()
	ctx := context.Background()
	h := Hi{}

	if err := s.Run(ctx, h); err != nil {
		log.Fatalf("could not run server: %s", err)
	}
}
