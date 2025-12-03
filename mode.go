package main

import (
	"io"
	"net"
)

type Mode string

const (
	ModeProxy  Mode = "proxy"
	ModeBypass Mode = "bypass"
)

type ConnectionHandler interface {
	Init() error
	Handle(conn net.Conn, sni string, reader io.Reader)
}
