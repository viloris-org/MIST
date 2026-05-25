#!/usr/bin/env bash
set -euo pipefail

# ====================================================================
#  MIST (薄雾) Server — one-click interactive install & systemd setup
# ====================================================================

# --- i18n ------------------------------------------------------------------
declare -A T
if [[ "$LANG" =~ ^zh ]]; then
	T[title]="MIST (薄雾) Server 一键配置与安装助手"
	T[err_root]="请使用 root 权限运行此脚本 (例如: sudo bash install.sh)"
	T[check_bin]="检查 mist-server 二进制文件..."
	T[bin_found]="找到当前目录下的 mist-server 二进制。"
	T[bin_not_found]="当前目录未找到 mist-server 二进制，正在下载预编译版本..."
	T[download_fail]="下载 mist-server 失败，尝试使用 Go 进行本地编译..."
	T[compiling]="正在编译 mist-server..."
	T[compile_success]="编译成功！"
	T[compile_fail]="编译失败，请确保 Go 环境及依赖完整。"
	T[go_env_fail]="未检测到 Go 环境且当前目录下没有 mist-server 二进制。请先编译或安装 Go。"
	T[existing_install]="检测到已有 MIST Server 安装。"
	T[current_config]="当前配置文件"
	T[prompt_existing_action]="请选择操作"
	T[action_update]="更新程序并保留现有配置 (默认)"
	T[action_reconfigure]="重新交互配置并覆盖现有配置"
	T[update_preserve]="正在更新程序并保留现有配置..."
	T[update_success]="MIST Server 已更新，并保留现有配置。"
	T[backup_config]="已备份原配置"
	T[config_collect]="交互式配置参数进行中..."
	T[prompt_port]="请输入服务监听端口"
	T[prompt_pw]="请输入连接密码"
	T[default_random]="默认随机"
	T[default]="默认"
	T[cert_mode_title]="请选择您的 TLS 证书管理模式："
	T[cert_mode_self]="自签名证书模式 (self-signed) - 适合本地、内网或带指纹锁定的私密场景 (默认)"
	T[cert_mode_acme]="Let's Encrypt 证书模式 (acme) - 自动申请和续签公开域名证书，公网抗封锁最强"
	T[cert_mode_custom]="自定义证书文件模式 (custom)  - 使用您自己已有的证书和私钥文件"
	T[prompt_choice]="请选择模式"
	T[err_acme_domain]="域名模式下必须指定证书域名，请重新输入"
	T[prompt_domain]="请输入您已解析到当前服务器的裸域名 (Domain)"
	T[prompt_acme_port]="请输入 ACME 挑战验证端口"
	T[prompt_acme_cache]="请输入 ACME 缓存存放路径"
	T[prompt_acme_email]="请输入您的 ACME 注册邮箱 (可选)"
	T[prompt_cert_path]="请输入您的证书 file 路径 (PEM 格式 cert-file)"
	T[err_file_not_found]="文件不存在，请重新输入"
	T[prompt_key_path]="请输入您的私钥 file 路径 (key-file)"
	T[prompt_self_name]="请输入自签名证书名称或 IP 地址 [默认自动推断]"
	T[prompt_fallback]="请输入非授权流量回落地址 (如 127.0.0.1:80，留空默认返回 HTTP 400)"
	T[web_title]="Web 管理面板配置"
	T[prompt_web_enable]="是否启用 Web 管理面板? (y/n)"
	T[prompt_web_listen]="请输入管理面板监听地址"
	T[prompt_web_pw]="请输入管理面板登录密码 (留空则不启用密码认证)"
	T[prompt_web_tls]="是否为管理面板启用 TLS/HTTPS? (y/n)"
	T[prompt_web_tls_reuse]="是否复用主服务 TLS 证书? (y/n)"
	T[prompt_web_tls_cert]="请输入管理面板 TLS 证书文件路径"
	T[prompt_web_tls_key]="请输入管理面板 TLS 私钥文件路径"
	T[summary_web]="管理面板"
	T[summary_web_enabled]="已启用"
	T[summary_web_disabled]="未启用"
	T[web_url]="管理面板地址"
	T[web_pw]="管理面板密码"
	T[summary_title]="配置确认"
	T[summary_port]="监听端口"
	T[summary_pw]="连接密码"
	T[summary_cert]="证书模式"
	T[summary_fallback]="回落地址"
	T[confirm_install]="确认以上配置并开始安装? (y/n)"
	T[install_cancelled]="安装已取消。"
	T[service_config]="配置服务运行方式..."
	T[prompt_systemd]="是否将 MIST Server 安装为 Systemd 开机自启动服务? (y/n)"
	T[installing_systemd]="正在为您配置 Systemd 自启动服务..."
	T[systemd_success]="MIST Server 已成功安装并以 Systemd 服务自启动运行！"
	T[deploy_result]="部署结果与客户端连接指引："
	T[status]="服务状态"
	T[status_active]="已启动并启用自启动"
	T[server_addr]="服务器地址"
	T[server_port]="服务端口"
	T[connect_pw]="连接密码"
	T[cert_mode]="证书模式"
	T[extract_fingerprint]="正在为您提取证书 SHA-256 指纹..."
	T[cert_fingerprint]="证书指纹"
	T[client_cmd]="客户端一键连接指令"
	T[fingerprint_hint]="暂时未能在日志中抓取到 SHA-256，可能启动中，请运行此命令手动获取指纹"
	T[extract_custom]="正在从自定义证书中提取 SHA-256 指纹..."
	T[client_conn_type]="客户端连接方式"
	T[client_conn_custom_hint]="根据您的自定义证书域，使用 IP + SNI，或直接使用解析到该 IP 的域名连入。"
	T[gen_start_sh]="正在为您生成本地前台启动脚本 start.sh..."
	T[start_sh_success]="启动脚本已生成！您可以执行 ./start.sh 运行服务器。"
