#!/bin/sh
set -eu

PATH="${PATH:-/usr/bin:/bin}:/usr/local/sbin:/usr/sbin:/sbin"
export PATH

WORKDIR="/opt/gopanel"
BIN_DIR="/usr/local/bin"
TARGET_BIN="$BIN_DIR/gopanel"
LABEL="com.gopanel.service"
PRIVILEGE_CMD=""
INIT_SYSTEM=""
TMP_BIN=""
BACKUP_BIN=""
TMP_SERVICE=""
IS_UPDATE=false
WAS_RUNNING=false
SERVICE_STOPPED=false
BIN_REPLACED=false
SERVICE_WRITTEN=false

fail() {
    printf 'Error: %s\n' "$*" >&2
    exit 1
}

require_command() {
    command -v "$1" > /dev/null 2>&1 || fail "Required command not found: $1. Please install it first."
}

as_root() {
    if [ -n "$PRIVILEGE_CMD" ]; then
        "$PRIVILEGE_CMD" "$@"
    else
        "$@"
    fi
}

setup_privilege() {
    if [ "$(id -u)" -eq 0 ]; then
        return
    fi
    if command -v sudo > /dev/null 2>&1; then
        PRIVILEGE_CMD="sudo"
        sudo -v || fail "Unable to obtain root privileges with sudo."
    elif command -v doas > /dev/null 2>&1; then
        PRIVILEGE_CMD="doas"
        doas true || fail "Unable to obtain root privileges with doas."
    else
        fail "Run this script as root, or install and configure sudo or doas."
    fi
}

# Keep runtime detection aligned with uninstall_gopanel.sh; tools alone do not identify PID 1.
detect_init_system() {
    case "$OS" in
        Linux)
            PID1=""
            if [ -r /proc/1/comm ]; then
                read -r PID1 < /proc/1/comm || true
            elif command -v ps > /dev/null 2>&1; then
                PID1=$(ps -p 1 -o comm= 2>/dev/null) || PID1=""
            fi
            if [ "$PID1" = "systemd" ] && [ -d /run/systemd/system ]; then
                require_command systemctl
                INIT_SYSTEM="systemd"
                SERVICE_DIR="/etc/systemd/system"
                SERVICE_FILE="$SERVICE_DIR/gopanel.service"
            elif [ -f /run/openrc/softlevel ]; then
                case "$PID1" in
                    init|/sbin/init|busybox|openrc-init) ;;
                    *) fail "OpenRC runtime files exist, but PID 1 ($PID1) is not an init process." ;;
                esac
                require_command rc-service
                require_command rc-update
                require_command openrc-run
                require_command grep
                OPENRC_RUN=$(command -v openrc-run)
                INIT_SYSTEM="openrc"
                SERVICE_DIR="/etc/init.d"
                SERVICE_FILE="$SERVICE_DIR/gopanel"
            else
                fail "No supported running init system detected (PID 1: $PID1). Use systemd/OpenRC, or download and run GoPanel manually in containers and other init systems."
            fi
            ;;
        FreeBSD)
            require_command service
            require_command sysrc
            [ -r /etc/rc.subr ] || fail "FreeBSD rc.subr is missing."
            [ -x /usr/sbin/daemon ] || fail "FreeBSD daemon is missing."
            INIT_SYSTEM="freebsd"
            SERVICE_DIR="/usr/local/etc/rc.d"
            SERVICE_FILE="$SERVICE_DIR/gopanel"
            ;;
        Darwin)
            require_command launchctl
            require_command grep
            require_command chown
            INIT_SYSTEM="launchd"
            SERVICE_DIR="/Library/LaunchDaemons"
            SERVICE_FILE="$SERVICE_DIR/$LABEL.plist"
            ;;
        *) fail "Unsupported operating system: $OS. Download the appropriate release and run it manually." ;;
    esac
}

