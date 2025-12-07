package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"

	"git.capy.fun/sni-proxy/config"
	"git.capy.fun/sni-proxy/upstream"
)

type Proxy struct {
	config   config.ProxyConfig
	upstream Upstream
}

type Upstream interface {
	Connect(sni string, timeout time.Duration) (net.Conn, error)
	Close() error
}

func NewProxy(config config.ProxyConfig) *Proxy {
	return &Proxy{config: config}
}

func (p *Proxy) Init() error {
	var up Upstream

	switch p.config.UpstreamType {
	case config.UpstreamTypeHttpProxy:
		up = upstream.NewHttpProxy(p.config.HttpProxyConfig)
	case config.UpstreamTypeSSH:
		up = upstream.NewSSH(p.config.SSHConfig)
	case config.UpstreamTypeVLESSReality:
		up = upstream.NewVlessReality(p.config.VLESSRealityConfig)
	case "":
		return errors.New("upstream type not specified")
	default:
		return fmt.Errorf("unsupported upstream type: %s", p.config.UpstreamType)
	}

	p.upstream = up

	return nil
}

func (p *Proxy) Handle(ctx context.Context, conn net.Conn, sni string, reader io.Reader) {
	// dial upstream
	upstreamConn, err := p.upstream.Connect(sni, p.config.UpstreamTimeout)
	if err != nil {
		slog.ErrorContext(ctx, "failed to connect to upstream", slog.Any("error", err))
		return
	}
	defer upstreamConn.Close()

	var wg sync.WaitGroup
	wg.Go(func() { _, _ = io.Copy(conn, upstreamConn) })
	wg.Go(func() { _, _ = io.Copy(upstreamConn, reader) })
	wg.Wait()
}
