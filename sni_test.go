package main

import (
	"crypto/tls"
	"net"
	"testing"
)

func TestSNIFromConn(t *testing.T) {
	const wantSNI = "test.example.com"

	serverConn, clientConn := net.Pipe()

	defer serverConn.Close()
	defer clientConn.Close()

	// start client goroutine to send TLS ClientHello
	go func() {
		tlsClient := tls.Client(clientConn, &tls.Config{
			ServerName:         wantSNI,
			InsecureSkipVerify: true,
		})
		_ = tlsClient.Handshake()
	}()

	sni, _, err := sniFromConn(serverConn)
	if err != nil {
		t.Fatalf("sniFromConn() error: %v", err)
	}
	if sni != wantSNI {
		t.Errorf("got %s, want: %s", sni, wantSNI)
	}
}
