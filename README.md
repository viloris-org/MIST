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

## Compliance and Legal

MIST is a general-purpose network transport tool. It does not collect telemetry, phone home, or include any backdoor or bypass mechanism. Like any networking software, it can be used for both legitimate and illegitimate purposes. The authors and contributors provide this software as-is for authorized use only.

### Authorized Use

This software is intended solely for legitimate and authorized purposes, including:

- Internal infrastructure access and private overlay networking
- Remote system administration and DevOps workflows
- Secure connectivity within commercial, enterprise, and self-managed environments
- Educational use, security research in authorized contexts (e.g. CTF, lab environments), and personal self-hosting

You may not use MIST for any purpose that violates applicable laws, regulations, or the rights of others. If you are uncertain whether your use case is authorized, consult qualified legal counsel before deploying.

### Operator Responsibilities

Deploying MIST in any environment makes you the operator of that deployment. Operators are solely responsible for:

- Ensuring the deployment complies with all applicable local, national, and international laws and regulations
- Obtaining any required authorizations, licenses, or permits before operating encrypted tunneling infrastructure
- Restricting access to approved users and systems through appropriate authentication, firewall rules, and access controls
- Protecting credentials, private keys, logs, certificate material, and configuration from unauthorized access
- Monitoring service health, capacity, and security according to organizational operational requirements
- Maintaining an upgrade and rollback process for production environments
- Complying with data protection and privacy obligations, including those related to user traffic that may transit the server
- Keeping accurate deployment records aligned with change management and access control policies

### No Liability

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

The authors and contributors assume no liability for:

- Misuse of the software by operators or end users
- Damages resulting from misconfiguration, inadequate security practices, or failure to follow operational best practices
- Violations of law, regulation, or third-party rights arising from any deployment or use of the software
- Traffic transmitted through MIST tunnels, including any illegal or unauthorized content
- Security incidents resulting from operator failure to apply updates, manage credentials, or secure infrastructure

### Jurisdiction and Export

MIST is developed and distributed globally. By using or distributing this software, you represent that your use and distribution comply with all applicable export control laws, sanctions regulations, and trade restrictions of your jurisdiction and any jurisdiction where the software is deployed.

### Third-Party Services

If you deploy MIST on infrastructure provided by third parties (cloud providers, VPS hosts, CDN services, domain registrars, certificate authorities, etc.), you are responsible for complying with those providers' terms of service and acceptable use policies. The authors make no representation that use of MIST is permitted under any specific provider's terms.

### Reporting

To report a security vulnerability, please open a private security advisory on the GitHub repository or email <connect@viloris.org>. For other concerns, contact the maintainers through the project's official channels.

---

Protocol v3 with HMAC frame integrity and challenge-response authentication. Compatible with `mist/0.0.2` and later.
