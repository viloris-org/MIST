#!/usr/bin/env bash

# ====================================================================
#  MIST (薄雾) Server 一键交互式安装与自启动配置脚本 (Multilingual)
# ====================================================================

# 终端颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0;37m' # 无颜色

# 自动检测语言 (中文/英文)
LANG_CODE="en"
if [[ "$LANG" =~ ^zh ]]; then
    LANG_CODE="zh"
fi

# 多语言字典辅助函数
msg() {
    local key="$1"
    case "$LANG_CODE" in
        zh)
            case "$key" in
                err_root) echo "请使用 root 权限运行此脚本 (例如: sudo bash install.sh)" ;;
                check_bin) echo "检查 mist-server 二进制文件..." ;;
                bin_found) echo "找到当前目录下的 mist-server 二进制。" ;;
                bin_not_found) echo "当前目录未找到 mist-server 二进制，尝试使用 Go 进行本地编译..." ;;
                compiling) echo "正在编译 mist-server..." ;;
                compile_success) echo "编译成功！" ;;
                compile_fail) echo "编译失败，请确保 Go 环境及依赖完整。" ;;
                go_env_fail) echo "未检测到 Go 环境且当前目录下没有 mist-server 二进制。请先编译或安装 Go。" ;;
                config_collect) echo "交互式配置参数进行中..." ;;
                prompt_port) echo "请输入服务监听端口" ;;
                prompt_pw) echo "请输入连接密码" ;;
                default_random) echo "默认随机" ;;
                default) echo "默认" ;;
                cert_mode_title) echo "请选择您的 TLS 证书管理模式：" ;;
                cert_mode_self) echo "自签名证书模式 (self-signed) - 适合本地、内网或带指纹锁定的私密场景 (默认)" ;;
                cert_mode_acme) echo "Let's Encrypt 证书模式 (acme) - 自动申请和续签公开域名证书，公网抗封锁最强" ;;
                cert_mode_custom) echo "自定义证书文件模式 (custom)  - 使用您自己已有的证书和私钥文件" ;;
                prompt_choice) echo "请选择模式" ;;
                err_acme_domain) echo "域名模式下必须指定证书域名，请重新输入" ;;
                prompt_domain) echo "请输入您已解析到当前服务器的裸域名 (Domain)" ;;
                prompt_acme_port) echo "请输入 ACME 挑战验证端口" ;;
                prompt_acme_cache) echo "请输入 ACME 缓存存放路径" ;;
                prompt_acme_email) echo "请输入您的 ACME 注册邮箱 (可选)" ;;
                prompt_cert_path) echo "请输入您的证书 file 路径 (PEM 格式 cert-file)" ;;
                err_file_not_found) echo "文件不存在，请重新输入" ;;
                prompt_key_path) echo "请输入您的私钥 file 路径 (key-file)" ;;
                prompt_self_name) echo "请输入自签名证书名称或 IP 地址 [默认自动推断]" ;;
                prompt_fallback) echo "请输入非授权流量回落地址 (如 127.0.0.1:80，留空默认返回 HTTP 400)" ;;
                service_config) echo "配置服务运行方式..." ;;
                prompt_systemd) echo "是否将 MIST Server 安装为 Systemd 开机自启动服务? (y/n)" ;;
                installing_systemd) echo "正在为您配置 Systemd 自启动服务..." ;;
                systemd_success) echo "MIST Server 已成功安装并以 Systemd 服务自启动运行！" ;;
                deploy_result) echo "部署结果与客户端连接指引：" ;;
                status) echo "服务状态" ;;
                status_active) echo "已启动并启用自启动" ;;
                server_addr) echo "服务器地址" ;;
                server_port) echo "服务端口" ;;
                connect_pw) echo "连接密码" ;;
                cert_mode) echo "证书模式" ;;
                extract_fingerprint) echo "正在为您提取证书 SHA-256 指纹..." ;;
                cert_fingerprint) echo "证书指纹" ;;
                client_cmd) echo "客户端一键连接指令" ;;
                fingerprint_hint) echo "暂时未能在日志中抓取到 SHA-256，可能启动中，请运行此命令手动获取指纹" ;;
                extract_custom) echo "正在从自定义证书中提取 SHA-256 指纹..." ;;
                client_conn_type) echo "客户端连接方式" ;;
                client_conn_custom_hint) echo "根据您的自定义证书域，使用 IP + SNI，或直接使用解析到该 IP 的域名连入。" ;;
                gen_start_sh) echo "正在为您生成本地前台启动脚本 start.sh..." ;;
                start_sh_success) echo "启动脚本已生成！您可以执行 ./start.sh 运行服务器。" ;;
                assistant_title) echo "MIST (薄雾) Server 一键配置与安装助手" ;;
            esac
            ;;
        *)
            case "$key" in
                err_root) echo "Please run this script with root privileges (e.g. sudo bash install.sh)" ;;
                check_bin) echo "Checking mist-server binary..." ;;
                bin_found) echo "mist-server binary found in the current directory." ;;
                bin_not_found) echo "mist-server binary not found, attempting to compile locally using Go..." ;;
                compiling) echo "Compiling mist-server..." ;;
                compile_success) echo "Compilation succeeded!" ;;
                compile_fail) echo "Compilation failed. Please ensure the Go environment and dependencies are complete." ;;
                go_env_fail) echo "Go environment not found and mist-server binary not found. Please install Go or build first." ;;
                config_collect) echo "Interactive parameter configuration in progress..." ;;
                prompt_port) echo "Please enter the service listening port" ;;
                prompt_pw) echo "Please enter the connection password" ;;
                default_random) echo "default random" ;;
                default) echo "default" ;;
                cert_mode_title) echo "Please select your TLS certificate management mode:" ;;
                cert_mode_self) echo "Self-signed mode (self-signed) - suitable for local, internal, or pinned SHA-256 setups (Default)" ;;
                cert_mode_acme) echo "Let's Encrypt mode (acme) - auto-apply & renew public domain certificates (Strongest anti-blocking)" ;;
                cert_mode_custom) echo "Custom certificate mode (custom) - use your own existing certificate and private key files" ;;
                prompt_choice) echo "Please select mode" ;;
                err_acme_domain) echo "Domain mode requires a certificate domain. Please enter again" ;;
                prompt_domain) echo "Please enter the bare domain resolved to this server (Domain)" ;;
                prompt_acme_port) echo "Please enter the ACME HTTP-01 challenge port" ;;
                prompt_acme_cache) echo "Please enter the ACME cache path" ;;
                prompt_acme_email) echo "Please enter your ACME email (Optional)" ;;
                prompt_cert_path) echo "Please enter your certificate file path (PEM format cert-file)" ;;
                err_file_not_found) echo "File does not exist, please enter again" ;;
                prompt_key_path) echo "Please enter your private key file path (key-file)" ;;
                prompt_self_name) echo "Please enter self-signed certificate name or IP [Default auto-derived]" ;;
                prompt_fallback) echo "Please enter unauthorized traffic fallback address (e.g. 127.0.0.1:80, leave empty for HTTP 400)" ;;
                service_config) echo "Configuring service runtime mode..." ;;
                prompt_systemd) echo "Install MIST Server as a Systemd service to auto-start on boot? (y/n)" ;;
                installing_systemd) echo "Configuring Systemd service..." ;;
                systemd_success) echo "MIST Server has been successfully installed and started as a Systemd service!" ;;
                deploy_result) echo "Deployment results & client connection instructions:" ;;
                status) echo "Service Status" ;;
                status_active) echo "Started and enabled on boot" ;;
                server_addr) echo "Server Address" ;;
                server_port) echo "Service Port" ;;
                connect_pw) echo "Connection PW" ;;
                cert_mode) echo "Cert Mode" ;;
                extract_fingerprint) echo "Extracting certificate SHA-256 fingerprint..." ;;
                cert_fingerprint) echo "Cert Fingerprint" ;;
                client_cmd) echo "👉 Client connection command" ;;
                fingerprint_hint) echo "Could not catch SHA-256 fingerprint in logs yet. It might still be starting. Run this to check" ;;
                extract_custom) echo "Extracting SHA-256 fingerprint from custom certificate..." ;;
                client_conn_type) echo "Client connection details" ;;
                client_conn_custom_hint) echo "Use IP + SNI, or directly connect with the domain resolving to this IP according to your custom cert." ;;
                gen_start_sh) echo "Generating local foreground startup script start.sh..." ;;
                start_sh_success) echo "Startup script start.sh generated! You can run the server via ./start.sh" ;;
                assistant_title) echo "MIST Server Configuration & Installation Assistant" ;;
            esac
            ;;
    esac
}