else
	T[title]="MIST Server Configuration & Installation Assistant"
	T[err_root]="Please run this script with root privileges (e.g. sudo bash install.sh)"
	T[check_bin]="Checking mist-server binary..."
	T[bin_found]="mist-server binary found in the current directory."
	T[bin_not_found]="mist-server binary not found, downloading prebuilt binary..."
	T[download_fail]="Failed to download mist-server, attempting to compile locally using Go..."
	T[compiling]="Compiling mist-server..."
	T[compile_success]="Compilation succeeded!"
	T[compile_fail]="Compilation failed. Please ensure the Go environment and dependencies are complete."
	T[go_env_fail]="Go environment not found and mist-server binary not found. Please install Go or build first."
	T[existing_install]="Existing MIST Server installation detected."
	T[current_config]="Current config file"
	T[prompt_existing_action]="Please select an action"
	T[action_update]="Update binary and keep existing config (Default)"
	T[action_reconfigure]="Reconfigure interactively and overwrite existing config"
	T[update_preserve]="Updating binary while preserving existing config..."
	T[update_success]="MIST Server updated with existing config preserved."
	T[backup_config]="Backed up previous config"
	T[config_collect]="Interactive parameter configuration in progress..."
	T[prompt_port]="Please enter the service listening port"
	T[prompt_pw]="Please enter the connection password"
	T[default_random]="default random"
	T[default]="default"
	T[cert_mode_title]="Please select your TLS certificate management mode:"
	T[cert_mode_self]="Self-signed mode (self-signed) - suitable for local, internal, or pinned SHA-256 setups (Default)"
	T[cert_mode_acme]="Let's Encrypt mode (acme) - auto-apply & renew public domain certificates (Strongest anti-blocking)"
	T[cert_mode_custom]="Custom certificate mode (custom) - use your own existing certificate and private key files"
	T[prompt_choice]="Please select mode"
	T[err_acme_domain]="Domain mode requires a certificate domain. Please enter again"
	T[prompt_domain]="Please enter the bare domain resolved to this server (Domain)"
	T[prompt_acme_port]="Please enter the ACME HTTP-01 challenge port"
	T[prompt_acme_cache]="Please enter the ACME cache path"
	T[prompt_acme_email]="Please enter your ACME email (Optional)"
	T[prompt_cert_path]="Please enter your certificate file path (PEM format cert-file)"
	T[err_file_not_found]="File does not exist, please enter again"
	T[prompt_key_path]="Please enter your private key file path (key-file)"
	T[prompt_self_name]="Please enter self-signed certificate name or IP [Default auto-derived]"
	T[prompt_fallback]="Please enter unauthorized traffic fallback address (e.g. 127.0.0.1:80, leave empty for HTTP 400)"
	T[web_title]="Web Dashboard Configuration"
	T[prompt_web_enable]="Enable the web dashboard? (y/n)"
	T[prompt_web_listen]="Please enter the dashboard listen address"
	T[prompt_web_pw]="Please enter the dashboard login password (leave empty to disable authentication)"
	T[prompt_web_tls]="Enable TLS/HTTPS for the dashboard? (y/n)"
	T[prompt_web_tls_reuse]="Reuse the main server TLS certificate? (y/n)"
	T[prompt_web_tls_cert]="Please enter the dashboard TLS certificate file path"
	T[prompt_web_tls_key]="Please enter the dashboard TLS private key file path"
	T[summary_web]="Dashboard"
	T[summary_web_enabled]="Enabled"
	T[summary_web_disabled]="Disabled"
	T[web_url]="Dashboard URL"
	T[web_pw]="Dashboard Password"
	T[summary_title]="Configuration Summary"
	T[summary_port]="Listening Port"
	T[summary_pw]="Connection Password"
	T[summary_cert]="Certificate Mode"
	T[summary_fallback]="Fallback Address"
	T[confirm_install]="Proceed with the above configuration? (y/n)"
	T[install_cancelled]="Installation cancelled."
	T[service_config]="Configuring service runtime mode..."
	T[prompt_systemd]="Install MIST Server as a Systemd service to auto-start on boot? (y/n)"
	T[installing_systemd]="Configuring Systemd service..."
	T[systemd_success]="MIST Server has been successfully installed and started as a Systemd service!"
	T[deploy_result]="Deployment results & client connection instructions:"
	T[status]="Service Status"
	T[status_active]="Started and enabled on boot"
	T[server_addr]="Server Address"
	T[server_port]="Service Port"
	T[connect_pw]="Connection Password"
	T[cert_mode]="Cert Mode"
	T[extract_fingerprint]="Extracting certificate SHA-256 fingerprint..."
	T[cert_fingerprint]="Cert Fingerprint"
	T[client_cmd]="Client connection command"
	T[fingerprint_hint]="Could not catch SHA-256 fingerprint in logs yet. It might still be starting. Run this to check"
	T[extract_custom]="Extracting SHA-256 fingerprint from custom certificate..."
	T[client_conn_type]="Client connection details"
	T[client_conn_custom_hint]="Use IP + SNI, or directly connect with the domain resolving to this IP according to your custom cert."
	T[gen_start_sh]="Generating local foreground startup script start.sh..."
	T[start_sh_success]="Startup script start.sh generated! You can run the server via ./start.sh"
