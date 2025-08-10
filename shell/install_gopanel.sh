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
# Set download link
DOWNLOAD_URL="https://github.com/BapiGso/gopanel/releases/latest/download/gopanel_${OS_LOWER}_${GOARCH}"
# Download gopanel
sudo wget "$DOWNLOAD_URL" -O /usr/local/bin/gopanel
# Check if download succeeded
if [ $? -ne 0 ]; then
    echo "Failed to download gopanel. Please check your network connection or download manually."
    exit 1
fi
# Grant execute permission
sudo chmod +x /usr/local/bin/gopanel
# Create working directory
WORKDIR="/opt/gopanel"
sudo mkdir -p "$WORKDIR"
# Create service file
if [ "$OS" = "Linux" ]; then
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