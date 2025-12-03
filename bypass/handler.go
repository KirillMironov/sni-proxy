package bypass

import (
	"context"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"
)

type Handler struct {
	clientHelloTimeout    time.Duration
	clientHelloBufferSize uint
	clientHelloChunkSize  uint

	// custom resolver to avoid dns loops
	resolver *net.Resolver
}

func NewHandler(clientHelloTimeout time.Duration, clientHelloBufferSize, clientHelloChunkSize uint) *Handler {
	return &Handler{
		clientHelloTimeout:    clientHelloTimeout,
		clientHelloBufferSize: clientHelloBufferSize,
		clientHelloChunkSize:  clientHelloChunkSize,
	}
}

func (h *Handler) Init() error {
	h.resolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: 5 * time.Second}
			return d.DialContext(ctx, "udp", "1.1.1.1:53")
		},
	}

	return nil
}

func (h *Handler) Handle(conn net.Conn, sni string, reader io.Reader) {
	// resolve upstream
	ips, err := h.resolver.LookupHost(context.Background(), sni)
	if err != nil || len(ips) == 0 {
		slog.Error("dns lookup failed", slog.String("sni", sni), slog.Any("error", err))
		return
	}
	target := net.JoinHostPort(ips[0], "443")

	// dial upstream
	targetConn, err := net.DialTimeout("tcp", target, 5*time.Second)
	if err != nil {
		slog.Error("Dial failed", slog.Any("err", err))
		return
	}
	defer targetConn.Close()

	clientHelloBuf := make([]byte, h.clientHelloBufferSize)
	n, err := reader.Read(clientHelloBuf)
	if err != nil {
		return
	}
	clientHelloData := clientHelloBuf[:n]

	// write ClientHello packets in chunks
	for i := 0; i < len(clientHelloData); i += int(h.clientHelloChunkSize) {
		end := i + int(h.clientHelloChunkSize)
		if end > len(clientHelloData) {
			end = len(clientHelloData)
		}

		chunk := clientHelloData[i:end]

		_, err = targetConn.Write(chunk)
		if err != nil {
			slog.Error("failed to write chunk", slog.Any("error", err))
			return
		}
	}

	var wg sync.WaitGroup
	wg.Go(func() { _, _ = io.Copy(conn, targetConn) })
	wg.Go(func() { _, _ = io.Copy(targetConn, reader) })
	wg.Wait()
}
