## SNI Proxy

### Overview

This project implements an SNI (Server Name Indication) proxy in Go.<br> 
It is designed to intercept TLS ClientHello messages, extract the SNI value, and route the connection through a specified upstream HTTP proxy.

---

### Use Case and Workflow

1. **Setup DNS Overrides:** Configure your DNS so that traffic intended for the restricted service points to the SNI Proxy's address.
2. **Proxy Operation:** The SNI Proxy intercepts incoming traffic, extracts the destination hostname (SNI), and forwards it through the configured upstream proxy.
3. **Upstream Proxy:** The upstream HTTP proxy completes the connection to the destination server, allowing access to region-specific content.
4. **Transparent Access:** For the client application, the process is transparent and requires no additional setup beyond DNS configuration.

---

### Configuration

The proxy is configured using environment variables:

| Environment Variable         | Description                                |  Default  |  Required  |
|------------------------------|--------------------------------------------|:---------:|:----------:|
| `LISTEN_ADDRESS`             | Address on which the SNI proxy listens     |  `:443`   |     No     |
| `CLIENT_HELLO_TIMEOUT`       | Read timeout for ClientHello message       |   `5s`    |     No     |
| `PROXY_ADDRESS`              | Address of the upstream HTTP proxy         |     -     |    Yes     |
| `PROXY_USERNAME`             | Username for upstream proxy authentication |     -     |    Yes     |
| `PROXY_PASSWORD`             | Password for upstream proxy authentication |     -     |    Yes     |
