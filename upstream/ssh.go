package upstream

import (
	"encoding/base64"
	"fmt"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

type SSHConfig struct {
	Address    string `envconfig:"SSH_ADDRESS"`
	User       string `envconfig:"SSH_USER"`
	PrivateKey string `envconfig:"SSH_PRIVATE_KEY"`
}

type SSH struct {
	config SSHConfig
}

func NewSSH(config SSHConfig) *SSH {
	return &SSH{config: config}
}

func (s *SSH) Connect(sni string, timeout time.Duration) (net.Conn, error) {
	var authMethods []ssh.AuthMethod

	if s.config.PrivateKey != "" {
		privateKey, err := base64.StdEncoding.DecodeString(s.config.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to base64 decode private key: %w", err)
		}

		signer, err := ssh.ParsePrivateKey(privateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %v", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	sshConfig := &ssh.ClientConfig{
		User:            s.config.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         timeout,
	}

	// connect to ssh server
	sshClient, err := ssh.Dial("tcp", s.config.Address, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to dial ssh server: %v", err)
	}

	// create a tunnel through ssh
	conn, err := sshClient.Dial("tcp", net.JoinHostPort(sni, "443"))
	if err != nil {
		return nil, fmt.Errorf("failed to dial through ssh tunnel: %v", err)
	}

	return conn, nil
}

func (s *SSH) Close() error {
	return nil
}
