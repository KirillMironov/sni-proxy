package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"time"
)

func sniFromConn(conn io.Reader) (string, io.Reader, error) {
	var (
		sni         string
		peekedBytes = new(bytes.Buffer)
		reader      = io.TeeReader(conn, peekedBytes)
	)

	_ = tls.Server(wrappedConn{conn: reader}, &tls.Config{
		GetConfigForClient: func(info *tls.ClientHelloInfo) (*tls.Config, error) {
			sni = info.ServerName
			return nil, nil
		},
	}).Handshake()

	if sni == "" {
		return "", nil, errors.New("sni not found in client hello")
	}

	return sni, io.MultiReader(peekedBytes, conn), nil
}

type wrappedConn struct {
	conn io.Reader
}

func (w wrappedConn) Read(b []byte) (int, error) {
	return w.conn.Read(b)
}

func (w wrappedConn) Write(_ []byte) (int, error) {
	return 0, io.ErrClosedPipe
}

func (wrappedConn) Close() error { return nil }

func (wrappedConn) LocalAddr() net.Addr { return nil }

func (wrappedConn) RemoteAddr() net.Addr { return nil }

func (wrappedConn) SetDeadline(_ time.Time) error { return nil }

func (wrappedConn) SetReadDeadline(_ time.Time) error { return nil }

func (wrappedConn) SetWriteDeadline(_ time.Time) error { return nil }