detect_platform() {
    OS=$(uname -s)
    ARCH=$(uname -m)
    case "$OS" in
        Linux) GOOS="linux"; ASSET_OS="linux" ;;
        FreeBSD) GOOS="freebsd"; ASSET_OS="freebsd" ;;
        Darwin) GOOS="darwin"; ASSET_OS="mac" ;;
        *) fail "Unsupported operating system: $OS. Download the appropriate release and run it manually." ;;
    esac
    case "$ARCH" in
        x86_64|amd64) GOARCH="amd64" ;;
        arm64|aarch64) GOARCH="arm64" ;;
        armv7l|armv7) GOARCH="arm" ;;
        s390x) GOARCH="s390x" ;;
        riscv64) GOARCH="riscv64" ;;
        *) fail "No prebuilt GoPanel release for $OS/$ARCH. Please build and install it manually." ;;
    esac
    # Only accept combinations built by .github/workflows/release.yaml.
    case "$GOOS/$GOARCH" in
        linux/amd64|linux/arm64|linux/arm|linux/s390x|linux/riscv64|darwin/amd64|darwin/arm64|freebsd/amd64) ;;
        *) fail "No prebuilt GoPanel release for $OS/$ARCH. Please build and install it manually." ;;
    esac
    DOWNLOAD_URL="https://github.com/BapiGso/gopanel/releases/latest/download/gopanel_${ASSET_OS}_${GOARCH}"
}

service_is_registered() {
    [ -f "$SERVICE_FILE" ] && return 0
    case "$INIT_SYSTEM" in
        systemd) as_root systemctl cat gopanel > /dev/null 2>&1 ;;
        openrc) as_root rc-service --exists gopanel > /dev/null 2>&1 ;;
        launchd) as_root launchctl print "system/$LABEL" > /dev/null 2>&1 ;;
        *) return 1 ;;
    esac
}

service_is_running() {
    case "$INIT_SYSTEM" in
        systemd) as_root systemctl is-active --quiet gopanel ;;
        openrc) as_root rc-service gopanel status ;;
        freebsd) as_root service gopanel onestatus ;;
        launchd)
            SERVICE_STATE=$(as_root launchctl print "system/$LABEL" 2>/dev/null) || return 1
            printf '%s\n' "$SERVICE_STATE" | grep -Eq '^[[:space:]]*state = running[[:space:]]*$'
            ;;
    esac
}

stop_service() {
    case "$INIT_SYSTEM" in
        systemd) as_root systemctl stop gopanel ;;
        openrc) as_root rc-service gopanel stop ;;
        freebsd)
            if service_is_running > /dev/null 2>&1; then
                as_root service gopanel onestop
            fi
            ;;
        launchd)
            if as_root launchctl print "system/$LABEL" > /dev/null 2>&1; then
                as_root launchctl bootout "system/$LABEL"
            fi
            ;;
    esac
}

start_service() {
    case "$INIT_SYSTEM" in
        systemd)
            as_root systemctl reset-failed gopanel || return
            as_root systemctl start gopanel
            ;;
        openrc) as_root rc-service gopanel start ;;
        freebsd) as_root service gopanel start ;;
        launchd)
            as_root launchctl enable "system/$LABEL" || return
            as_root launchctl bootstrap system "$SERVICE_FILE"
            ;;
    esac
}

wait_for_service() {
    ATTEMPT=0
    sleep 2 || return
    while [ "$ATTEMPT" -lt 10 ]; do
        if service_is_running > /dev/null 2>&1; then
            return 0
        fi
        ATTEMPT=$((ATTEMPT + 1))
        sleep 1 || return
    done
    return 1
}

show_service_status() {
    case "$INIT_SYSTEM" in
        systemd) as_root systemctl --no-pager --full status gopanel ;;
        openrc) as_root rc-service gopanel status ;;
        freebsd) as_root service gopanel onestatus ;;
        launchd) as_root launchctl print "system/$LABEL" ;;
    esac
}

write_service() {
    TMP_SERVICE=$(as_root mktemp "$SERVICE_DIR/.gopanel.service.XXXXXX") || fail "Failed to create temporary service file."
    case "$INIT_SYSTEM" in
        systemd)
            as_root tee "$TMP_SERVICE" > /dev/null <<EOF || fail "Failed to write systemd service."
