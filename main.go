package main

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"sync"
	"time"

	"github.com/kelseyhightower/envconfig"

	"git.capy.fun/sni-proxy/upstream"
)

type Config struct {
	ListenAddress      string        `envconfig:"LISTEN_ADDRESS" default:":443"`
	ClientHelloTimeout time.Duration `envconfig:"CLIENT_HELLO_TIMEOUT" default:"5s"`
	Upstream           struct {
		Type            UpstreamType  `envconfig:"UPSTREAM_TYPE"`
		Timeout         time.Duration `envconfig:"UPSTREAM_TIMEOUT" default:"5s"`
		HttpProxyConfig upstream.HttpProxyConfig
		SSHConfig       upstream.SSHConfig
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

	var (
		up  Upstream
		err error
	)

	switch config.Upstream.Type {
	case UpstreamTypeHttpProxy:
		up = upstream.NewHttpProxy(config.Upstream.HttpProxyConfig)
	case UpstreamTypeSSH:
		up = upstream.NewSSH(config.Upstream.SSHConfig)
	case "":
		return errors.New("upstream type not specified")
	default:
		return fmt.Errorf("unsupported upstream type: %s", config.Upstream.Type)
	}
	if err != nil {
		return fmt.Errorf("failed to initialize upstream: %w", err)
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

		go handleConnection(conn, up, config)
	}
}

func handleConnection(conn net.Conn, up Upstream, config Config) {
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

	// dial upstream
	upstreamConn, err := up.Connect(sni, config.Upstream.Timeout)
	if err != nil {
		slog.Error("failed to connect to upstream", slog.Any("error", err))
		return
	}
	defer upstreamConn.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, _ = io.Copy(conn, upstreamConn)
	}()

	go func() {
		defer wg.Done()
		_, _ = io.Copy(upstreamConn, reader)
	}()

	wg.Wait()

	slog.Info("client connection closed", slog.String("sni", sni))
}