fi

msg() { echo "${T[$1]:-$1}"; }

# --- helpers ---------------------------------------------------------------

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[0;33m'
BLUE='\033[0;34m'; PURPLE='\033[0;35m'; CYAN='\033[0;36m'; NC='\033[0m'

DOWNLOAD_BASE="${DOWNLOAD_BASE:-https://github.com/viloris-org/MIST/releases/latest/download}"
BIN_NAME="mist-server"
CONF_DIR="${CONF_DIR:-/etc/mist}"
CONF_FILE="$CONF_DIR/server.conf"
SERVICE_FILE="/etc/systemd/system/mist-server.service"
INSTALL_PATH="/usr/local/bin/mist-server"
REQUESTED_ACTION="${1:-}"
case "$REQUESTED_ACTION" in
	--update|update) REQUESTED_ACTION="update" ;;
	--reconfigure|reconfigure) REQUESTED_ACTION="reconfigure" ;;
	"") ;;
	*) echo "Usage: $0 [--update|--reconfigure]" >&2; exit 1 ;;
esac

die()  { echo -e "${RED}[ERROR]${NC} $1" >&2; exit 1; }
info() { echo -e "${BLUE}[INFO]${NC} $1"; }
ok()   { echo -e "${GREEN}[SUCCESS]${NC} $1"; }

