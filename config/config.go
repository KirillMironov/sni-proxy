package config

import "time"

type Config struct {
	Mode               Mode          `envconfig:"MODE" default:"proxy"`
	ListenAddress      string        `envconfig:"LISTEN_ADDRESS" default:":443"`
	ClientHelloTimeout time.Duration `envconfig:"CLIENT_HELLO_TIMEOUT" default:"5s"`
	LogLevel           string        `envconfig:"LOG_LEVEL" default:"info"`
	ProxyConfig        ProxyConfig
	BypassConfig       BypassConfig
}

type ProxyConfig struct {
	UpstreamType       UpstreamType  `envconfig:"UPSTREAM_TYPE"`
	UpstreamTimeout    time.Duration `envconfig:"UPSTREAM_TIMEOUT" default:"5s"`
	HttpProxyConfig    HttpProxyConfig
	SSHConfig          SSHConfig
	VLESSRealityConfig VLESSRealityConfig
}

type BypassConfig struct {
	ClientHello struct {
		BufferSize uint `envconfig:"CLIENT_HELLO_BUFFER_SIZE" default:"4096"`
		ChunkSize  uint `envconfig:"CLIENT_HELLO_CHUNK_SIZE" default:"1"`
	}
}

type (
	HttpProxyConfig struct {
		Address  string `envconfig:"HTTP_PROXY_ADDRESS"`
		Username string `envconfig:"HTTP_PROXY_USERNAME"`
		Password string `envconfig:"HTTP_PROXY_PASSWORD"`
	}

	SSHConfig struct {
		Address    string `envconfig:"SSH_ADDRESS"`
		User       string `envconfig:"SSH_USER"`
		PrivateKey string `envconfig:"SSH_PRIVATE_KEY"`
	}

	VLESSRealityConfig struct {
		Address     string `envconfig:"VLESS_REALITY_ADDRESS"`
		UUID        string `envconfig:"VLESS_REALITY_UUID"`
		ShortID     string `envconfig:"VLESS_REALITY_SHORTID"`
		PublicKey   string `envconfig:"VLESS_REALITY_PUBLIC_KEY"`
		ServerName  string `envconfig:"VLESS_REALITY_SERVER_NAME"`
		Fingerprint string `envconfig:"VLESS_REALITY_FINGERPRINT"`
	}
)
