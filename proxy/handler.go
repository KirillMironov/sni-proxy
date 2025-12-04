package proxy

import (
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

type Handler struct {
	config   config.ProxyConfig
	upstream Upstream
}

type Upstream interface {
	Connect(sni string, timeout time.Duration) (net.Conn, error)
	Close() error
}

func NewHandler(config config.ProxyConfig) *Handler {
	return &Handler{config: config}
}

func (h *Handler) Init() error {
	var up Upstream

	switch h.config.UpstreamType {
	case config.UpstreamTypeHttpProxy:
		up = upstream.NewHttpProxy(h.config.HttpProxyConfig)
	case config.UpstreamTypeSSH:
		up = upstream.NewSSH(h.config.SSHConfig)
	case config.UpstreamTypeVLESSReality:
		up = upstream.NewVlessReality(h.config.VLESSRealityConfig)
	case "":
		return errors.New("upstream type not specified")
	default:
		return fmt.Errorf("unsupported upstream type: %s", h.config.UpstreamType)
	}

	h.upstream = up

	return nil
}

func (h *Handler) Handle(conn net.Conn, sni string, reader io.Reader) {
	// dial upstream
	upstreamConn, err := h.upstream.Connect(sni, h.config.UpstreamTimeout)
	if err != nil {
		slog.Error("failed to connect to upstream", slog.Any("error", err))
		return
	}
	defer upstreamConn.Close()

	var wg sync.WaitGroup
	wg.Go(func() { _, _ = io.Copy(conn, upstreamConn) })
	wg.Go(func() { _, _ = io.Copy(upstreamConn, reader) })
	wg.Wait()
}