# 打印漂亮的横幅
echo -e "${CYAN}"
echo "=========================================================="
echo " __  __ ___ ____ _____ "
echo "|  \/  |_ _/ ___|_   _|"
echo "| |\/| || |\___ \ | |  "
echo "| |  | || | ___) || |  "
echo "|_|  |_|___|____/ |_|  "
echo "                               "
echo "        $(msg assistant_title)"
echo "=========================================================="
echo -e "${NC}"

# 1. 权限检查
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}[ERROR]${NC} $(msg err_root)"
    exit 1
fi

# 2. 检测并编译 mist-server
echo -e "${BLUE}[1/4]${NC} $(msg check_bin)"
if [ -f "./mist-server" ]; then
    echo -e "${GREEN}[SUCCESS]${NC} $(msg bin_found)"
else
    echo -e "${YELLOW}[INFO]${NC} $(msg bin_not_found)"
    if command -v go >/dev/null 2>&1; then
        echo -e "${BLUE}$(msg compiling)...${NC}"
        if go build -o mist-server ./cmd/mist-server; then
            echo -e "${GREEN}[SUCCESS]${NC} $(msg compile_success)"
        else
            echo -e "${RED}[ERROR]${NC} $(msg compile_fail)"
            exit 1
        fi
    else
        echo -e "${RED}[ERROR]${NC} $(msg go_env_fail)"
        exit 1
    fi
