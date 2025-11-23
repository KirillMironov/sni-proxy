## SNI Proxy

### Overview

This project implements an SNI (Server Name Indication) proxy in Go.<br> 
It is designed to intercept TLS ClientHello messages, extract the SNI value, and route the connection through a specified upstream.

---

### Use Case and Workflow

1. **Setup DNS Overrides:** Configure your DNS so that traffic intended for the restricted service points to the SNI Proxy's address.
2. **Proxy Operation:** The SNI Proxy intercepts incoming traffic, extracts the destination hostname (SNI), and forwards it through the configured upstream.
3. **Upstream:** The upstream completes the connection to the destination server, allowing access to region-specific content.
4. **Transparent Access:** For the client application, the process is transparent and requires no additional setup beyond DNS configuration.

---

### Configuration

The proxy is configured using environment variables:

| Environment Variable   | Description                                           | Default | Required |
|------------------------|-------------------------------------------------------|:-------:|:--------:|
| `LISTEN_ADDRESS`       | Address on which the SNI proxy listens                | `:443`  |    No    |
| `CLIENT_HELLO_TIMEOUT` | Read timeout for ClientHello message                  |  `5s`   |    No    |
| `UPSTREAM_TYPE`        | Upstream type: `http-proxy`, `ssh` or `vless-reality` |    -    |   Yes    |
| `UPSTREAM_TIMEOUT`     | Timeout for upstream to complete the request          |  `5s`   |    No    |


#### HTTP Proxy Upstream (`UPSTREAM_TYPE` is `http-proxy`)
| Environment Variable  | Description                                | Default | Required |
|-----------------------|--------------------------------------------|:-------:|:--------:|
| `HTTP_PROXY_ADDRESS`  | Address of the upstream HTTP proxy         |    -    |   Yes    |
| `HTTP_PROXY_USERNAME` | Username for upstream proxy authentication |    -    |   Yes    |
| `HTTP_PROXY_PASSWORD` | Password for upstream proxy authentication |    -    |   Yes    |

#### SSH Upstream (`UPSTREAM_TYPE` is `ssh`)
| Environment Variable | Description                        | Default | Required |
|----------------------|------------------------------------|:-------:|:--------:|
| `SSH_ADDRESS`        | Address of the upstream SSH server |    -    |   Yes    |
| `SSH_USER`           | SSH user                           |    -    |   Yes    |
| `SSH_PRIVATE_KEY`    | Base64 encoded private key         |    -    |   Yes    |

#### VLESS Reality Upstream (`UPSTREAM_TYPE` is `vless-reality`)
| Environment Variable        | Description                                                         | Default  | Required |
|-----------------------------|---------------------------------------------------------------------|:--------:|:--------:|
| `VLESS_REALITY_ADDRESS`     | The upstream server address (e.g., `1.2.3.4:443`)                   |    -     |   Yes    |
| `VLESS_REALITY_UUID`        | The user UUID v4 used for authentication                            |    -     |   Yes    |
| `VLESS_REALITY_SHORTID`     | The Reality Short ID (hex string)                                   |    -     |   Yes    |
| `VLESS_REALITY_PUBLIC_KEY`  | The X25519 public key of the Reality server                         |    -     |   Yes    |
| `VLESS_REALITY_SERVER_NAME` | The SNI used to mask traffic (e.g., `example.com`)                  |    -     |   Yes    |
| `VLESS_REALITY_FINGERPRINT` | The uTLS client fingerprint to simulate (e.g., `chrome`, `firefox`) | `chrome` |    No    |
