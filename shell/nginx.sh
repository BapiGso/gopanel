#!/bin/bash
PATH=/bin:/sbin:/usr/bin:/usr/sbin:/usr/local/bin:/usr/local/sbin:~/bin
export PATH
LANG=en_US.UTF-8

public_file=/www/server/panel/install/public.sh
publicFileMd5=$(md5sum ${public_file} 2>/dev/null|awk '{print $1}')
md5check="977453aae3465f41a9f6039ee8c84c10"
if [ "${publicFileMd5}" != "${md5check}"  ]; then
	wget -O Tpublic.sh http://download.bt.cn/install/public.sh -T 20;
	publicFileMd5=$(md5sum Tpublic.sh 2>/dev/null|awk '{print $1}')
	if [ "${publicFileMd5}" == "${md5check}"  ]; then
		\cp -rpa Tpublic.sh $public_file
	fi
	rm -f Tpublic.sh
fi
. $public_file
download_Url=$NODE_URL

Is_64bit=$(getconf LONG_BIT)
Centos6Check=$(cat /etc/redhat-release|grep ' 6.'|grep -i centos)
Centos7Check=$(cat /etc/redhat-release|grep ' 7.'|grep -i centos)
Centos8Check=$(cat /etc/redhat-release|grep ' 8.'|grep -i centos)
sysType=$(uname -a|grep x86_64)
Setup_Path="/www/server/nginx"

if [ "${Centos7Check}" ]; then
	rpm_path="centos7"
elif [ "${Centos8Check}" ]; then
	rpm_path="centos8"
fi

if [ -z "${rpm_path}" ] || [ "${Is_64bit}" = "32" ] || [ -z "${sysType}" ] || [ "${2}" == "1.18.gmssl" ]; then
	wget -O nginx.sh ${download_Url}/install/0/nginx.sh && sh nginx.sh $1 $2
	exit;
fi

Install_Jemalloc(){
	if [ ! -f '/usr/local/lib/libjemalloc.so' ]; then
		wget -O jemalloc-5.0.1.tar.bz2 ${download_Url}/src/jemalloc-5.0.1.tar.bz2
		tar -xvf jemalloc-5.0.1.tar.bz2
		cd jemalloc-5.0.1
		./configure
		make && make install
		ldconfig
		cd ..
		rm -rf jemalloc*
	fi
}
Install_cjson()
{
	if [ ! -f /usr/local/lib/lua/5.1/cjson.so ];then
		wget -O lua-cjson-2.1.0.tar.gz $download_Url/install/src/lua-cjson-2.1.0.tar.gz -T 20
		tar xvf lua-cjson-2.1.0.tar.gz
		rm -f lua-cjson-2.1.0.tar.gz
		cd lua-cjson-2.1.0
		make
		make install
		cd ..
		rm -rf lua-cjson-2.1.0
	fi
}
Install_LuaJIT()
{
	if [ ! -d '/usr/local/include/luajit-2.0' ];then
		yum install libtermcap-devel ncurses-devel libevent-devel readline-devel -y
		wget -c -O LuaJIT-2.0.4.tar.gz ${download_Url}/install/src/LuaJIT-2.0.4.tar.gz -T 5
		tar xvf LuaJIT-2.0.4.tar.gz
		cd LuaJIT-2.0.4
		make linux
		make install
		cd ..
		rm -rf LuaJIT-*
		export LUAJIT_LIB=/usr/local/lib
		export LUAJIT_INC=/usr/local/include/luajit-2.0/
		ln -sf /usr/local/lib/libluajit-5.1.so.2 /usr/local/lib64/libluajit-5.1.so.2
		echo "/usr/local/lib" >> /etc/ld.so.conf
		ldconfig
	fi
}
Install_Nginx(){
	Uninstall_Nginx
	if [ -f "/www/server/panel/vhost/nginx/btwaf.conf" ]; then
		if [ ! -f "/www/server/btwaf/waf.lua" ]; then
			rm -f /www/server/panel/vhost/nginx/btwaf.conf
		fi
	fi
	wget ${download_Url}/rpm/${rpm_path}/${Is_64bit}/bt-${nginxVersion}.rpm
	rpm -ivh bt-${nginxVersion}.rpm --force --nodeps
	rm -f bt-${nginxVersion}.rpm
	echo ${nginxVersion} > ${Setup_Path}/rpm.pl
	if [ "${version}" == "tengine" ]; then
		echo "-Tengine2.2.3" > ${Setup_Path}/version.pl
	elif [ "${version}" == "openresty" ]; then
		echo "openresty" > ${Setup_Path}/version.pl
	else
		echo "${ngxVer}" > ${Setup_Path}/version.pl
	fi
	wget -O /www/server/nginx/conf/enable-php-80.conf ${download_Url}/conf/enable-php-80.conf
	AA_PANEL_CHECK=$(cat /www/server/panel/config/config.json|grep "English")
	if [ "${AA_PANEL_CHECK}" ];then
		\cp -rf /www/server/panel/data/empty.html /www/server/nginx/html/index.html
		chmod 644 /www/server/nginx/html/index.html
		wget -O /www/server/panel/vhost/nginx/0.default.conf ${download_Url}/conf/nginx/en.0.default.conf
		/etc/init.d/nginx reload
	fi
}
Bt_Check(){
	checkFile="/www/server/panel/install/check.sh"
	wget -O ${checkFile} ${download_Url}/tools/check.sh
	. ${checkFile}
}
Uninstall_Nginx(){
	/etc/init.d/nginx stop
	chkconfig --del nginx
	chkconfig --level 2345 nginx off
	if [ -f "${Setup_Path}/rpm.pl" ]; then
		yum remove bt-$(cat ${Setup_Path}/rpm.pl) -y
	fi
	rm -f /etc/init.d/nginx
	rm -rf /www/server/nginx

}

actionType=$1
version=$2

if [ "$actionType" == 'install' ];then
	nginxVersion="tengine"
	if [ "${version}" == "1.10" ] || [ "${version}" == "1.12" ]; then
		nginxVersion="nginx112"
		ngxVer="1.12.2"
	elif [ "${version}" == "1.14" ]; then
		nginxVersion="nginx114"
		ngxVer="1.14.2"
	elif [ "${version}" == "1.15" ]; then
		nginxVersion="nginx115"
		ngxVer="1.15.10"
	elif [ "${version}" == "1.16" ]; then
		nginxVersion="nginx116"
		ngxVer="1.16.0"
	elif [ "${version}" == "1.17" ]; then
		nginxVersion="nginx117"
		ngxVer="1.17.0"
	elif [ "${version}" == "1.18" ]; then
		nginxVersion="nginx118"
		ngxVer="1.18.0"
	elif [ "${version}" == "1.19" ]; then
		nginxVersion="nginx119"
		ngxVer="1.19.0"
	elif [ "${version}" == "1.20" ]; then
		nginxVersion="nginx120"
		ngxVer="1.20.0"
	elif [ "${version}" == "1.8" ]; then
		nginxVersion="nginx108"
		ngxVer="1.8.1"
	elif [ "${version}" == "openresty" ]; then
		nginxVersion="openresty"
	else
		version="tengine"
	fi
	Install_Jemalloc
	Install_Jemalloc
	Install_LuaJIT
	Install_LuaJIT
	Install_cjson
	Install_Nginx
	Bt_Check
else
	if [ "$actionType" == 'uninstall' ];then
	Uninstall_Nginx
	fi
fi

