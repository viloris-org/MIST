# MIST · 薄雾

[English](../../README.md) · 中文 · [日本語](README.ja.md) · [Español](README.es.md) · [Русский](README.ru.md)

薄雾（MIST）是一个基于 TLS 的紧凑型私有覆盖传输工具，为合规的企业、商业和自托管部署提供可靠的安全连接。

项目致力于提供轻量的运维表面、可预测的性能和清晰的配置，适合需要内部访问、基础设施隧道、远程运维和跨可信环境服务连接的团队。

当前实现提供：

- 基于 TLS 的客户端/服务端传输
- 基于单个认证会话的流多路复用
- 通过客户端 SOCKS5/HTTP 监听器提供本地应用访问
- 跨平台客户端库（`mistclient`），支持 Android、iOS、OpenWRT 及嵌入式使用
- 可配置的数据包填充支持
- 自签名、ACME 和自定义证书模式
- 通过捆绑安装器提供可选的 systemd 安装
- HMAC 帧完整性和质询-响应认证（协议 v3）

## 一行命令安装服务端

```bash
curl -fsSL https://mist.viloris.org/install-server.sh | bash
```

该脚本会自动检测您的 Linux 架构，下载最新的二进制文件，并安装到 `/usr/local/bin`。

交互式安装并配置 systemd 服务：

```bash
curl -fsSL https://mist.viloris.org/install.sh | bash
```

预编译的二进制文件可在 [Releases](https://github.com/viloris-org/MIST/releases) 页面获取。

## 构建

```bash
go build ./cmd/mist-server
go build ./cmd/mist-client
```

## 客户端库

`mistclient/` 包提供了一个跨平台的客户端库：

```go
import "mist/mistclient"

opts := mistclient.Options{
    ServerAddr: "example.com:8443",
    Password:   "your-password",
    Logger:     myLogger, // 实现 mistclient.Logger 接口
}
client, _ := mistclient.NewClient(opts)
defer client.Close()

conn, _ := client.DialStream(ctx, destination)
```

`Logger` 接口允许各平台注入自己的日志实现——Android 使用 `android.util.Log`，iOS 使用 `os_log`，CLI 使用 `logrus`。完整配置参见 `../../mistclient/options.go`。

## 服务端手动使用

```bash
./mist-server -l 0.0.0.0:8443 -p "your-password"
```

`-l` 设置监听地址，`-p` 设置共享密码。

### 证书模式

自签名：

```bash
./mist-server \
  -l 0.0.0.0:8443 \
  -p "your-password" \
  -cert-type self-signed \
  -cert-name 203.0.113.10
```

若省略 `-cert-name`，服务端将从监听地址自动推断。`0.0.0.0` 回退为 `127.0.0.1`。

ACME：

```bash
./mist-server \
  -l 0.0.0.0:8443 \
  -p "your-password" \
  -cert-type acme \
  -cert-name example.com \
  -acme-http :80 \
  -acme-cache ./cert-cache
```

自定义证书：

```bash
./mist-server \
  -l 0.0.0.0:8443 \
  -p "your-password" \
  -cert-type custom \
  -cert-file /path/to/cert.pem \
  -key-file /path/to/key.pem
```

## 客户端手动使用

```bash
./mist-client -l 127.0.0.1:1080 -s example.com:8443 -p "your-password"
```

使用自签名证书指纹锁定：

```bash
./mist-client \
  -l 127.0.0.1:1080 \
  -s 203.0.113.10:8443 \
  -p "your-password" \
  -tls-cert-sha256 "server-certificate-sha256"
```

通过 IP 连接时显式指定 SNI：

```bash
./mist-client \
  -l 127.0.0.1:1080 \
  -s 203.0.113.10:8443 \
  -sni example.com \
  -p "your-password"
```

客户端也支持 `mist://` URL：

```bash
./mist-client -l 127.0.0.1:1080 -s "mist://password@example.com:8443?sni=example.com"
```

## 运行时说明

- 设置 `LOG_LEVEL=debug` 获取详细日志。
- 仅在客户端设置 `TLS_KEY_LOG=/path/to/keylog.txt` 用于 TLS 调试。
- 请勿将密码、私钥、启动脚本和证书缓存提交到版本控制。
- 生产环境请使用外部密钥管理和证书生命周期工具。

## 合规与法律

薄雾（MIST）是一个通用网络传输工具。它不收集遥测数据，不联网回传，也不包含任何后门或绕过机制。与任何网络软件一样，它既可用于合法用途也可能被用于非法用途。作者和贡献者按"原样"提供本软件，仅供授权使用。

### 授权使用

本软件仅适用于合法和授权目的，包括：

- 内部基础设施访问和私有覆盖网络
- 远程系统管理和 DevOps 工作流
- 商业、企业和自托管环境中的安全连接
- 教育用途、授权环境下的安全研究（如 CTF、实验环境）以及个人自托管

您不得将薄雾用于任何违反适用法律、法规或他人权利的用途。如果您不确定您的使用场景是否被授权，请在部署前咨询合格的法律顾问。

### 运营者责任

在任何环境中部署薄雾即意味着您成为该部署的运营者。运营者对以下事项承担全部责任：

- 确保部署符合所有适用的本地、国家和国际法律法规
- 在运营加密隧道基础设施之前获取任何所需的授权、许可或批准
- 通过适当的认证、防火墙规则和访问控制限制已批准的用户和系统的访问
- 保护凭证、私钥、日志、证书材料和配置免受未经授权的访问
- 根据组织的运营要求监控服务健康、容量和安全
- 维护生产环境的升级和回滚流程
- 遵守数据保护和隐私义务，包括与可能经过服务器的用户流量相关的义务
- 保留与变更管理和访问控制政策一致的准确部署记录

### 免责声明

本软件按"原样"提供，不提供任何明示或暗示的保证，包括但不限于适销性、特定用途适用性和非侵权的保证。在任何情况下，作者或版权持有人均不对因合同、侵权或其他原因引起的任何索赔、损害或其他责任承担责任。

作者和贡献者对以下事项不承担任何责任：

- 运营者或最终用户对本软件的滥用
- 由于配置错误、安全实践不足或未遵循运维最佳实践而造成的损害
- 因任何部署或使用本软件而导致的违反法律、法规或第三方权利的行为
- 通过薄雾隧道传输的流量，包括任何非法或未经授权的内容
- 因运营者未能应用更新、管理凭证或保护基础设施而导致的安全事件

### 管辖权和出口

薄雾在全球范围内开发和分发。使用或分发本软件即表示您声明您的使用和分发符合您所在司法管辖区以及软件部署所在司法管辖区的所有适用出口管制法律、制裁法规和贸易限制。

### 第三方服务

如果您在第三方提供的基础设施上部署薄雾（云服务商、VPS 主机、CDN 服务、域名注册商、证书颁发机构等），您有责任遵守这些服务商的条款和可接受使用政策。作者不对在任何特定服务商条款下是否允许使用薄雾做出任何声明。

### 报告

如需报告安全漏洞，请在 GitHub 仓库上提交私有安全通告，或发送邮件至 <connect@viloris.org>。如有其他问题，请通过项目的官方渠道联系维护者。

---

协议 v3，具有 HMAC 帧完整性和质询-响应认证。兼容 `mist/0.0.2` 及更高版本。
