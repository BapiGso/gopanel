#!/bin/bash
PATH=/bin:/sbin:/usr/bin:/usr/sbin:/usr/local/bin:/usr/local/sbin:~/bin
export PATH
install_tmp='/tmp/bt_install.pl'
public_file=/www/server/panel/install/public.sh
[ ! -f $public_file ] && wget -O $public_file http://download.bt.cn/install/public.sh -T 5;

publicFileMd5=$(md5sum ${public_file}|awk '{print $1}')
md5check="84cc939e9cda4405b6fb0c6f5bbc93b5"
[ "${publicFileMd5}" != "${md5check}"  ] && wget -O $public_file http://download.bt.cn/install/public.sh -T 5;

. $public_file
download_Url=$NODE_URL
sysType=$(uname -a|grep x86_64)

if [ "${PM}" == "yum" ];then
	Centos7Check=$(cat /etc/redhat-release|grep ' 7.'|grep -i centos)
	Centos8Check=$(cat /etc/redhat-release|grep ' 8.'|grep -i centos)
	if [ "${Centos7Check}" ]; then
		OS_SYS="rhel"
		OS_VER="70"
	elif [ "${Centos8Check}" ]; then
		OS_SYS="rhel"
		OS_VER="80"
	fi
fi

if  [ "${PM}" == "apt-get" ]; then
	UBUNTU_VER=$(cat /etc/issue|grep -i ubuntu|awk '{print $2}'|cut -d. -f1)
	DEBIAN_VER=$(cat /etc/issue|grep -i debian|awk '{print $3}')
	if [ "${UBUNTU_VER}" == "18" ] || [ "${UBUNTU_VER}" == "20" ];then
		OS_SYS="ubuntu"
		OS_VER="${UBUNTU_VER}04"
	elif [ "${DEBIAN_VER}" == "10" ]; then
		OS_SYS="debian"
		OS_VER="${DEBIAN_VER}"
	fi
fi

if [ -z "${OS_VER}" ] && [ "$1" == "install" ];then
	wget -O mongodb.sh ${download_Url}/install/0/old/mongodb.sh && sh mongodb.sh $1 $2
	exit;
fi

mongodb_version="4.4.6"
mongodb_tools_version="100.3.1"
mongodb_path=/www/server/mongodb


