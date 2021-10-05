package main

import (
	"context"
	"fmt"
	"github.com/wneessen/postfix-policy-server"
	"log"
)

type Hn struct {
}

func main() {
	s := pps.New()
	ctx := context.Background()
	h := Hn{}

	if err := s.Run(ctx, h); err != nil {
		log.Fatalf("could not run server: %s", err)
	}
}

func (h Hn) Handle(ps *pps.PolicySet) pps.PostfixResp {
	fmt.Printf("All Data: %+v\n", ps)

	return pps.RespReject
}
