package proxy

import (
	"net"
	"time"

	"git.capy.fun/sni-proxy/upstream"
)

type UpstreamType string

const (
	UpstreamTypeHttpProxy    UpstreamType = "http-proxy"
	UpstreamTypeSSH          UpstreamType = "ssh"
	UpstreamTypeVLESSReality UpstreamType = "vless-reality"
)

type Upstream interface {
	Connect(sni string, timeout time.Duration) (net.Conn, error)
	Close() error
}

type UpstreamConfig struct {
	Type               UpstreamType  `envconfig:"UPSTREAM_TYPE"`
	Timeout            time.Duration `envconfig:"UPSTREAM_TIMEOUT" default:"5s"`
	HttpProxyConfig    upstream.HttpProxyConfig
	SSHConfig          upstream.SSHConfig
	VLESSRealityConfig upstream.VLESSRealityConfig
}