# Ensure we can read user input even when piped
if [ "$REQUESTED_ACTION" != "update" ] && [ ! -r /dev/tty ]; then
	die "This installer requires an interactive terminal."
fi

ask() {
	local prompt="$1" var_name="$2"
	if [ ! -r /dev/tty ]; then
		die "This installer requires an interactive terminal."
	fi
	read -r -p "$prompt" "$var_name" < /dev/tty || true
}

random_alnum() {
	local length="$1"
	local value=""
	while [ "${#value}" -lt "$length" ]; do
		value="${value}$(od -An -N32 -tx1 /dev/urandom | tr -d ' \n')"
	done
	printf '%s' "${value:0:length}"
}

detect_platform() {
	case "$(uname -s)" in Linux) ;; *) return 1 ;; esac
	case "$(uname -m)" in
		x86_64|amd64)  echo "linux-amd64" ;;
		aarch64|arm64) echo "linux-arm64" ;;
		*) return 1 ;;
	esac
}

download() {
	local url="$1" dest="$2"
	if command -v curl >/dev/null 2>&1; then
		curl -fsSL "$url" -o "$dest" || { rm -f "$dest"; return 1; }
	elif command -v wget >/dev/null 2>&1; then
		wget -q "$url" -O "$dest" || { rm -f "$dest"; return 1; }
	else
		return 1
	fi
}

backup_existing_config() {
	if [ -f "$CONF_FILE" ]; then
		local backup="$CONF_FILE.$(date +%Y%m%d%H%M%S).bak"
		cp -a "$CONF_FILE" "$backup"
		info "$(msg backup_config): $backup"
	fi
}

install_server_binary() {
	install -m 0755 ./mist-server "$INSTALL_PATH"
}

write_server_conf() {
	local args="$1"
	mkdir -p "$CONF_DIR"
	cat > "$CONF_FILE" <<EOF
ARGS="$args"
EOF
}

write_systemd_service() {
	cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=MIST Server Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$CONF_DIR
EnvironmentFile=$CONF_FILE
ExecStart=$INSTALL_PATH \$ARGS
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF
}

restart_systemd_service() {
	systemctl daemon-reload
	systemctl enable mist-server
	systemctl restart mist-server
}

update_existing_install() {
	info "$(msg update_preserve)"
	if [ ! -f "$CONF_FILE" ]; then
		die "Existing systemd service found but $CONF_FILE is missing. Please reconfigure instead."
	fi
	install_server_binary
	write_systemd_service
	restart_systemd_service
	ok "$(msg update_success)"
}

