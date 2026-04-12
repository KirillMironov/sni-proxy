package upstream

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"time"

	xnet "github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/uuid"
	"github.com/xtls/xray-core/transport/internet/reality"

	"git.capy.fun/sni-proxy/config"
)

type VLESSReality struct {
	config config.VLESSRealityConfig
}

func NewVlessReality(config config.VLESSRealityConfig) *VLESSReality {
	return &VLESSReality{config: config}
}

func (*VLESSReality) Init() error {
	return nil
}

func (v *VLESSReality) Connect(sni string, timeout time.Duration) (net.Conn, error) {
	conn, err := net.DialTimeout("tcp", v.config.Address, timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to dial tcp: %w", err)
	}

	realityConn, err := v.realityHandshake(conn, timeout)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("reality handshake failed: %w", err)
	}

	if err = realityConn.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
		realityConn.Close()
		return nil, fmt.Errorf("failed to set write deadline: %w", err)
	}

	if err = v.writeVlessRequest(realityConn, sni); err != nil {
		realityConn.Close()
		return nil, fmt.Errorf("failed to write vless request: %w", err)
	}

	// reset deadline
	if err = realityConn.SetWriteDeadline(time.Time{}); err != nil {
		realityConn.Close()
		return nil, fmt.Errorf("failed to reset write deadline: %w", err)
	}

	return &VLESSConn{Conn: realityConn}, nil
}

func (v *VLESSReality) realityHandshake(conn net.Conn, timeout time.Duration) (net.Conn, error) {
	shortID, err := hex.DecodeString(v.config.ShortID)
	if err != nil {
		return nil, fmt.Errorf("failed to decode short id: %w", err)
	}

	publicKey, err := base64.RawURLEncoding.DecodeString(v.config.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key: %w", err)
	}

	cfg := &reality.Config{
		Show:        false,
		Fingerprint: v.config.Fingerprint,
		ServerName:  v.config.ServerName,
		ShortId:     shortID,
		PublicKey:   publicKey,
	}

	if cfg.Fingerprint == "" {
		cfg.Fingerprint = "chrome"
	}

	dest, err := xnet.ParseDestination("tcp:" + v.config.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to parse destination: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return reality.UClient(conn, cfg, ctx, dest)
}

func (v *VLESSReality) writeVlessRequest(conn io.Writer, sni string) error {
	buf := bytes.NewBuffer(nil)

	buf.WriteByte(0) // version

	uid, err := uuid.ParseString(v.config.UUID)
	if err != nil {
		return fmt.Errorf("failed to parse uuid: %w", err)
	}
	buf.Write(uid.Bytes())

	buf.WriteByte(0) // addons

	buf.WriteByte(1) // tcp
	if err = binary.Write(buf, binary.BigEndian, uint16(443)); err != nil {
		return fmt.Errorf("failed to write port number: %w", err)
	}

	buf.WriteByte(2) // sni
	buf.WriteByte(byte(len(sni)))
	buf.WriteString(sni)

	_, err = conn.Write(buf.Bytes())
	return err
}

func (v *VLESSReality) Close() error {
	return nil
}

// VLESSConn is a wrapper that strips the VLESS response header on the first Read
type VLESSConn struct {
	net.Conn
	headerRead bool
}

// Read overrides net.Conn.Read to lazily strip the VLESS header
func (vc *VLESSConn) Read(p []byte) (int, error) {
	if !vc.headerRead {
		// This is the first time we are reading.
		// We must consume the VLESS response header first.
		if err := vc.readVlessResponse(); err != nil {
			return 0, err
		}
		vc.headerRead = true
	}

	return vc.Conn.Read(p)
}

func (vc *VLESSConn) readVlessResponse() error {
	// buffer for Version(1) + AddonsLen(1)
	header := make([]byte, 2)
	if _, err := io.ReadFull(vc.Conn, header); err != nil {
		return fmt.Errorf("failed to read vless response header: %w", err)
	}

	if header[0] != 0 {
		return fmt.Errorf("unexpected vless version: %d", header[0])
	}

	addonsLen := int(header[1])

	if addonsLen > 0 {
		addons := make([]byte, addonsLen)
		if _, err := io.ReadFull(vc.Conn, addons); err != nil {
			return fmt.Errorf("failed to read vless addons: %w", err)
		}
	}

	return nil
}
