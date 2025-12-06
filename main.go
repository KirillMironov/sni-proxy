package main

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"time"

	"github.com/kelseyhightower/envconfig"

	"git.capy.fun/sni-proxy/config"
	"git.capy.fun/sni-proxy/handler"
)

type ConnectionHandler interface {
	Init() error
	Handle(conn net.Conn, sni string, reader io.Reader)
}

func main() {
	logger := slog.New(newSlogHandler(os.Stdout, slog.LevelInfo))
	slog.SetDefault(logger)

	if err := run(); err != nil {
		slog.Error("failed to run", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	var cfg config.Config
	if err := envconfig.Process("", &cfg); err != nil {
		return err
	}

	var connectionHandler ConnectionHandler

	switch cfg.Mode {
	case config.ModeProxy:
		connectionHandler = handler.NewProxy(cfg.ProxyConfig)
	case config.ModeBypass:
		connectionHandler = handler.NewBypass(cfg.BypassConfig)
	case "":
		return errors.New("mode not specified")
	default:
		return fmt.Errorf("unsupported mode: %s", cfg.Mode)
	}

	if err := connectionHandler.Init(); err != nil {
		return fmt.Errorf("failed to initialize connection handler: %w", err)
	}

	ln, err := net.Listen("tcp", cfg.ListenAddress)
	if err != nil {
		return err
	}
	slog.Info("server is listening", slog.String("address", cfg.ListenAddress))

	for {
		conn, err := ln.Accept()
		if err != nil {
			slog.Error("failed to accept connection", slog.Any("error", err))
			continue
		}

		go handleConnection(conn, connectionHandler, cfg.ClientHelloTimeout)
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