# Build server argument string from collected config
build_args() {
	local args="-l 0.0.0.0:$PORT -p $PASSWORD -cert-type $CERT_TYPE"
	case "$CERT_TYPE" in
		self-signed)
			[ -n "$CERT_NAME" ] && args="$args -cert-name $CERT_NAME" ;;
		acme)
			args="$args -cert-name $CERT_NAME -acme-http $ACME_HTTP -acme-cache $ACME_CACHE"
			[ -n "$ACME_EMAIL" ] && args="$args -acme-email $ACME_EMAIL" ;;
		custom)
			args="$args -cert-file $CERT_FILE -key-file $KEY_FILE" ;;
	esac
	[ -n "$FALLBACK" ] && args="$args -fallback $FALLBACK"
	# Web dashboard flags.
	if [ "$WEB_ENABLED" = "true" ]; then
		args="$args -web -web-listen $WEB_LISTEN"
		[ -n "$WEB_PASSWORD" ] && args="$args -web-password $WEB_PASSWORD"
		if [ "$WEB_TLS_ENABLED" = "true" ]; then
			args="$args -web-tls"
		elif [ -n "$WEB_TLS_CERT" ]; then
			args="$args -web-tls-cert $WEB_TLS_CERT -web-tls-key $WEB_TLS_KEY"
		fi
	fi
	echo "$args"
}

# Extract SHA-256 fingerprint from journalctl with retry
extract_fingerprint() {
	local sha256=""
	for i in $(seq 1 10); do
		sha256=$(journalctl -u mist-server -n 50 --no-pager 2>/dev/null | grep -oE 'sha256 [0-9a-fA-F]{64}' | awk '{print $2}' | tail -1)
		[ -n "$sha256" ] && break
		sleep 1
	done
	echo "$sha256"
}

# Print client connection instructions based on cert type
print_client_guide() {
	local sha256="$1"
	case "$CERT_TYPE" in
		self-signed)
			if [ -n "$sha256" ]; then
				echo -e "  $(msg cert_fingerprint):   ${GREEN}$sha256${NC}"
				echo -e "\n  👉 ${PURPLE}$(msg client_cmd):${NC}"
				echo -e "  ./mist-client -l 127.0.0.1:1080 -s $PUBLIC_IP:$PORT -p $PASSWORD -tls-cert-sha256 $sha256"
			else
				echo -e "  ${YELLOW}[INFO]${NC} $(msg fingerprint_hint):"
				echo -e "  journalctl -u mist-server -n 20 --no-pager"
			fi ;;
		acme)
			echo -e "\n  👉 ${PURPLE}$(msg client_cmd):${NC}"
			echo -e "  ./mist-client -l 127.0.0.1:1080 -s $CERT_NAME:$PORT -p $PASSWORD" ;;
		custom)
			if [ -n "$sha256" ]; then
				echo -e "  $(msg cert_fingerprint):   ${GREEN}$sha256${NC}"
			fi
			echo -e "\n  👉 ${PURPLE}$(msg client_conn_type):${NC}"
			echo -e "  $(msg client_conn_custom_hint)" ;;
	esac
}

dashboard_url() {
	local scheme="http"
	local hostport="$WEB_LISTEN"
	if [ "$WEB_TLS_ENABLED" = "true" ] || [ -n "$WEB_TLS_CERT" ]; then
		scheme="https"
	fi
	if [ "$WEB_TLS_ENABLED" = "true" ] && [ "$CERT_TYPE" = "acme" ] && [ -n "$CERT_NAME" ]; then
		local port="${WEB_LISTEN##*:}"
		if [[ "$port" =~ ^[0-9]+$ ]]; then
			hostport="$CERT_NAME:$port"
		else
			hostport="$CERT_NAME"
		fi
	fi
	printf '%s://%s' "$scheme" "$hostport"
}

# --- banner ----------------------------------------------------------------
echo -e "${CYAN}"
echo "=========================================================="
echo " __  __ ___ ____ _____ "
echo "|  \/  |_ _/ ___|_   _|"
echo "| |\/| || |\___ \ | |  "
echo "| |  | || | ___) || |  "
echo "|_|  |_|___|____/ |_|  "
echo "                               "
echo "        $(msg title)"
echo "=========================================================="
echo -e "${NC}"

# --- 1. privilege check ----------------------------------------------------
[ "$EUID" -eq 0 ] || die "$(msg err_root)"

