package main

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
)

const (
	clientHelloMaxSize = 1 << 12 // 4KB max to peek ClientHello
	clientHelloTimeout = 5 * time.Second
)

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

func handleConnection(clientConn net.Conn, config Config) {
	defer clientConn.Close()

	// set a read deadline for ClientHello peek
	err := clientConn.SetReadDeadline(time.Now().Add(clientHelloTimeout))
	if err != nil {
		return
	}

	// peek first bytes for ClientHello
	buf := make([]byte, clientHelloMaxSize)

	n, err := clientConn.Read(buf)
	if err != nil {
		log.Printf("failed to read ClientHello: %v", err)
		return
	}

	buf = buf[:n]

	sniHost, err := parseSNI(buf)
	if err != nil {
		log.Printf("failed to parse SNI: %v", err)
		return
	}
	log.Printf("client SNI hostname: %s", sniHost)

	// reset deadline to no deadline
	clientConn.SetReadDeadline(time.Time{})

	// dial upstream HTTP proxy
	upstreamConn, err := net.Dial("tcp", config.UpstreamProxy)
	if err != nil {
		log.Printf("failed to connect to upstream proxy: %v", err)
		return
	}
	defer upstreamConn.Close()

	// send CONNECT request to upstream proxy with Basic Auth
	targetAddr := sniHost + ":443"
	auth := config.ProxyUsername + ":" + config.ProxyPassword
	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
	connectReq := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\nProxy-Authorization: %s\r\n\r\n", targetAddr, targetAddr, authHeader)

	_, err = upstreamConn.Write([]byte(connectReq))
	if err != nil {
		log.Printf("failed to send CONNECT to upstream proxy: %v", err)
		return
	}

	// Read upstream proxy response
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

	// send ClientHello manually before piping
	_, err = upstreamConn.Write(buf)
	if err != nil {
		log.Printf("Failed to send ClientHello to upstream: %v", err)
		return
	}

	// proxy data between clientConn and upstreamConn
	done := make(chan struct{}, 2)

	go func() {
		io.Copy(upstreamConn, clientConn)
		done <- struct{}{}
	}()
	go func() {
		io.Copy(clientConn, upstreamConn)
		done <- struct{}{}
	}()

	<-done // wait for one side to finish

	log.Printf("connection closed for host %s", sniHost)
}

// parseSNI extracts the SNI hostname from a TLS ClientHello packet.
// This is a minimal parser that reads raw ClientHello bytes and extracts the server name.
// Reference: https://tools.ietf.org/html/rfc6066#section-3 and TLS specs.
// Returns hostname or error.
func parseSNI(clientHello []byte) (string, error) {
	// TLS record header: 5 bytes
	if len(clientHello) < 5 {
		return "", errors.New("clientHello too short")
	}
	// Check record type = 22 (handshake)
	if clientHello[0] != 22 {
		return "", errors.New("not a handshake record")
	}
	// Skip record header, start of handshake is at byte 5
	// Handshake message type = 1 (ClientHello)
	if clientHello[5] != 1 {
		return "", errors.New("not a ClientHello")
	}
	// Skip fixed header fields to reach extensions
	// Parsing TLS handshake to get to extensions:
	// Skip:
	//   Handshake type (1 byte)
	//   Length (3 bytes)
	//   Version (2 bytes)
	//   Random (32 bytes)
	//   SessionID length (1 byte) + SessionID (variable)
	//   Cipher Suites length (2 bytes) + Cipher Suites (variable)
	//   Compression methods length (1 byte) + Compression methods (variable)
	if len(clientHello) < 43 {
		return "", errors.New("clientHello too short for extensions")
	}
	pos := 43 // position after random

	// Skip SessionID
	sessionIDLen := int(clientHello[pos])
	pos += 1 + sessionIDLen
	if pos+2 > len(clientHello) {
		return "", errors.New("clientHello truncated after sessionID")
	}

	// Skip Cipher Suites
	cipherSuitesLen := int(clientHello[pos])<<8 | int(clientHello[pos+1])
	pos += 2 + cipherSuitesLen
	if pos+1 > len(clientHello) {
		return "", errors.New("clientHello truncated after cipher suites")
	}

	// Skip Compression Methods
	compMethodsLen := int(clientHello[pos])
	pos += 1 + compMethodsLen
	if pos+2 > len(clientHello) {
		return "", errors.New("clientHello truncated after compression methods")
	}

	// Extensions length
	extensionsLen := int(clientHello[pos])<<8 | int(clientHello[pos+1])
	pos += 2
	end := pos + extensionsLen
	if end > len(clientHello) {
		return "", errors.New("clientHello truncated in extensions")
	}

	for pos+4 <= end {
		extType := int(clientHello[pos])<<8 | int(clientHello[pos+1])
		extLen := int(clientHello[pos+2])<<8 | int(clientHello[pos+3])
		pos += 4
		if pos+extLen > end {
			return "", errors.New("clientHello extension truncated")
		}
		if extType == 0x00 { // Server Name extension
			// The server name extension format:
			// - List length (2 bytes)
			// - Name Type (1 byte, 0 for host_name)
			// - Hostname length (2 bytes)
			// - Hostname (variable)
			extData := clientHello[pos : pos+extLen]
			if len(extData) < 5 {
				return "", errors.New("server name extension too short")
			}
			listLen := int(extData[0])<<8 | int(extData[1])
			if listLen+2 != len(extData) {
				return "", errors.New("server name list length mismatch")
			}
			// Only parse first name
			nameType := extData[2]
			if nameType != 0 {
				return "", errors.New("unsupported server name type")
			}
			nameLen := int(extData[3])<<8 | int(extData[4])
			if 5+nameLen > len(extData) {
				return "", errors.New("server name length invalid")
			}
			hostname := string(extData[5 : 5+nameLen])
			return hostname, nil
		}
		pos += extLen
	}
	return "", errors.New("no server name found")
}
