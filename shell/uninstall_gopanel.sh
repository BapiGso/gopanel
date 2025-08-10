#!/bin/sh

# Check operating system
OS=$(uname -s)

# Stop and disable gopanel service
if [ "$OS" = "Linux" ]; then
    sudo systemctl stop gopanel
    sudo systemctl disable gopanel
    sudo rm /etc/systemd/system/gopanel.service
    sudo systemctl daemon-reload
elif [ "$OS" = "FreeBSD" ]; then
    sudo service gopanel stop
    sudo sysrc gopanel_enable=NO
    sudo rm /usr/local/etc/rc.d/gopanel
elif [ "$OS" = "Darwin" ]; then
    PLIST_PATH="/Library/LaunchDaemons/com.gopanel.service.plist"
    if [ -f "$PLIST_PATH" ]; then
        sudo launchctl unload -w "$PLIST_PATH"
        sudo rm "$PLIST_PATH"
    else
        echo "LaunchDaemon configuration file not found: $PLIST_PATH"
    fi
else
    echo "Unsupported operating system: $OS, unable to stop service."
    exit 1
fi

# Remove gopanel binary
if [ -f "/usr/local/bin/gopanel" ]; then
    sudo rm /usr/local/bin/gopanel
    echo "Gopanel binary removed."
else
    echo "Gopanel binary does not exist, skipping removal."
fi

# Remove configuration files
CONFIG_DIR="/opt/gopanel"
if [ -d "$CONFIG_DIR" ]; then
    echo "Removing configuration files..."
    sudo rm -f "$CONFIG_DIR/gopanel_config.json"
    sudo rm -f "$CONFIG_DIR/gopanel_Caddyfile"
    sudo rm -f "$CONFIG_DIR/gopanel_frps.conf"
    sudo rm -f "$CONFIG_DIR/gopanel_frpc.conf"
    
    # Ask whether to remove the entire working directory
    echo "Do you want to remove the entire working directory $CONFIG_DIR? (y/n)"
    read -r response
    if [ "$response" = "y" ] || [ "$response" = "Y" ]; then
        sudo rm -rf "$CONFIG_DIR"
        echo "Working directory removed."
    else
        echo "Working directory retained."
    fi
else
    echo "Configuration directory does not exist, skipping removal."
fi

# Clean up log files (macOS only)
if [ "$OS" = "Darwin" ]; then
    sudo rm -f /var/log/gopanel.err
    sudo rm -f /var/log/gopanel.out
fi

echo "Gopanel has been successfully uninstalled."