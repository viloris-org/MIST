# MIST

MIST is a compact TLS-based private overlay transport for reliable secure connectivity in compliant enterprise, commercial, and self-managed deployments.

The project aims to provide a small operational surface, predictable performance, and clear configuration for teams that need internal access, infrastructure tunneling, remote operations, and service connectivity across trusted environments.

The current implementation provides:

- TLS-based client/server transport
- Stream multiplexing over a single authenticated session
- Local application access through a client-side SOCKS5/HTTP listener
- Cross-platform client library (`mistclient`) for Android, iOS, OpenWRT, and embedded use
- Configurable packet padding support
- Self-signed, ACME, and custom certificate modes
- Optional systemd installation through the bundled installer
- HMAC frame integrity and challenge-response authentication (protocol v3)

## One-Line Server Install

```bash
curl -fsSL https://mist.viloris.org/install-server.sh | bash
```

The script detects your Linux architecture, downloads the latest binary, and installs to `/usr/local/bin`.

For an interactive setup with systemd service:

```bash
curl -fsSL https://mist.viloris.org/install.sh | bash
```

Pre-built binaries are available on the [Releases](https://github.com/viloris-org/MIST/releases) page.

## Build

```bash
go build ./cmd/mist-server
go build ./cmd/mist-client
```

## Library

The `mistclient/` package provides a cross-platform client library:

```go
import "mist/mistclient"

opts := mistclient.Options{
    ServerAddr: "example.com:8443",
    Password:   "your-password",
    Logger:     myLogger, // implements mistclient.Logger
}
client, _ := mistclient.NewClient(opts)
defer client.Close()

conn, _ := client.DialStream(ctx, destination)
```

The `Logger` interface lets each platform inject its own logging — `android.util.Log` on Android, `os_log` on iOS, `logrus` on CLI. See `mistclient/options.go` for full configuration.

## Manual Server Usage

```bash
./mist-server -l 0.0.0.0:8443 -p "your-password"
```

`-l` sets the listen address. `-p` sets the shared password.

### Certificate Modes

Self-signed:

```bash
./mist-server \
  -l 0.0.0.0:8443 \
  -p "your-password" \
  -cert-type self-signed \
  -cert-name 203.0.113.10
```

If `-cert-name` is omitted, the server derives it from the listen address. `0.0.0.0` falls back to `127.0.0.1`.

ACME:

```bash
./mist-server \
  -l 0.0.0.0:8443 \
  -p "your-password" \
  -cert-type acme \
  -cert-name example.com \
  -acme-http :80 \
  -acme-cache ./cert-cache
```

Custom:

```bash
./mist-server \
  -l 0.0.0.0:8443 \
  -p "your-password" \
  -cert-type custom \
  -cert-file /path/to/cert.pem \
  -key-file /path/to/key.pem
```

## Manual Client Usage

```bash
./mist-client -l 127.0.0.1:1080 -s example.com:8443 -p "your-password"
```

With self-signed certificate pinning:

```bash
./mist-client \
  -l 127.0.0.1:1080 \
  -s 203.0.113.10:8443 \
  -p "your-password" \
  -tls-cert-sha256 "server-certificate-sha256"
```

With explicit SNI when connecting by IP:

```bash
./mist-client \
  -l 127.0.0.1:1080 \
  -s 203.0.113.10:8443 \
  -sni example.com \
  -p "your-password"
```

The client also accepts `mist://` URLs:

```bash
./mist-client -l 127.0.0.1:1080 -s "mist://password@example.com:8443?sni=example.com"
```

## Runtime Notes

- Set `LOG_LEVEL=debug` for verbose logs.
- Set `TLS_KEY_LOG=/path/to/keylog.txt` on the client only for TLS debugging.
- Keep passwords, private keys, startup scripts, and certificate cache out of version control.
- Use externally managed secrets and certificate lifecycle tooling for production.

## Compatibility

Protocol v3 with HMAC frame integrity and challenge-response authentication. Compatible with `mist/0.0.2` and later.
