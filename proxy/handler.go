package proxy

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/kelseyhightower/envconfig"

	"git.capy.fun/sni-proxy/upstream"
)

type Handler struct {
	clientHelloTimeout time.Duration

	upstreamConfig UpstreamConfig
	upstream       Upstream
}

func NewHandler(clientHelloTimeout time.Duration) *Handler {
	return &Handler{clientHelloTimeout: clientHelloTimeout}
}

func (h *Handler) Init() error {
	var config UpstreamConfig
	if err := envconfig.Process("", &config); err != nil {
		return err
	}

	var up Upstream

	switch config.Type {
	case UpstreamTypeHttpProxy:
		up = upstream.NewHttpProxy(config.HttpProxyConfig)
	case UpstreamTypeSSH:
		up = upstream.NewSSH(config.SSHConfig)
	case UpstreamTypeVLESSReality:
		up = upstream.NewVlessReality(config.VLESSRealityConfig)
	case "":
		return errors.New("upstream type not specified")
	default:
		return fmt.Errorf("unsupported upstream type: %s", config.Type)
	}

	h.upstreamConfig = config
	h.upstream = up

	return nil
}

func (h *Handler) Handle(conn net.Conn, sni string, reader io.Reader) {
	// dial upstream
	upstreamConn, err := h.upstream.Connect(sni, h.upstreamConfig.Timeout)
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