fi

# 3. 交互式参数收集
echo -e "\n${BLUE}[2/4]${NC} $(msg config_collect)"

# 3.1 监听端口
read -p "$(msg prompt_port) [$(msg default): 8443]: " input_port
PORT=${input_port:-8443}

# 3.2 密码配置
default_pw=$(head /dev/urandom | tr -dc A-Za-z0-9 | head -c 16)
read -p "$(msg prompt_pw) [$(msg default_random): $default_pw]: " input_pw
PASSWORD=${input_pw:-$default_pw}

# 3.3 证书模式选择
echo -e "\n$(msg cert_mode_title)"
echo -e "  ${CYAN}[1]${NC} $(msg cert_mode_self)"
echo -e "  ${CYAN}[2]${NC} $(msg cert_mode_acme)"
echo -e "  ${CYAN}[3]${NC} $(msg cert_mode_custom)"
read -p "$(msg prompt_choice) [1-3, $(msg default) 1]: " mode_choice

CERT_TYPE="self-signed"
CERT_NAME=""
ACME_HTTP=":80"
ACME_CACHE="cert-cache"
ACME_EMAIL=""
CERT_FILE=""
KEY_FILE=""

case $mode_choice in
    2)
        CERT_TYPE="acme"
        read -p "$(msg prompt_domain): " cert_domain
        while [ -z "$cert_domain" ]; do
            read -p "${RED}[ERROR]${NC} $(msg err_acme_domain): " cert_domain
        done
        CERT_NAME="$cert_domain"

        read -p "$(msg prompt_acme_port) [$(msg default) :80]: " input_acme_http
        ACME_HTTP=${input_acme_http:-:80}

        read -p "$(msg prompt_acme_cache) [$(msg default) cert-cache]: " input_acme_cache
        ACME_CACHE=${input_acme_cache:-cert-cache}

        read -p "$(msg prompt_acme_email): " CERT_EMAIL
        ;;
    3)
        CERT_TYPE="custom"
        read -p "$(msg prompt_cert_path): " input_cert_file
        while [ ! -f "$input_cert_file" ]; do
            read -p "${RED}[$(msg err_file_not_found)]${NC} $(msg prompt_cert_path): " input_cert_file
        done
        CERT_FILE="$input_cert_file"

        read -p "$(msg prompt_key_path): " input_key_file
        while [ ! -f "$input_key_file" ]; do
            read -p "${RED}[$(msg err_file_not_found)]${NC} $(msg prompt_key_path): " input_key_file
        done
        KEY_FILE="$input_key_file"
        ;;
    *)
        CERT_TYPE="self-signed"
        read -p "$(msg prompt_self_name): " input_cert_name
        CERT_NAME="$input_cert_name"
        ;;
esac

# 3.4 探测回落配置 (Fallback)
read -p "$(msg prompt_fallback): " input_fallback
FALLBACK="$input_fallback"

# 4. 设置自启动或独立启动
echo -e "\n${BLUE}[3/4]${NC} $(msg service_config)"
read -p "$(msg prompt_systemd) [$(msg default): y]: " is_systemd
IS_SYSTEMD_CONFIRM=${is_systemd:-y}
if [[ "$IS_SYSTEMD_CONFIRM" =~ ^[Yy]$ ]]; then
    echo -e "$(msg installing_systemd)"

    # 动态拼接所需命令行参数，避免 Systemd 环境变量空值展开错位
    ARGS="-l 0.0.0.0:$PORT -p $PASSWORD -cert-type $CERT_TYPE"
    if [ "$CERT_TYPE" = "self-signed" ]; then
        if [ -n "$CERT_NAME" ]; then
            ARGS="$ARGS -cert-name $CERT_NAME"
        fi
    elif [ "$CERT_TYPE" = "acme" ]; then
        ARGS="$ARGS -cert-name $CERT_NAME -acme-http $ACME_HTTP -acme-cache $ACME_CACHE"
        if [ -n "$ACME_EMAIL" ]; then
            ARGS="$ARGS -acme-email $ACME_EMAIL"
        fi
    elif [ "$CERT_TYPE" = "custom" ]; then
        ARGS="$ARGS -cert-file $CERT_FILE -key-file $KEY_FILE"
    fi
    if [ -n "$FALLBACK" ]; then
        ARGS="$ARGS -fallback $FALLBACK"
    fi

    # 创建工作与配置文件夹
    CONF_DIR="/etc/mist"
    mkdir -p "$CONF_DIR"
    
    # 拷贝二进制
    cp ./mist-server /usr/local/bin/mist-server
    chmod +x /usr/local/bin/mist-server

    # 生成环境配置文件
    cat > "$CONF_DIR/server.conf" <<EOF
