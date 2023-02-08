package main

import (
	"github.com/noetarbouriech/go-jitsu/server"
)

const (
	host = "localhost"
	port = 3000
)

func main() {
	server.InitServer(host, port)
}
