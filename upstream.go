package main

import (
	"net"
	"time"
)

type UpstreamType string

const (
	UpstreamTypeHttpProxy UpstreamType = "http-proxy"
	UpstreamTypeSSH       UpstreamType = "ssh"
)

type Upstream interface {
	Connect(sni string, timeout time.Duration) (net.Conn, error)
	Close() error
}

type UpstreamConfig interface {
	Type() UpstreamType
	Validate() error
}
