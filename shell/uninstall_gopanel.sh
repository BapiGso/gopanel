#!/bin/sh

# 检查操作系统
OS=$(uname -s)

# 停止并禁用 gopanel 服务
if [ "$OS" = "Linux" ]; then
    sudo systemctl stop gopanel
    sudo systemctl disable gopanel
    sudo rm /etc/systemd/system/gopanel.service
elif [ "$OS" = "FreeBSD" ]; then
    sudo service gopanel stop
    sudo sysrc gopanel_enable=NO
    sudo rm /usr/local/etc/rc.d/gopanel
else
    echo "不支持的操作系统: $OS，无法停止服务。"
    exit 1
fi

# 删除 gopanel 二进制文件
sudo rm /usr/local/bin/gopanel

# 删除配置文件
sudo rm /usr/local/bin/gopanel_config.json
sudo rm /usr/local/bin/gopanel_Caddyfile
sudo rm /usr/local/bin/gopanel_frps.conf

echo "gopanel 已成功卸载。"