# --- 2. acquire binary -----------------------------------------------------
info "[1/5] $(msg check_bin)"
if [ -f "./mist-server" ]; then
	ok "$(msg bin_found)"
else
	info "$(msg bin_not_found)"
	PLATFORM=$(detect_platform || true)
	if [ -n "$PLATFORM" ]; then
		TMPBIN="$(mktemp /tmp/mist-server.XXXXXX)"
		DOWNLOAD_URL="${DOWNLOAD_BASE}/${BIN_NAME}-${PLATFORM}"
		if download "$DOWNLOAD_URL" "$TMPBIN"; then
			mv "$TMPBIN" ./mist-server
			chmod +x ./mist-server
			ok "$(msg bin_found)"
		else
			rm -f "$TMPBIN"
			info "$(msg download_fail)"
			if command -v go >/dev/null 2>&1; then
				info "$(msg compiling)..."
				go build -o mist-server ./cmd/mist-server && ok "$(msg compile_success)" || die "$(msg compile_fail)"
			else
				die "$(msg go_env_fail)"
			fi
		fi
	else
		if command -v go >/dev/null 2>&1; then
			info "$(msg compiling)..."
			go build -o mist-server ./cmd/mist-server && ok "$(msg compile_success)" || die "$(msg compile_fail)"
		else
			die "$(msg go_env_fail)"
		fi
	fi
fi

RECONFIGURE_EXISTING="false"
if [ "$REQUESTED_ACTION" = "update" ]; then
	update_existing_install
	exit 0
elif [ "$REQUESTED_ACTION" = "reconfigure" ]; then
	if [ -f "$CONF_FILE" ] || [ -f "$SERVICE_FILE" ]; then
		RECONFIGURE_EXISTING="true"
	fi
elif [ -f "$CONF_FILE" ] || [ -f "$SERVICE_FILE" ]; then
	echo -e "\n${YELLOW}[INFO]${NC} $(msg existing_install)"
	[ -f "$CONF_FILE" ] && echo -e "  $(msg current_config): ${CYAN}$CONF_FILE${NC}"
	if [ -f "$CONF_FILE" ]; then
		echo -e "  ${CYAN}[1]${NC} $(msg action_update)"
		echo -e "  ${CYAN}[2]${NC} $(msg action_reconfigure)"
		ask "$(msg prompt_existing_action) [1-2, $(msg default) 1]: " existing_action
		case "${existing_action:-1}" in
			2)
				RECONFIGURE_EXISTING="true"
				;;
			*)
				update_existing_install
				exit 0
				;;
		esac
	else
		RECONFIGURE_EXISTING="true"
	fi
fi

# --- 3. interactive config -------------------------------------------------
echo -e "\n${BLUE}[2/5]${NC} $(msg config_collect)"

# port (with validation)
while true; do
	ask "$(msg prompt_port) [$(msg default): 8443]: " input_port
	PORT=${input_port:-8443}
	if [[ "$PORT" =~ ^[0-9]+$ ]] && [ "$PORT" -ge 1 ] && [ "$PORT" -le 65535 ]; then
		break
	fi
	echo -e "${RED}[ERROR]${NC} Invalid port (1-65535). Please try again."
done

# password
default_pw=$(random_alnum 16)
ask "$(msg prompt_pw) [$(msg default_random): $default_pw]: " input_pw
PASSWORD=${input_pw:-$default_pw}

# certificate mode
echo -e "\n$(msg cert_mode_title)"
echo -e "  ${CYAN}[1]${NC} $(msg cert_mode_self)"
echo -e "  ${CYAN}[2]${NC} $(msg cert_mode_acme)"
echo -e "  ${CYAN}[3]${NC} $(msg cert_mode_custom)"
ask "$(msg prompt_choice) [1-3, $(msg default) 1]: " mode_choice

