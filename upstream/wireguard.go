package upstream

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"time"

	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun/netstack"

	"git.capy.fun/sni-proxy/config"
)

type Wireguard struct {
	config config.WireguardConfig

	tnet *netstack.Net
	dev  *device.Device
}

func NewWireguard(config config.WireguardConfig) *Wireguard {
	return &Wireguard{config: config}
}

func (w *Wireguard) Init() error {
	if w.config.MTU == 0 {
		w.config.MTU = 1420
	}
	if w.config.DNS == "" {
		w.config.DNS = "1.1.1.1"
	}
	if w.config.KeepaliveInterval == 0 {
		w.config.KeepaliveInterval = 25
	}

	privateKeyHex, err := w.keyToHex(w.config.PrivateKey)
	if err != nil {
		return fmt.Errorf("private key: %w", err)
	}

	publicKeyHex, err := w.keyToHex(w.config.PublicKey)
	if err != nil {
		return fmt.Errorf("public key: %w", err)
	}

	tunnelAddr, err := netip.ParseAddr(w.config.TunnelIP)
	if err != nil {
		return fmt.Errorf("tunnel ip %q: %w", w.config.TunnelIP, err)
	}

	dnsAddr, err := netip.ParseAddr(w.config.DNS)
	if err != nil {
		return fmt.Errorf("dns %q: %w", w.config.DNS, err)
	}

	tunDev, tnet, err := netstack.CreateNetTUN(
		[]netip.Addr{tunnelAddr},
		[]netip.Addr{dnsAddr},
		w.config.MTU,
	)
	if err != nil {
		return fmt.Errorf("create tun: %w", err)
	}

	logger := device.NewLogger(device.LogLevelSilent, "")
	dev := device.NewDevice(tunDev, conn.NewDefaultBind(), logger)

	var ipc strings.Builder

	fmt.Fprintf(&ipc, "private_key=%s\n", privateKeyHex)
	fmt.Fprintf(&ipc, "public_key=%s\n", publicKeyHex)
	fmt.Fprintf(&ipc, "endpoint=%s\n", w.config.Endpoint)
	fmt.Fprintf(&ipc, "persistent_keepalive_interval=%d\n", w.config.KeepaliveInterval)
	ipc.WriteString("allowed_ip=0.0.0.0/0\n")
	ipc.WriteString("allowed_ip=::/0\n")

	if w.config.PresharedKey != "" {
		presharedKeyHex, err := w.keyToHex(w.config.PresharedKey)
		if err != nil {
			dev.Close()
			return fmt.Errorf("preshared key: %w", err)
		}
		fmt.Fprintf(&ipc, "preshared_key=%s\n", presharedKeyHex)
	}

	if err = dev.IpcSet(ipc.String()); err != nil {
		dev.Close()
		return fmt.Errorf("device IpcSet: %w", err)
	}

	if err = dev.Up(); err != nil {
		dev.Close()
		return fmt.Errorf("device up: %w", err)
	}

	w.dev = dev
	w.tnet = tnet

	return nil
}

func (w *Wireguard) Connect(sni string, timeout time.Duration) (net.Conn, error) {
	address := net.JoinHostPort(sni, "443")

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	wgConn, err := w.tnet.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", address, err)
	}

	return wgConn, nil
}

func (w *Wireguard) Close() error {
	if w.dev != nil {
		w.dev.Close()
	}

	return nil
}

func (w *Wireguard) keyToHex(b64 string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		raw, err = base64.RawStdEncoding.DecodeString(b64)
		if err != nil {
			return "", fmt.Errorf("base64 decode: %w", err)
		}
	}

	if len(raw) != 32 {
		return "", fmt.Errorf("expected 32-byte key, got %d", len(raw))
	}

	return hex.EncodeToString(raw), nil
}