ARGS="$ARGS"
EOF

    # 生成 Systemd Service
    cat > /etc/systemd/system/mist-server.service <<EOF
[Unit]
Description=MIST Server Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$CONF_DIR
EnvironmentFile=$CONF_DIR/server.conf
ExecStart=/usr/local/bin/mist-server \$ARGS
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

    # 载入并启动
    systemctl daemon-reload
    systemctl enable mist-server
    systemctl restart mist-server

    echo -e "${GREEN}[SUCCESS]${NC} $(msg systemd_success)"
    
    # 获取服务器公网 IP
    PUBLIC_IP=$(curl -s https://ipinfo.io/ip || curl -s https://api.ipify.org || echo "PUBLIC_IP")

    echo -e "\n${BLUE}[4/4]${NC} $(msg deploy_result)"
    echo -e "=========================================================="
    echo -e "  $(msg status):   ${GREEN}$(msg status_active)${NC}"
    echo -e "  $(msg server_addr): ${CYAN}$PUBLIC_IP${NC}"
    echo -e "  $(msg server_port):   ${CYAN}$PORT${NC}"
    echo -e "  $(msg connect_pw):   ${YELLOW}$PASSWORD${NC}"
    echo -e "  $(msg cert_mode):   ${CYAN}$CERT_TYPE${NC}"

    if [ "$CERT_TYPE" = "self-signed" ]; then
        echo -e "  $(msg extract_fingerprint)"
        sleep 2
        # 从 journalctl 抓取生成的证书 sha256 指纹
        SHA256=$(journalctl -u mist-server -n 50 --no-pager | grep -oE "sha256 [0-9a-fA-F]{64}" | awk '{print $2}')
        if [ -n "$SHA256" ]; then
            echo -e "  $(msg cert_fingerprint):   ${GREEN}$SHA256${NC}"
            echo -e "\n  👉 ${PURPLE}$(msg client_cmd):${NC}"
            echo -e "  ./mist-client -l 127.0.0.1:1080 -s $PUBLIC_IP:$PORT -p $PASSWORD -tls-cert-sha256 $SHA256"
        else
            echo -e "  ${YELLOW}[INFO]${NC} $(msg fingerprint_hint):"
            echo -e "  journalctl -u mist-server -n 20 --no-pager"
        fi
    elif [ "$CERT_TYPE" = "acme" ]; then
        echo -e "\n  👉 ${PURPLE}$(msg client_cmd):${NC}"
        echo -e "  ./mist-client -l 127.0.0.1:1080 -s $CERT_NAME:$PORT -p $PASSWORD"
    else
        echo -e "  $(msg extract_custom)"
        sleep 2
        SHA256=$(journalctl -u mist-server -n 50 --no-pager | grep -oE "sha256 [0-9a-fA-F]{64}" | awk '{print $2}')
        if [ -n "$SHA256" ]; then
            echo -e "  $(msg cert_fingerprint):   ${GREEN}$SHA256${NC}"
        fi
        echo -e "\n  👉 ${PURPLE}$(msg client_conn_type):${NC}"
        echo -e "  $(msg client_conn_custom_hint)"
    fi
    echo -e "=========================================================="
else
    # 仅生成前台启动脚本
    echo -e "$(msg gen_start_sh)"
    cat > "./start.sh" <<EOF
#!/usr/bin/env bash
./mist-server \
    -l "0.0.0.0:$PORT" \
    -p "$PASSWORD" \
    -cert-type "$CERT_TYPE" \
    -cert-name "$CERT_NAME" \
    -acme-http "$ACME_HTTP" \
    -acme-cache "$ACME_CACHE" \
    -acme-email "$ACME_EMAIL" \
    -cert-file "$CERT_FILE" \
    -key-file "$KEY_FILE" \
    -fallback "$FALLBACK"
EOF
    chmod +x ./start.sh

    echo -e "${GREEN}[SUCCESS]${NC} $(msg start_sh_success)"
fi