CERT_TYPE="self-signed"
CERT_NAME=""
ACME_HTTP=":80"
ACME_CACHE="cert-cache"
ACME_EMAIL=""
CERT_FILE=""
KEY_FILE=""

case ${mode_choice:-1} in
	2)
		CERT_TYPE="acme"
		while [ -z "${cert_domain:-}" ]; do
			ask "$(msg prompt_domain): " cert_domain
			[ -z "$cert_domain" ] && echo -e "${RED}[ERROR]${NC} $(msg err_acme_domain)"
		done
		CERT_NAME="$cert_domain"
		ask "$(msg prompt_acme_port) [$(msg default) :80]: " input_acme_http
		ACME_HTTP=${input_acme_http:-:80}
		ask "$(msg prompt_acme_cache) [$(msg default) cert-cache]: " input_acme_cache
		ACME_CACHE=${input_acme_cache:-cert-cache}
		ask "$(msg prompt_acme_email): " ACME_EMAIL
		;;
	3)
		CERT_TYPE="custom"
		while true; do
			ask "$(msg prompt_cert_path): " CERT_FILE
			[ -f "$CERT_FILE" ] && break
			echo -e "${RED}[ERROR]${NC} $(msg err_file_not_found)"
		done
		while true; do
			ask "$(msg prompt_key_path): " KEY_FILE
			[ -f "$KEY_FILE" ] && break
			echo -e "${RED}[ERROR]${NC} $(msg err_file_not_found)"
		done
		;;
	*)
		CERT_TYPE="self-signed"
		ask "$(msg prompt_self_name): " CERT_NAME
		;;
esac

# fallback
ask "$(msg prompt_fallback): " FALLBACK
FALLBACK="${FALLBACK:-}"

# web dashboard
echo -e "\n${PURPLE}━━━ $(msg web_title) ━━━${NC}"
WEB_ENABLED="false"
WEB_LISTEN="127.0.0.1:9090"
WEB_PASSWORD=""
WEB_TLS_ENABLED="false"
WEB_TLS_CERT=""
WEB_TLS_KEY=""

ask "$(msg prompt_web_enable) [$(msg default): n]: " web_choice
if [[ "${web_choice:-n}" =~ ^[Yy]$ ]]; then
	WEB_ENABLED="true"

	ask "$(msg prompt_web_listen) [$(msg default) $WEB_LISTEN]: " input_web_listen
	WEB_LISTEN=${input_web_listen:-$WEB_LISTEN}

	# Generate a random password for dashboard
	default_web_pw=$(random_alnum 16)
	ask "$(msg prompt_web_pw) [$(msg default_random): $default_web_pw]: " input_web_pw
	WEB_PASSWORD=${input_web_pw:-$default_web_pw}

	ask "$(msg prompt_web_tls) [$(msg default): n]: " web_tls_choice
	if [[ "${web_tls_choice:-n}" =~ ^[Yy]$ ]]; then
		ask "$(msg prompt_web_tls_reuse) [$(msg default): y]: " web_tls_reuse
		if [[ "${web_tls_reuse:-y}" =~ ^[Yy]$ ]]; then
			WEB_TLS_ENABLED="true"
		else
			while true; do
				ask "$(msg prompt_web_tls_cert): " WEB_TLS_CERT
				[ -f "$WEB_TLS_CERT" ] && break
				echo -e "${RED}[ERROR]${NC} $(msg err_file_not_found)"
			done
			while true; do
				ask "$(msg prompt_web_tls_key): " WEB_TLS_KEY
				[ -f "$WEB_TLS_KEY" ] && break
				echo -e "${RED}[ERROR]${NC} $(msg err_file_not_found)"
			done
		fi
	fi
fi

