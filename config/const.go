package config

type Mode string

const (
	ModeProxy  Mode = "proxy"
	ModeBypass Mode = "bypass"
)

type UpstreamType string

const (
	UpstreamTypeHttpProxy    UpstreamType = "http-proxy"
	UpstreamTypeSSH          UpstreamType = "ssh"
	UpstreamTypeVLESSReality UpstreamType = "vless-reality"
	UpstreamTypeWireguard    UpstreamType = "wireguard"
)