[Unit]
Description=GoPanel Service
After=network.target
[Service]
Type=simple
User=root
ExecStart=${TARGET_BIN} -w ${WORKDIR}
Restart=on-failure
RestartSec=2
WorkingDirectory=${WORKDIR}
[Install]
WantedBy=multi-user.target
EOF
            as_root chmod 644 "$TMP_SERVICE" || fail "Failed to set systemd service permissions."
            ;;
        openrc)
            as_root tee "$TMP_SERVICE" > /dev/null <<EOF || fail "Failed to write OpenRC service."
#!${OPENRC_RUN}
description="GoPanel Service"
command="${TARGET_BIN}"
command_args="-w ${WORKDIR}"
command_background=true
pidfile="/run/\${RC_SVCNAME}.pid"
directory="${WORKDIR}"

depend() {
    need net
    after firewall
}
EOF
            as_root chmod 755 "$TMP_SERVICE" || fail "Failed to set OpenRC service permissions."
            ;;
        freebsd)
            as_root tee "$TMP_SERVICE" > /dev/null <<EOF || fail "Failed to write FreeBSD service."
#!/bin/sh
# PROVIDE: gopanel
# REQUIRE: DAEMON NETWORKING
# KEYWORD: shutdown
. /etc/rc.subr
name="gopanel"
rcvar="\${name}_enable"
load_rc_config "\${name}"
: \${gopanel_enable:="NO"}
: \${gopanel_user:="root"}
gopanel_chdir="${WORKDIR}"
required_files="${TARGET_BIN}"
pidfile="/var/run/\${name}.pid"
command="/usr/sbin/daemon"
procname="${TARGET_BIN}"
# Track the application PID so rc.d status cannot mistake a launcher for GoPanel.
command_args="-f -p \${pidfile} ${TARGET_BIN} -w ${WORKDIR}"
run_rc_command "\$1"
EOF
            as_root chmod 755 "$TMP_SERVICE" || fail "Failed to set FreeBSD service permissions."
            ;;
        launchd)
            as_root tee "$TMP_SERVICE" > /dev/null <<EOF || fail "Failed to write launchd service."
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>${LABEL}</string>
    <key>ProgramArguments</key>
    <array>
        <string>${TARGET_BIN}</string>
        <string>-w</string>
        <string>${WORKDIR}</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardErrorPath</key>
    <string>/var/log/gopanel.err</string>
    <key>StandardOutPath</key>
    <string>/var/log/gopanel.out</string>
    <key>WorkingDirectory</key>
    <string>${WORKDIR}</string>
</dict>
</plist>
EOF
            as_root chown root:wheel "$TMP_SERVICE" || fail "Failed to set launchd service ownership."
            as_root chmod 644 "$TMP_SERVICE" || fail "Failed to set launchd service permissions."
            ;;
    esac
    as_root mv -f "$TMP_SERVICE" "$SERVICE_FILE" || fail "Failed to replace service configuration."
    TMP_SERVICE=""
    SERVICE_WRITTEN=true
    case "$INIT_SYSTEM" in
        systemd)
            as_root systemctl daemon-reload || fail "Failed to reload systemd."
            as_root systemctl enable gopanel || fail "Failed to enable GoPanel."
            ;;
        openrc) as_root rc-update add gopanel default || fail "Failed to enable GoPanel." ;;
        freebsd) as_root sysrc gopanel_enable=YES || fail "Failed to enable GoPanel." ;;
    esac
}

