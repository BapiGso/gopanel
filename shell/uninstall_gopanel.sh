#!/bin/sh

# 检查操作系统
OS=$(uname -s)

# 停止并禁用 gopanel 服务
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
        echo "找不到 LaunchDaemon 配置文件：$PLIST_PATH"
    fi
else
    echo "不支持的操作系统: $OS，无法停止服务。"
    exit 1
fi

# 删除 gopanel 二进制文件
sudo rm /usr/local/bin/gopanel

# 删除配置文件
CONFIG_DIR="/opt/gopanel"
sudo rm -f "$CONFIG_DIR/gopanel_config.json"
sudo rm -f "$CONFIG_DIR/gopanel_Caddyfile"
sudo rm -f "$CONFIG_DIR/gopanel_frps.conf"
sudo rm -f "$CONFIG_DIR/gopanel_frpc.conf"

echo "gopanel 已成功卸载。"