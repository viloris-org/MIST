# MIST

MIST is a compact TLS-based private overlay transport for reliable secure connectivity in compliant enterprise, commercial, and self-managed deployments.

The project aims to provide a small operational surface, predictable performance, and clear configuration for teams that need internal access, infrastructure tunneling, remote operations, and service connectivity across trusted environments.

The current implementation provides:

- TLS-based client/server transport
- Stream multiplexing over a single authenticated session
- Local application access through a client-side listener
- Configurable packet padding support
- Self-signed, ACME, and custom certificate modes
- Optional systemd installation through the bundled installer
- Simple deployment model for internal services, lab networks, and managed infrastructure

## Build

```bash
go build -o mist-server ./cmd/server
go build -o mist-client ./cmd/client
```

## Quick Install

An interactive installer is available for server deployments:

```bash
sudo bash install.sh
```

The installer can:

1. Build or reuse the server binary.
2. Configure the listen port, password, and certificate mode.
3. Install the server as a systemd service.
4. Print a client command after installation.

Do not commit generated runtime files, private keys, ACME cache directories, or scripts containing real passwords.

For production use, operate MIST only on networks and systems where you have authorization. Review local policy, logging requirements, certificate handling, and data protection obligations before deployment.

## Manual Server Usage

Basic server:

```bash
./mist-server -l 0.0.0.0:8443 -p "your-password"
```

`-l` sets the listen address. `-p` sets the shared password used by the client.

### Certificate Modes

Self-signed certificate:

```bash
./mist-server \
  -l 0.0.0.0:8443 \
  -p "your-password" \
  -cert-type self-signed \
  -cert-name 203.0.113.10
```

If `-cert-name` is omitted, the server derives it from the listen address. An unspecified address such as `0.0.0.0` falls back to `127.0.0.1`.

ACME certificate:

```bash
./mist-server \
  -l 0.0.0.0:8443 \
  -p "your-password" \
  -cert-type acme \
  -cert-name example.com \
  -acme-http :80 \
  -acme-cache ./cert-cache
```

The domain must resolve to the server, and the ACME HTTP-01 challenge port must be reachable.

Custom certificate:

```bash
./mist-server \
  -l 0.0.0.0:8443 \
  -p "your-password" \
  -cert-type custom \
  -cert-file /path/to/cert.pem \
  -key-file /path/to/key.pem
```

## Manual Client Usage

Start a local client listener and connect it to the server:

```bash
./mist-client -l 127.0.0.1:1080 -s example.com:8443 -p "your-password"
```

`127.0.0.1:1080` is the local client listen address used by applications on the same host.

When using a self-signed certificate, pin the certificate fingerprint printed by the server:

```bash
./mist-client \
  -l 127.0.0.1:1080 \
  -s 203.0.113.10:8443 \
  -p "your-password" \
  -tls-cert-sha256 "server-certificate-sha256"
```

When using a publicly trusted certificate, connect with the domain name:

```bash
./mist-client -l 127.0.0.1:1080 -s example.com:8443 -p "your-password"
```

If you connect by IP while using a domain certificate, provide the expected SNI:

```bash
./mist-client \
  -l 127.0.0.1:1080 \
  -s 203.0.113.10:8443 \
  -sni example.com \
  -p "your-password"
```

## Runtime Notes

- Set `LOG_LEVEL=debug` for verbose logs.
- Set `TLS_KEY_LOG=/path/to/keylog.txt` on the client only when you need TLS debugging.
- Keep passwords, private keys, generated startup scripts, and certificate cache files out of version control.
- Use externally managed secrets and certificate lifecycle tooling for commercial deployments.
- Keep deployment records aligned with your organization's change management and access control policies.

## Compliance And Operations

MIST is intended for authorized secure connectivity use cases such as internal access, controlled remote administration, testing environments, private overlay networking, and commercial infrastructure integration.

Operators are responsible for:

- Ensuring deployments comply with applicable laws, contracts, and internal policies.
- Restricting access to approved users and systems.
- Protecting credentials, private keys, logs, and certificate material.
- Monitoring service health and capacity according to operational requirements.
- Maintaining an upgrade and rollback process for production environments.

## Compatibility

This branch keeps the existing wire format and client configuration behavior for deployment continuity. Existing clients, servers, and integration tooling that already support the current protocol should continue to work while the public naming and configuration surface are being refined.
