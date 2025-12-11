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

	if tcp, ok := targetConn.(*net.TCPConn); ok {
		_ = tcp.SetNoDelay(true)
	}

	clientHelloBuf := make([]byte, b.config.ClientHello.BufferSize)
	n, err := reader.Read(clientHelloBuf)
	if err != nil {
		return
	}
	clientHelloData := clientHelloBuf[:n]

	if err = b.splitClientHello(clientHelloData, targetConn); err != nil {
		slog.ErrorContext(ctx, "failed to split ClientHello", slog.Any("error", err))
		return
	}

	var wg sync.WaitGroup
	wg.Go(func() { _, _ = io.Copy(conn, targetConn) })
	wg.Go(func() { _, _ = io.Copy(targetConn, reader) })
	wg.Wait()
}

func (b *Bypass) splitClientHello(clientHelloData []byte, targetConn net.Conn) error {
	if len(clientHelloData) < 10 || clientHelloData[0] != 0x16 {
		_, err := targetConn.Write(clientHelloData)
		return err
	}

	totalLen := int(clientHelloData[3])<<8 | int(clientHelloData[4])
	if len(clientHelloData) < 5+totalLen {
		_, err := targetConn.Write(clientHelloData)
		return err
	}

	// split strictly at the first byte of the payload
	splitPos := 1

	header1 := []byte{
		0x16, clientHelloData[1], clientHelloData[2], 0x00, byte(splitPos),
	}
	if _, err := targetConn.Write(header1); err != nil {
		return err
	}
	if _, err := targetConn.Write(clientHelloData[5 : 5+splitPos]); err != nil {
		return err
	}

	remaining := totalLen - splitPos

	header2 := []byte{
		0x16, clientHelloData[1], clientHelloData[2], byte(remaining >> 8), byte(remaining),
	}
	if _, err := targetConn.Write(header2); err != nil {
		return err
	}
	if _, err := targetConn.Write(clientHelloData[5+splitPos : 5+totalLen]); err != nil {
		return err
	}

	return nil
}
