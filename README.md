## SNI Proxy

### Overview

This project implements an SNI (Server Name Indication) proxy in Go. It operates in two modes:

1.  **Proxy Mode:** Intercepts TLS ClientHello messages and tunnels the connection through a specified upstream.
2.  **Bypass Mode:** Connects directly to the destination using TCP fragmentation to bypass DPI (Deep Packet Inspection) filters.

---

### Workflow

1.  **Setup DNS Overrides:** Configure your DNS (or `/etc/hosts`) so that traffic intended for the restricted service points to the SNI Proxy's address (e.g., `127.0.0.1`).
2.  **Select Mode:** Choose between `proxy` (tunneling, default) or `bypass` (DPI bypass) using `MODE` environment variable.
3.  **Operation:**
    *   **In Proxy Mode:** The proxy routes traffic through a specified upstream to avoid geographical restrictions.
    *   **In Bypass Mode:** The proxy manipulates TCP packets (splitting the SNI) to bypass DPI filters.

---

### Configuration

SNI Proxy is configured via environment variables.

#### 1. General Configuration

These variables apply to both operation modes.

| Environment Variable   | Description                                      | Default | Required |
|------------------------|--------------------------------------------------|:-------:|:--------:|
| `MODE`                 | Operation mode: `proxy` or `bypass`              | `proxy` |    No    |
| `LISTEN_ADDRESS`       | Address on which the SNI proxy listens           | `:443`  |    No    |
| `CLIENT_HELLO_TIMEOUT` | Read timeout for the initial ClientHello message |  `5s`   |    No    |
| `LOG_LEVEL`            | Logging level: `debug`, `info` or `error`        | `info`  |    No    |

---

#### 2. Bypass Mode Configuration

**When `MODE` is set to `bypass`**

| Environment Variable       | Description                                                                   | Default | Required |
|----------------------------|-------------------------------------------------------------------------------|:-------:|:--------:|
| `CLIENT_HELLO_BUFFER_SIZE` | Buffer size for reading the initial handshake                                 | `4096`  |    No    |

---

#### 3. Proxy Mode Configuration

**When `MODE` is set to `proxy`**

| Environment Variable | Description                                           | Default | Required |
|----------------------|-------------------------------------------------------|:-------:|:--------:|
| `UPSTREAM_TYPE`      | Upstream type: `http-proxy`, `ssh` or `vless-reality` |    -    |   Yes    |
| `UPSTREAM_TIMEOUT`   | Timeout for upstream to complete the connection       |  `5s`   |    No    |


**HTTP Proxy Upstream** (`UPSTREAM_TYPE=http-proxy`)

| Environment Variable  | Description                                | Default | Required |
|-----------------------|--------------------------------------------|:-------:|:--------:|
| `HTTP_PROXY_ADDRESS`  | Address of the upstream HTTP proxy         |    -    |   Yes    |
| `HTTP_PROXY_USERNAME` | Username for upstream proxy authentication |    -    |   Yes    |
| `HTTP_PROXY_PASSWORD` | Password for upstream proxy authentication |    -    |   Yes    |

**SSH Upstream** (`UPSTREAM_TYPE=ssh`)

| Environment Variable | Description                        | Default | Required |
|----------------------|------------------------------------|:-------:|:--------:|
| `SSH_ADDRESS`        | Address of the upstream SSH server |    -    |   Yes    |
| `SSH_USER`           | SSH user                           |    -    |   Yes    |
| `SSH_PRIVATE_KEY`    | Base64 encoded private key         |    -    |   Yes    |

**VLESS Reality Upstream** (`UPSTREAM_TYPE=vless-reality`)

| Environment Variable        | Description                                                         | Default  | Required |
|-----------------------------|---------------------------------------------------------------------|:--------:|:--------:|
| `VLESS_REALITY_ADDRESS`     | The upstream server address (e.g., `1.2.3.4:443`)                   |    -     |   Yes    |
| `VLESS_REALITY_UUID`        | The user UUID v4 used for authentication                            |    -     |   Yes    |
| `VLESS_REALITY_SHORTID`     | The Reality Short ID (hex string)                                   |    -     |   Yes    |
| `VLESS_REALITY_PUBLIC_KEY`  | The X25519 public key of the Reality server                         |    -     |   Yes    |
| `VLESS_REALITY_SERVER_NAME` | The SNI used to mask traffic (e.g., `example.com`)                  |    -     |   Yes    |
| `VLESS_REALITY_FINGERPRINT` | The uTLS client fingerprint to simulate (e.g., `chrome`, `firefox`) | `chrome` |    No    |
