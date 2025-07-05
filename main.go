package main

import (
	"bufio"
	"encoding/base64"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	ListenAddress      string        `envconfig:"LISTEN_ADDRESS" default:":443"`
	ClientHelloTimeout time.Duration `envconfig:"CLIENT_HELLO_TIMEOUT" default:"5s"`
	Proxy              struct {
		Address  string `envconfig:"PROXY_ADDRESS" required:"true"`
		Username string `envconfig:"PROXY_USERNAME" required:"true"`
		Password string `envconfig:"PROXY_PASSWORD" required:"true"`
	}
}

func main() {
	logger := slog.New(newHandler(os.Stdout, slog.LevelInfo))
	slog.SetDefault(logger)

	if err := run(); err != nil {
		slog.Error("failed to run", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	var config Config
	if err := envconfig.Process("", &config); err != nil {
		return err
	}

	ln, err := net.Listen("tcp", config.ListenAddress)
	if err != nil {
		return err
	}
	slog.Info("server is listening", slog.String("address", config.ListenAddress))

	for {
		conn, err := ln.Accept()
		if err != nil {
			slog.Error("failed to accept connection", slog.Any("error", err))
			continue
		}

		go handleConnection(conn, config)
	}
}

func handleConnection(conn net.Conn, config Config) {
	defer conn.Close()

	// set a read deadline for ClientHello peek
	if err := conn.SetReadDeadline(time.Now().Add(config.ClientHelloTimeout)); err != nil {
		return
	}

	sni, reader, err := sniFromConn(conn)
	if err != nil {
		slog.Error("failed to get sni from connection", slog.Any("error", err))
		return
	}
	slog.Info("new client", slog.String("sni", sni))

	// reset deadline to no deadline
	_ = conn.SetReadDeadline(time.Time{})

	// dial upstream HTTP proxy
	upstreamConn, err := net.Dial("tcp", config.Proxy.Address)
	if err != nil {
		slog.Error("failed to connect to upstream proxy", slog.Any("error", err))
		return
	}
	defer upstreamConn.Close()

	// send CONNECT request to upstream proxy with Basic Auth
	credentials := config.Proxy.Username + ":" + config.Proxy.Password
	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(credentials))

	connectReq := &http.Request{
		URL:    new(url.URL),
		Method: http.MethodConnect,
		Host:   net.JoinHostPort(sni, ":443"),
		Header: http.Header{
			"Proxy-Authorization": []string{authHeader},
		},
	}

	if err = connectReq.Write(upstreamConn); err != nil {
		slog.Error("failed to write connect request to upstream proxy", slog.Any("error", err))
		return
	}

	// read upstream proxy response
	respReader := bufio.NewReader(upstreamConn)
	respLine, err := respReader.ReadString('\n')
	if err != nil {
		slog.Error("failed to read response from upstream proxy", slog.Any("error", err))
		return
	}
	if !strings.Contains(respLine, "200") {
		slog.Error("upstream proxy rejected connect request", slog.String("response", respLine))
		return
	}

	// consume remaining headers
	for {
		line, err := respReader.ReadString('\n')
		if err != nil {
			slog.Error("failed to read response headers", slog.Any("error", err))
			return
		}
		if line == "\r\n" {
			break
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		_, _ = io.Copy(conn, upstreamConn)

		if c := conn.(*net.TCPConn); c != nil {
			_ = c.CloseWrite()
		}
	}()

	go func() {
		defer wg.Done()

		_, _ = io.Copy(upstreamConn, reader)

		if c := upstreamConn.(*net.TCPConn); c != nil {
			_ = c.CloseWrite()
		}
	}()

	wg.Wait()

	slog.Info("client connection closed", slog.String("sni", sni))
}
