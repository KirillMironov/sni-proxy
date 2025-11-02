package main

import "net"

type UpstreamType string

const (
	UpstreamTypeHttpProxy UpstreamType = "http-proxy"
	UpstreamTypeVLESS                  = "vless"
)

type Upstream interface {
	Connect(sni string) (net.Conn, error)
	Close() error
}

type UpstreamConfig interface {
	Type() UpstreamType
	Validate() error
}
