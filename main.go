package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"time"

	"github.com/kelseyhightower/envconfig"

	"git.capy.fun/sni-proxy/bypass"
	"git.capy.fun/sni-proxy/proxy"
)

type Config struct {
	Mode          Mode   `envconfig:"MODE" default:"proxy"`
	ListenAddress string `envconfig:"LISTEN_ADDRESS" default:":443"`
	ClientHello   struct {
		Timeout    time.Duration `envconfig:"CLIENT_HELLO_TIMEOUT" default:"5s"`
		BufferSize uint          `envconfig:"CLIENT_HELLO_BUFFER_SIZE" default:"4096"`
		ChunkSize  uint          `envconfig:"CLIENT_HELLO_CHUNK_SIZE" default:"1"`
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

	var connectionHandler ConnectionHandler

	switch config.Mode {
	case ModeProxy:
		connectionHandler = proxy.NewHandler(config.ClientHello.Timeout)
	case ModeBypass:
		connectionHandler = bypass.NewHandler(config.ClientHello.Timeout, config.ClientHello.BufferSize, config.ClientHello.ChunkSize)
	case "":
		return errors.New("mode not specified")
	default:
		return fmt.Errorf("unsupported mode: %s", config.Mode)
	}

	if err := connectionHandler.Init(); err != nil {
		return fmt.Errorf("failed to initialize connection handler: %w", err)
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

		go handleConnection(conn, connectionHandler, config.ClientHello.Timeout)
	}
}

func handleConnection(conn net.Conn, connectionHandler ConnectionHandler, clientHelloTimeout time.Duration) {
	defer conn.Close()

	// set a read deadline for ClientHello peek
	if err := conn.SetReadDeadline(time.Now().Add(clientHelloTimeout)); err != nil {
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

	connectionHandler.Handle(conn, sni, reader)

	slog.Info("client connection closed", slog.String("sni", sni))
}
