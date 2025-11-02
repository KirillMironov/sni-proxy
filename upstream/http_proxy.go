package upstream

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

type HttpProxyConfig struct {
	Address  string `envconfig:"HTTP_PROXY_ADDRESS"`
	Username string `envconfig:"HTTP_PROXY_USERNAME"`
	Password string `envconfig:"HTTP_PROXY_PASSWORD"`
}

type HttpProxy struct {
	config HttpProxyConfig
}

func NewHttpProxy(config HttpProxyConfig) *HttpProxy {
	return &HttpProxy{config: config}
}

func (h *HttpProxy) Connect(sni string, timeout time.Duration) (net.Conn, error) {
	// dial upstream HTTP proxy
	upstreamConn, err := net.Dial("tcp", h.config.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to upstream proxy: %v", err)
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

	if err = upstreamConn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return nil, fmt.Errorf("failed to set upstream timeout: %v", err)
	}

	if err = connectReq.Write(upstreamConn); err != nil {
		return nil, fmt.Errorf("failed to write connect request to upstream proxy: %v", err)
	}

	// read upstream proxy response
	resp, err := http.ReadResponse(bufio.NewReader(upstreamConn), connectReq)
	if err != nil {
		return nil, fmt.Errorf("failed to read response from upstream proxy: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream proxy rejected connect request, code: %d", resp.StatusCode)
	}

	return upstreamConn, nil
}

func (h *HttpProxy) Close() error {
	return nil
}