# --- 4. summary & confirmation ---------------------------------------------
echo -e "\n${BLUE}[3/5]${NC} ${PURPLE}━━━ $(msg summary_title) ━━━${NC}"
echo -e "  $(msg summary_port):     ${GREEN}$PORT${NC}"
echo -e "  $(msg summary_pw):   ${GREEN}$PASSWORD${NC}"
echo -e "  $(msg summary_cert):     ${GREEN}$CERT_TYPE${NC}"
[ -n "$CERT_NAME" ]  && echo -e "  Cert Name:       ${CYAN}$CERT_NAME${NC}"
[ "${FALLBACK:-}" ]  && echo -e "  $(msg summary_fallback):  ${CYAN}$FALLBACK${NC}" || echo -e "  $(msg summary_fallback):  ${YELLOW}(none — HTTP 400)${NC}"
if [ "$WEB_ENABLED" = "true" ]; then
	echo -e "  $(msg summary_web):     ${GREEN}$(msg summary_web_enabled)${NC}"
	echo -e "  $(msg web_url):     ${CYAN}$(dashboard_url)${NC}"
	echo -e "  $(msg web_pw):     ${YELLOW}$WEB_PASSWORD${NC}"
	[ "$WEB_TLS_ENABLED" = "true" ] && echo -e "  Dashboard TLS:   ${GREEN}Enabled (reusing server certificate)${NC}"
	[ -n "$WEB_TLS_CERT" ] && echo -e "  Dashboard TLS:   ${GREEN}Enabled${NC}"
else
	echo -e "  $(msg summary_web):     ${YELLOW}$(msg summary_web_disabled)${NC}"
fi
echo -e "${PURPLE}━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

ask "$(msg confirm_install) [$(msg default): y]: " confirmed
if [[ ! "${confirmed:-y}" =~ ^[Yy]$ ]]; then
	info "$(msg install_cancelled)"
	exit 0
fi

# --- 5. install ------------------------------------------------------------
echo -e "\n${BLUE}[4/5]${NC} $(msg service_config)"
ask "$(msg prompt_systemd) [$(msg default): y]: " is_systemd

ARGS=$(build_args)

if [[ "${is_systemd:-y}" =~ ^[Yy]$ ]]; then
	info "$(msg installing_systemd)"

	if [ "$RECONFIGURE_EXISTING" = "true" ]; then
		backup_existing_config
	fi
	mkdir -p "$CONF_DIR"
	install_server_binary
	write_server_conf "$ARGS"
	write_systemd_service
	restart_systemd_service

	ok "$(msg systemd_success)"

	# public IP detection
	PUBLIC_IP=$(curl -s --max-time 5 https://ipinfo.io/ip 2>/dev/null \
	         || curl -s --max-time 5 https://api.ipify.org 2>/dev/null \
	         || echo "PUBLIC_IP")

	echo -e "\n${BLUE}[5/5]${NC} $(msg deploy_result)"
	echo -e "=========================================================="
	echo -e "  $(msg status):   ${GREEN}$(msg status_active)${NC}"
	echo -e "  $(msg server_addr): ${CYAN}$PUBLIC_IP${NC}"
	echo -e "  $(msg server_port):   ${CYAN}$PORT${NC}"
	echo -e "  $(msg connect_pw):   ${YELLOW}$PASSWORD${NC}"
	echo -e "  $(msg cert_mode):   ${CYAN}$CERT_TYPE${NC}"

		if [ "$WEB_ENABLED" = "true" ]; then
			echo -e "  $(msg web_url): ${CYAN}$(dashboard_url)${NC}"
			echo -e "  $(msg web_pw): ${YELLOW}$WEB_PASSWORD${NC}"
		fi
	echo -e "  $(msg extract_fingerprint)"
	SHA256=$(extract_fingerprint)
	print_client_guide "$SHA256"
	echo -e "=========================================================="

else
	# foreground start script
	info "$(msg gen_start_sh)"
	cat > ./start.sh <<EOF
#!/usr/bin/env bash
exec ./mist-server $ARGS
EOF
	chmod +x ./start.sh
	ok "$(msg start_sh_success)"
fi
