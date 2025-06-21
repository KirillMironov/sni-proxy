package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/kelseyhightower/envconfig"
)

const clientHelloTimeout = 5 * time.Second

type Config struct {
	ListenAddress string `envconfig:"LISTEN_ADDRESS" default:":443"`
	UpstreamProxy string `envconfig:"UPSTREAM_PROXY" required:"true"`
	ProxyUsername string `envconfig:"PROXY_USERNAME" required:"true"`
	ProxyPassword string `envconfig:"PROXY_PASSWORD" required:"true"`
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
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
	log.Printf("listening on %s", config.ListenAddress)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			continue
		}

		go handleConnection(conn, config)
	}
}

func handleConnection(conn net.Conn, config Config) {
	defer conn.Close()

	// set a read deadline for ClientHello peek
	if err := conn.SetReadDeadline(time.Now().Add(clientHelloTimeout)); err != nil {
		return
	}

	sni, reader, err := sniFromConn(conn)
	if err != nil {
		log.Printf("failed to get sni from conn: %v", err)
		return
	}
	log.Printf("client sni: %s", sni)

	// reset deadline to no deadline
	_ = conn.SetReadDeadline(time.Time{})

	// dial upstream HTTP proxy
	upstreamConn, err := net.Dial("tcp", config.UpstreamProxy)
	if err != nil {
		log.Printf("failed to connect to upstream proxy: %v", err)
		return
	}
	defer upstreamConn.Close()

	// send CONNECT request to upstream proxy with Basic Auth
	targetAddr := sni + ":443"
	auth := config.ProxyUsername + ":" + config.ProxyPassword
	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
	connectReq := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\nProxy-Authorization: %s\r\n\r\n", targetAddr, targetAddr, authHeader)

	_, err = upstreamConn.Write([]byte(connectReq))
	if err != nil {
		log.Printf("failed to send CONNECT to upstream proxy: %v", err)
		return
	}

	// read upstream proxy response
	respReader := bufio.NewReader(upstreamConn)
	respLine, err := respReader.ReadString('\n')
	if err != nil {
		log.Printf("failed to read response from upstream proxy: %v", err)
		return
	}
	if !strings.Contains(respLine, "200") {
		log.Printf("upstream proxy rejected CONNECT: %s", respLine)
		return
	}

	// consume remaining headers
	for {
		line, err := respReader.ReadString('\n')
		if err != nil {
			log.Printf("failed to read response headers: %v", err)
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

	log.Printf("connection closed for host %s", sni)
}