cleanup() {
    RESULT=$?
    trap - 0
    set +e
    if [ "$RESULT" -ne 0 ]; then
        if [ "$BIN_REPLACED" = true ] && [ "$SERVICE_WRITTEN" = true ]; then
            stop_service > /dev/null 2>&1 || printf 'Warning: unable to stop the failed installation.\n' >&2
        fi
        if [ "$BIN_REPLACED" = true ] && [ -n "$BACKUP_BIN" ]; then
            printf 'Restoring the previous GoPanel binary...\n' >&2
            if as_root mv -f "$BACKUP_BIN" "$TARGET_BIN"; then
                BACKUP_BIN=""
                BIN_REPLACED=false
            else
                printf 'Warning: restore failed; the previous binary is retained at %s.\n' "$BACKUP_BIN" >&2
            fi
        fi
        if [ "$SERVICE_STOPPED" = true ] && [ "$WAS_RUNNING" = true ] && [ "$BIN_REPLACED" = false ]; then
            if start_service && wait_for_service; then
                printf 'Previous GoPanel service restarted.\n' >&2
            else
                printf 'Warning: unable to restart the previous service; check the service status manually.\n' >&2
            fi
        fi
    fi
    for TEMP_FILE in "$TMP_BIN" "$TMP_SERVICE"; do
        if [ -n "$TEMP_FILE" ]; then
            as_root rm -f "$TEMP_FILE" || printf 'Warning: unable to remove %s.\n' "$TEMP_FILE" >&2
        fi
    done
    if [ -n "$BACKUP_BIN" ] && { [ "$RESULT" -eq 0 ] || [ "$BIN_REPLACED" = false ]; }; then
        as_root rm -f "$BACKUP_BIN" || printf 'Warning: unable to remove %s.\n' "$BACKUP_BIN" >&2
    fi
    exit "$RESULT"
}

for TOOL in uname id mktemp mkdir mv cp chmod tee rm sleep; do
    require_command "$TOOL"
done
detect_platform
setup_privilege
detect_init_system
if command -v curl > /dev/null 2>&1; then
    DOWNLOADER="curl"
elif command -v wget > /dev/null 2>&1; then
    DOWNLOADER="wget"
else
    fail "Install curl or wget first (for example: apk add curl, apt install curl, or pkg install curl)."
fi

trap cleanup 0
trap 'exit 130' INT
trap 'exit 143' TERM
trap 'exit 129' HUP

if [ -f "$TARGET_BIN" ]; then
    IS_UPDATE=true
    printf 'Detected existing GoPanel installation, updating...\n'
fi
as_root mkdir -p "$BIN_DIR" "$WORKDIR" "$SERVICE_DIR" || fail "Failed to create installation directories."
TMP_BIN=$(as_root mktemp "$BIN_DIR/.gopanel.download.XXXXXX") || fail "Failed to create temporary binary file."
# Download before stopping the old service, then replace on the same filesystem.
if [ "$DOWNLOADER" = "curl" ]; then
    as_root curl --fail --location --silent --show-error --connect-timeout 15 --max-time 300 --output "$TMP_BIN" "$DOWNLOAD_URL" || fail "Failed to download GoPanel from $DOWNLOAD_URL."
else
    as_root wget -T 30 -O "$TMP_BIN" "$DOWNLOAD_URL" || fail "Failed to download GoPanel from $DOWNLOAD_URL."
fi
[ -s "$TMP_BIN" ] || fail "Downloaded GoPanel binary is empty."
as_root chmod 755 "$TMP_BIN" || fail "Failed to set binary permissions."
if [ "$IS_UPDATE" = true ]; then
    BACKUP_BIN=$(as_root mktemp "$BIN_DIR/.gopanel.backup.XXXXXX") || fail "Failed to create binary backup."
    as_root cp -p "$TARGET_BIN" "$BACKUP_BIN" || fail "Failed to back up the existing binary."
fi
if service_is_registered; then
    if service_is_running > /dev/null 2>&1; then
        WAS_RUNNING=true
    fi
    stop_service || fail "Failed to stop the existing service; the binary was not replaced."
    SERVICE_STOPPED=true
fi
as_root mv -f "$TMP_BIN" "$TARGET_BIN" || fail "Failed to replace GoPanel binary."
TMP_BIN=""
BIN_REPLACED=true
# Refresh service definitions on updates too, including fixes for older installations.
write_service
if ! start_service; then
    show_service_status || true
    fail "Failed to start GoPanel."
fi
if ! wait_for_service; then
    show_service_status || true
    fail "GoPanel did not reach the running state. Check the service logs."
fi
if [ "$IS_UPDATE" = true ]; then
    printf 'GoPanel update success!\n'
else
    printf 'GoPanel install success!\n'
fi
