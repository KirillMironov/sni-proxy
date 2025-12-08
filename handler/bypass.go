package handler

import (
	"context"
	"crypto/tls"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"

	"git.capy.fun/sni-proxy/config"
)

type Bypass struct {
	config config.BypassConfig

	// custom resolver to avoid dns loops
	resolver *net.Resolver
}

func NewBypass(config config.BypassConfig) *Bypass {
	return &Bypass{config: config}
}

func (b *Bypass) Init() error {
	b.resolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := &net.Dialer{
				Timeout: 5 * time.Second,
			}

			tlsConfig := &tls.Config{
				ServerName: "cloudflare-dns.com",
			}

			return tls.DialWithDialer(d, "tcp", "1.1.1.1:853", tlsConfig)
		},
	}

	return nil
}

func (b *Bypass) Handle(ctx context.Context, conn net.Conn, sni string, reader io.Reader) {
	// resolve upstream
	ips, err := b.resolver.LookupHost(context.Background(), sni)
	if err != nil || len(ips) == 0 {
		slog.ErrorContext(ctx, "dns lookup failed", slog.String("sni", sni), slog.Any("error", err))
		return
	}
	target := net.JoinHostPort(ips[0], "443")

	// dial upstream
	targetConn, err := net.DialTimeout("tcp", target, 5*time.Second)
	if err != nil {
		slog.ErrorContext(ctx, "dial failed", slog.Any("error", err))
		return
	}
	defer targetConn.Close()

	clientHelloBuf := make([]byte, b.config.ClientHello.BufferSize)
	n, err := reader.Read(clientHelloBuf)
	if err != nil {
		return
	}
	clientHelloData := clientHelloBuf[:n]

	// write ClientHello packets in chunks
	for i := 0; i < len(clientHelloData); i += int(b.config.ClientHello.ChunkSize) {
		end := i + int(b.config.ClientHello.ChunkSize)
		if end > len(clientHelloData) {
			end = len(clientHelloData)
		}

		chunk := clientHelloData[i:end]

		_, err = targetConn.Write(chunk)
		if err != nil {
			slog.ErrorContext(ctx, "failed to write chunk", slog.Any("error", err))
			return
		}
	}

	var wg sync.WaitGroup
	wg.Go(func() { _, _ = io.Copy(conn, targetConn) })
	wg.Go(func() { _, _ = io.Copy(targetConn, reader) })
	wg.Wait()
}
