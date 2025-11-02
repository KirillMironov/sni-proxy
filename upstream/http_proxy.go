package upstream

import (
	"bufio"
	"encoding/base64"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"time"
)

type HttpProxyConfig struct {
	Address  string        `envconfig:"HTTP_PROXY_ADDRESS" required:"true"`
	Username string        `envconfig:"HTTP_PROXY_USERNAME" required:"true"`
	Password string        `envconfig:"HTTP_PROXY_PASSWORD" required:"true"`
	Timeout  time.Duration `envconfig:"HTTP_PROXY_TIMEOUT" default:"5s"`
}

type HttpProxy struct {
	config HttpProxyConfig
	logger *slog.Logger
}

func NewHttpProxy(config HttpProxyConfig) *HttpProxy {
	return &HttpProxy{
		config: config,
		logger: slog.With(slog.String("upstream", "http-proxy")),
	}
}

func (h *HttpProxy) Connect(sni string) (net.Conn, error) {
	// dial upstream HTTP proxy
	upstreamConn, err := net.Dial("tcp", h.config.Address)
	if err != nil {
		h.logger.Error("failed to connect to upstream proxy", slog.Any("error", err))
		return nil, err
	}

	// send CONNECT request to upstream proxy with Basic Auth
	credentials := h.config.Username + ":" + h.config.Password
	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(credentials))

	connectReq := &http.Request{
		URL:    new(url.URL),
		Method: http.MethodConnect,
		Host:   net.JoinHostPort(sni, "443"),
		Header: http.Header{
			"Proxy-Authorization": []string{authHeader},
		},
	}

	if err = upstreamConn.SetReadDeadline(time.Now().Add(h.config.Timeout)); err != nil {
		h.logger.Error("failed to set upstream timeout", slog.Any("error", err))
		return nil, err
	}

	if err = connectReq.Write(upstreamConn); err != nil {
		h.logger.Error("failed to write connect request to upstream proxy", slog.Any("error", err))
		return nil, err
	}

	// read upstream proxy response
	resp, err := http.ReadResponse(bufio.NewReader(upstreamConn), connectReq)
	if err != nil {
		h.logger.Error("failed to read response from upstream proxy", slog.Any("error", err))
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		h.logger.Error("upstream proxy rejected connect request", slog.Int("code", resp.StatusCode))
		return nil, err
	}

	return upstreamConn, nil
}

func (h *HttpProxy) Close() error {
	return nil
}
