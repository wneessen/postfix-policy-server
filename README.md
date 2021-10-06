# postfix-policy-server
[![Go Reference](https://pkg.go.dev/badge/github.com/wneessen/postfix-policy-server.svg)](https://pkg.go.dev/github.com/wneessen/postfix-policy-server) [![Go Report Card](https://goreportcard.com/badge/github.com/wneessen/postfix-policy-server)](https://goreportcard.com/report/github.com/wneessen/postfix-policy-server) [![Build Status](https://api.cirrus-ci.com/github/wneessen/postfix-policy-server.svg)](https://cirrus-ci.com/github/wneessen/postfix-policy-server) <a href="https://ko-fi.com/D1D24V9IX"><img src="https://uploads-ssl.webflow.com/5c14e387dab576fe667689cf/5cbed8a4ae2b88347c06c923_BuyMeACoffee_blue.png" height="20" alt="buy ma a coffee"></a>

`postfix-policy-server` (pps) provides a simple framework to create 
[Postfix SMTP Access Policy Delegation Servers](http://www.postfix.org/SMTPD_POLICY_README.html) in Go.

## Usage
The `pps` framework allows you to start a new TCP server that listens for incoming policy requests from
a Postfix mail server. Once a new connection is established and the dataset from Postfix has been sent, 
the data will be processed as a [PolicySet](https://pkg.go.dev/github.com/wneessen/postfix-policy-server#PolicySet) 
and handed to a provided `Handle()` method that is provided by the user, using `pps`'s 
[Handler](https://pkg.go.dev/github.com/wneessen/postfix-policy-server#Handler) interface.

Check out the [Go reference](https://pkg.go.dev/github.com/wneessen/postfix-policy-server) for further
details or have a look at the example [echo-server](example-code/echo-server) that is provided with this package.