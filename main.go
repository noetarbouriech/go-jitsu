package main

import (
	"github.com/noetarbouriech/go-jitsu/server"
)

const (
	host = "0.0.0.0"
	port = 3000
)

func main() {
	server.InitServer(host, port)
}