Service_Add(){
	if [ "${PM}" == "yum" ] || [ "${PM}" == "dnf" ]; then
		chkconfig --add mongodb
		chkconfig --level 2345 mongodb on
	elif [ "${PM}" == "apt-get" ]; then
		update-rc.d mongodb defaults
	fi
}
Service_Del(){
 	if [ "${PM}" == "yum" ] || [ "${PM}" == "dnf" ]; then
 		chkconfig --del mongodb
	elif [ "${PM}" == "apt-get" ]; then
		update-rc.d mongodb remove
	fi
}
Install_mongodb()
{
	if [ ! -f /www/server/mongodb/bin/mongo ];then
		wget -O mongodb-linux-x86_64-${OS_SYS}${OS_VER}-${mongodb_version}.tgz $download_Url/src/mongodb-linux-x86_64-${OS_SYS}${OS_VER}-${mongodb_version}.tgz -T 5
		tar zxvf mongodb-linux-x86_64-${OS_SYS}${OS_VER}-${mongodb_version}.tgz
		mkdir -p $mongodb_path/data
		mkdir -p $mongodb_path/log
		\cp -a -r mongodb-linux-x86_64-${OS_SYS}${OS_VER}-${mongodb_version}/bin $mongodb_path/
		rm -rf mongodb-linux-x86_64-${OS_SYS}${OS_VER}-${mongodb_version}*

		wget -O mongodb-database-tools-${OS_SYS}${OS_VER}-x86_64-${mongodb_tools_version}.tgz $download_Url/src/mongodb-database-tools-${OS_SYS}${OS_VER}-x86_64-${mongodb_tools_version}.tgz
		tar -xvf mongodb-database-tools-${OS_SYS}${OS_VER}-x86_64-${mongodb_tools_version}.tgz
		\cp -rpa mongodb-database-tools-${OS_SYS}${OS_VER}-x86_64-${mongodb_tools_version}/bin/* $mongodb_path/bin
		rm -rf mongodb-database-tools-${OS_SYS}${OS_VER}-x86_64-${mongodb_tools_version}*

		groupadd mongo
		useradd -s /sbin/nologin -M -g mongo mongo

		chmod +x $mongodb_path/bin
		ln -sf $mongodb_path/bin/* /usr/bin/

		wget -O /etc/init.d/mongodb $download_Url/install/lib/plugin/mongodb/mongodb.init -T 5
		wget -O $mongodb_path/config.conf $download_Url/install/lib/plugin/mongodb/config.conf -T 5
		chmod +x /etc/init.d/mongodb
		chown -R mongo:mongo $mongodb_path
		/etc/init.d/mongodb start

		echo "${mongodb_version}" > ${mongodb_path}/version.pl
		echo "${mongodb_version}" > ${mongodb_path}/version_check.pl
	fi

	mkdir -p /www/server/panel/plugin/mongodb
	echo '正在安装脚本文件...' > $install_tmp
	grep "English" /www/server/panel/config/config.json
	if [ "$?" -ne 0 ];then
		wget -O /www/server/panel/plugin/mongodb/mongodb_main.py $download_Url/install/plugin/mongodb/mongodb_main.py -T 5
		wget -O /www/server/panel/plugin/mongodb/index.html $download_Url/install/plugin/mongodb/index.html -T 5
		wget -O /www/server/panel/plugin/mongodb/info.json $download_Url/install/plugin/mongodb/info.json -T 5
		wget -O /www/server/panel/plugin/mongodb/icon.png $download_Url/install/plugin/mongodb/icon.png -T 5
	else
		wget -O /www/server/panel/plugin/mongodb/mongodb_main.py $download_Url/install/plugin/mongodb_en/mongodb_main.py -T 5
		wget -O /www/server/panel/plugin/mongodb/index.html $download_Url/install/plugin/mongodb_en/index.html -T 5
		wget -O /www/server/panel/plugin/mongodb/info.json $download_Url/install/plugin/mongodb_en/info.json -T 5
		wget -O /www/server/panel/plugin/mongodb/icon.png $download_Url/install/plugin/mongodb_en/icon.png -T 5
	fi
	\cp -a -r /www/server/panel/plugin/mongodb/icon.png /www/server/panel/static/img/soft_ico/ico-mongodb.png
	echo "${mongodb_version}" > /www/server/panel/plugin/mongodb/version.pl
	echo "${mongodb_version}" > /www/server/panel/plugin/mongodb/version_check.pl
	echo '安装完成' > $install_tmp
}

Uninstall_mongodb()
{
	mongodb_path=/www/server/mongodb
	/etc/init.d/mongodb stop
	rm -f /etc/init.d/mongodb
	rm -f /usr/bin/mongo*
	rm -f /usr/bin/bsondump /usr/bin/install_compass
	rm -rf $mongodb_path/bin
	rm -rf $mongodb_path/log
	rm -rf /www/server/panel/plugin/mongodb
}
Update_mongodb(){
	mongoV=$(mongo --version|grep "shell version"|awk '{print $4}')
	if [ "${mongoV}" != "v4.4.2" ];then
		echo "当前版本无法升级至v4.4.2版本"
		exit 1;
	fi
	CompatibilityVersion=$(mongo --eval 'db.adminCommand( { getParameter: 1, featureCompatibilityVersion: 1 } )'|grep CompatibilityVersion|tr -d '{":}'|awk '{print $3}')
	if [ "${CompatibilityVersion}" != "4.0" ];then
		if  [ "${CompatibilityVersion}" != "3.6" ]; then
			echo "当前版本无法升级至${mongodb_version:0:3}版本"
			exit
		fi
	fi

	cd ${mongodb_path}
	wget -O src.tgz ${download_Url}/src/mongodb-linux-x86_64-${mongodb_version}.tgz -T 5
	tar -xvf src.tgz
	mv mongodb-linux-x86_64-${mongodb_version} src

	/etc/init.d/mongodb stop
	sleep 1
	[ -d "/www/server/mongoBak" ] && rm -rf /www/server/mongoBak
	\cp -rpf ${mongodb_path} /www/server/mongoBak
	\cp -pf ${mongodb_path}/src/bin/* ${mongodb_path}/bin/
	chown -R mongo:mongo ${mongodb_path}/bin
	/etc/init.d/mongodb start
	if [ ${CompatibilityVersion} != "${mongodb_version:0:3}" ]; then
		mongo --eval 'db.adminCommand( { setFeatureCompatibilityVersion: "4.0" } )'
	fi
	echo "${mongodb_version}" > ${mongodb_path}/version.pl
	echo "${mongodb_version}" > ${mongodb_path}/version_check.pl
	echo "${mongodb_version}" > /www/server/panel/plugin/mongodb/version.pl
	echo "${mongodb_version}" > /www/server/panel/plugin/mongodb/version_check.pl
}
Bt_Check(){
	checkFile="/www/server/panel/install/check.sh"
	wget -O ${checkFile} ${download_Url}/tools/check.sh
	. ${checkFile}
}
action=$1
version=$2
vphp=${version:0:1}${version:1:1}

if [ "$vphp" -ge "70" ];then
	wget -O php_mongodb.sh ${download_Url}/install/0/php_mongodb.sh
	bash php_mongodb.sh $1 $2
	exit;
fi


if [ "${1}" == 'install' ];then
	Install_mongodb
	Service_Add
	Bt_Check
elif [ "${1}" == 'update' ]; then
	Update_mongodb
else
	Service_Del
	Uninstall_mongodb
fi

