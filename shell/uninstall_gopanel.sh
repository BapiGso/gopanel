#!/bin/sh
set -eu

PATH="${PATH:-/usr/bin:/bin}:/usr/local/sbin:/usr/sbin:/sbin"
export PATH

WORKDIR="/opt/gopanel"
TARGET_BIN="/usr/local/bin/gopanel"
LABEL="com.gopanel.service"
PRIVILEGE_CMD=""
INIT_SYSTEM=""
PURGE=false

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

# Keep runtime detection aligned with install_gopanel.sh; tools alone do not identify PID 1.
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

case "${1:-}" in
    "") ;;
    --purge) PURGE=true ;;
    -h|--help)
        printf 'Usage: sh uninstall_gopanel.sh [--purge]\nBy default, configuration and data in /opt/gopanel are retained.\n'
        exit 0
        ;;
    *) fail "Unknown argument: $1. Use --purge to remove configuration and data." ;;
esac
[ "$#" -le 1 ] || fail "Too many arguments."
for TOOL in uname id rm; do
    require_command "$TOOL"
done
OS=$(uname -s)
case "$OS" in
    Linux|FreeBSD|Darwin) ;;
    *) fail "Unsupported operating system: $OS. Uninstall the manually deployed binary yourself." ;;
esac
setup_privilege
detect_init_system

if service_is_registered; then
    stop_service || fail "Failed to stop GoPanel; installation files were not removed."
    if service_is_running > /dev/null 2>&1; then
        fail "GoPanel is still running; installation files were not removed."
    fi
    case "$INIT_SYSTEM" in
        systemd) as_root systemctl disable gopanel || fail "Failed to disable GoPanel." ;;
        openrc)
            RUNLEVEL_SERVICES=$(as_root rc-update show default) || fail "Failed to inspect OpenRC runlevel."
            if printf '%s\n' "$RUNLEVEL_SERVICES" | grep -Eq '^[[:space:]]*gopanel([[:space:]]|$)'; then
                as_root rc-update del gopanel default || fail "Failed to disable GoPanel."
            fi
            ;;
        freebsd) as_root sysrc gopanel_enable=NO || fail "Failed to disable GoPanel." ;;
        launchd) as_root launchctl disable "system/$LABEL" || fail "Failed to disable GoPanel." ;;
    esac
    as_root rm -f "$SERVICE_FILE" || fail "Failed to remove service configuration."
    if [ "$INIT_SYSTEM" = "systemd" ]; then
        as_root systemctl daemon-reload || fail "Failed to reload systemd."
    fi
fi
as_root rm -f "$TARGET_BIN" || fail "Failed to remove GoPanel binary."

if [ "$OS" = "Darwin" ]; then
    as_root rm -f /var/log/gopanel.err /var/log/gopanel.out || fail "Failed to remove GoPanel logs."
fi
if [ -d "$WORKDIR" ]; then
    if [ "$PURGE" = false ] && [ -t 0 ]; then
        printf 'Remove all configuration and data in %s? [y/N] ' "$WORKDIR"
        RESPONSE=""
        read -r RESPONSE || RESPONSE=""
        case "$RESPONSE" in
            y|Y) PURGE=true ;;
        esac
    fi
    if [ "$PURGE" = true ]; then
        as_root rm -rf "$WORKDIR" || fail "Failed to remove the working directory."
        printf 'Working directory removed.\n'
    else
        printf 'Configuration and data retained in %s. Use --purge to remove them.\n' "$WORKDIR"
    fi
fi

printf 'GoPanel has been successfully uninstalled.\n'
