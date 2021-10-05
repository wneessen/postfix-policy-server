package main

import (
	"context"
	"github.com/wneessen/postfix-policy-server"
	"log"
)

type Hn struct {
}

func main() {
	s := postfix_policy_server.New()
	ctx := context.Background()
	h := Hn{}

	if err := s.Run(ctx, h); err != nil {
		log.Fatalf("could not run server: %s", err)
	}
}

func (h Hn) Handle() string {
	return ""
}
