#!/bin/sh
# Check if wget exists
if ! command -v wget > /dev/null 2>&1; then
    echo "wget is not installed, please install wget first."
    exit 1
fi
# Detect operating system
OS=$(uname -s)
# Detect system architecture
ARCH=$(uname -m)
# Map system architecture to Go arch
case "$ARCH" in
    x86_64)
        GOARCH="amd64"
        ;;
    arm64|aarch64)
        GOARCH="arm64"
        ;;
    armv7l)
        GOARCH="arm"
        ;;
    mips|mipsel)
        GOARCH="mips"
        ;;
    s390x)
        GOARCH="s390x"
        ;;
    riscv64)
        GOARCH="riscv64"
        ;;
    *)
        echo "Unsupported system architecture: $ARCH. Please manually download the appropriate gopanel version for your system."
        exit 1
        ;;
esac
# Convert OS name to lowercase
OS_LOWER=$(echo "$OS" | tr '[:upper:]' '[:lower:]')
# Check if operating system is supported
if [ "$OS" != "Linux" ] && [ "$OS" != "FreeBSD" ] && [ "$OS" != "Darwin" ]; then
    echo "Unsupported operating system: $OS. Please manually download the appropriate gopanel version for your system."
    exit 1
fi
# Detect init system on Linux
INIT_SYSTEM=""
if [ "$OS" = "Linux" ]; then
    if command -v systemctl > /dev/null 2>&1; then
        INIT_SYSTEM="systemd"
    elif command -v rc-service > /dev/null 2>&1; then
        INIT_SYSTEM="openrc"
    else
        echo "Unsupported init system. Only systemd and OpenRC (Alpine) are supported."
        exit 1
    fi
fi
WORKDIR="/opt/gopanel"
# Set download link
DOWNLOAD_URL="https://github.com/BapiGso/gopanel/releases/latest/download/gopanel_${OS_LOWER}_${GOARCH}"
# Check if this is an update
IS_UPDATE=false
if [ -f /usr/local/bin/gopanel ]; then
    IS_UPDATE=true
    echo "Detected existing gopanel installation, updating..."
    # Stop service before update
    if [ "$OS" = "Linux" ]; then
        if [ "$INIT_SYSTEM" = "systemd" ]; then
            sudo systemctl stop gopanel 2>/dev/null
        elif [ "$INIT_SYSTEM" = "openrc" ]; then
            sudo rc-service gopanel stop 2>/dev/null
        fi
    elif [ "$OS" = "FreeBSD" ]; then
        sudo service gopanel stop 2>/dev/null
    elif [ "$OS" = "Darwin" ]; then
        sudo launchctl unload /Library/LaunchDaemons/com.gopanel.service.plist 2>/dev/null
    fi
fi
# Download gopanel
sudo wget "$DOWNLOAD_URL" -O /usr/local/bin/gopanel
# Check if download succeeded
if [ $? -ne 0 ]; then
    echo "Failed to download gopanel. Please check your network connection or download manually."
    exit 1
fi
# Grant execute permission
sudo chmod +x /usr/local/bin/gopanel
# If updating, restart service and exit
if [ "$IS_UPDATE" = true ]; then
    if [ "$OS" = "Linux" ]; then
        if [ "$INIT_SYSTEM" = "systemd" ]; then
            sudo systemctl start gopanel
        elif [ "$INIT_SYSTEM" = "openrc" ]; then
            sudo rc-service gopanel start
        fi
    elif [ "$OS" = "FreeBSD" ]; then
        sudo service gopanel start
    elif [ "$OS" = "Darwin" ]; then
        sudo launchctl load -w /Library/LaunchDaemons/com.gopanel.service.plist
    fi
    sleep 2
    echo "gopanel update success!"
    exit 0
fi
# Create working directory
sudo mkdir -p "$WORKDIR"
# Create service file
if [ "$OS" = "Linux" ]; then
    if [ "$INIT_SYSTEM" = "systemd" ]; then
        cat << EOF | sudo tee /etc/systemd/system/gopanel.service > /dev/null
[Unit]
Description=GoPanel Service
After=network.target
[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/gopanel -w ${WORKDIR}
Restart=on-failure
WorkingDirectory=${WORKDIR}
[Install]
WantedBy=multi-user.target
EOF
        # Reload systemd configuration
        sudo systemctl daemon-reload
        # Enable and start gopanel service
        sudo systemctl enable gopanel
        sudo systemctl start gopanel
        # Check service status
        sleep 2
        sudo systemctl status gopanel
    elif [ "$INIT_SYSTEM" = "openrc" ]; then
        cat << EOF | sudo tee /etc/init.d/gopanel > /dev/null
#!/sbin/openrc-run
description="GoPanel Service"
command="/usr/local/bin/gopanel"
command_args="-w ${WORKDIR}"
command_background=true
pidfile="/run/\${RC_SVCNAME}.pid"
directory="${WORKDIR}"

depend() {
    need net
    after firewall
}
EOF
        # Grant execute permission to init script
        sudo chmod +x /etc/init.d/gopanel
        # Enable and start gopanel service
        sudo rc-update add gopanel default
        sudo rc-service gopanel start
        # Check service status
        sleep 2
        sudo rc-service gopanel status
    fi
elif [ "$OS" = "FreeBSD" ]; then
    cat << EOF | sudo tee /usr/local/etc/rc.d/gopanel > /dev/null
#!/bin/sh
# PROVIDE: gopanel
# REQUIRE: DAEMON NETWORKING
# KEYWORD: shutdown
. /etc/rc.subr
name="gopanel"
rcvar="\${name}_enable"
load_rc_config \${name}
: \${gopanel_enable:="NO"}
: \${gopanel_user:="root"}
command="/usr/local/bin/gopanel"
command_args="-w ${WORKDIR}"
run_rc_command "\$1"
EOF
    # Grant execute permission to rc.d script
    sudo chmod +x /usr/local/etc/rc.d/gopanel
    # Enable and start gopanel service
    sudo sysrc gopanel_enable=YES
    sudo service gopanel start
    # Check service status
    sleep 2
    sudo service gopanel status
elif [ "$OS" = "Darwin" ]; then
    # Create LaunchDaemon
    PLIST_PATH="/Library/LaunchDaemons/com.gopanel.service.plist"
    cat << EOF | sudo tee "$PLIST_PATH" > /dev/null
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.gopanel.service</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/gopanel</string>
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
    # Set permissions
    sudo chown root:wheel "$PLIST_PATH"
    sudo chmod 644 "$PLIST_PATH"
    # Load and start service
    sudo launchctl load -w "$PLIST_PATH"
    # Check service status
    sleep 2
    sudo launchctl print system/com.gopanel.service
else
    echo "Unsupported operating system: $OS. Cannot start service."
    exit 1
fi
echo "gopanel install success!"
