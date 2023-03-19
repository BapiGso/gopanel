#!/bin/bash
PATH=/bin:/sbin:/usr/bin:/usr/sbin:/usr/local/bin:/usr/local/sbin:~/bin
export PATH

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

run_path="/root"
mysql_dir="$Root_Path/server/mysql"
mysql_config="${mysql_dir}/bin/mysql_config"
php_path="$Root_Path/server/php"
Is_64bit=$(getconf LONG_BIT)
Root_Path=$(cat /var/bt_setupPath.conf)
apacheVersion=$(cat /var/bt_apacheVersion.pl)
sysType=$(uname -a|grep x86_64)

CENTOS_OS=$(cat /etc/redhat-release|grep  -oEi centos)
if [ "${CENTOS_OS}" ];then
	el=$(cat /etc/redhat-release|grep -oE "([6-8]\.)+[0-9]+"|cut -f1 -d ".")
	if [ -z "${el}" ];then
		el=$(cat /etc/redhat-release|grep -i "Centos Stream"|grep -oE 8)
	fi

fi

ALIYUN_OS=$(cat /etc/redhat-release|grep  -oEi aliyun)
if [ "${ALIYUN_OS}" ];then
	el=$(uname -r|grep -oE "al7|al8"|grep -oE "[7-8]")
fi

if [ "${el}" == "8" ];then
	yum config-manager --set-enabled PowerTools
    yum config-manager --set-enabled powertools
fi

Install_Libzip(){
	if [ "${el}" == "8" ];then
		yum install -y libzip-devel
	else
		mkdir libzip
		cd libzip
		wget -O libzip5-1.5.2.rpm ${download_Url}/rpm/remi/${el}/libzip5-1.5.2.rpm
		wget -O libzip5-devel-1.5.2.rpm ${download_Url}/rpm/remi/${el}/libzip5-devel-1.5.2.rpm
		wget -O libzip5-tools-1.5.2.rpm ${download_Url}/rpm/remi/${el}/libzip5-tools-1.5.2.rpm
		yum install * -y
		cd ..
		rm -rf libzip
	fi
}

if [ -z "${el}" ] || [ "${Is_64bit}" == "32" ] || [ -z "${sysType}" ];then
	wget -O php.sh $download_Url/install/0/php.sh -T 5
	bash php.sh $1 $2
	exit;
fi

LibCurlVer=$(/usr/local/curl/bin/curl -V|grep curl|awk '{print $2}'|cut -d. -f2)
if [[ "${LibCurlVer}" -lt "64" ]] || [ "${el}" == "6" ] ; then
	wget -O php.sh $download_Url/install/1/old/php.sh -T 5
	bash php.sh $1 $2
	exit;
fi


Set_Php_INI(){
	sed -i 's/disable_functions =.*/disable_functions = passthru,exec,system,putenv,chroot,chgrp,chown,shell_exec,popen,proc_open,pcntl_exec,ini_alter,ini_restore,dl,openlog,syslog,readlink,symlink,popepassthru,pcntl_alarm,pcntl_fork,pcntl_waitpid,pcntl_wait,pcntl_wifexited,pcntl_wifstopped,pcntl_wifsignaled,pcntl_wifcontinued,pcntl_wexitstatus,pcntl_wtermsig,pcntl_wstopsig,pcntl_signal,pcntl_signal_dispatch,pcntl_get_last_error,pcntl_strerror,pcntl_sigprocmask,pcntl_sigwaitinfo,pcntl_sigtimedwait,pcntl_exec,pcntl_getpriority,pcntl_setpriority,imap_open,apache_setenv/g' /www/server/php/${php_version}/etc/php.ini
	sed -i 's/expose_php = On/expose_php = Off/g' /www/server/php/${php_version}/etc/php.ini
}
Set_Php_FPM(){
	sed -i "s#listen.backlog.*#listen.backlog = 8192#" /www/server/php/${php_version}/etc/php-fpm.conf
}
Install_PHP(){
	if [ "${php_version}" -ge "73" ]; then
		Install_Libzip
		yum install libsodium-devel sqlite-devel oniguruma-devel -y
	fi
	wget ${download_Url}/rpm/centos${el}/${Is_64bit}/bt-php${php_version}.rpm
	rpm -ivh bt-php${php_version}.rpm --force --nodeps
	rm -f bt-php${php_version}.rpm
}
Bt_Check(){
	checkFile="/www/server/panel/install/check.sh"
	wget -O ${checkFile} ${download_Url}/tools/check.sh
	. ${checkFile}
}
Uninstall_PHP(){
	php_version=${1/./}
	if [ -f "/www/server/php/${php_version}/rpm.pl" ];then
		yum remove -y bt-php${php_version}
		[ ! -f "/www/server/php/${php_version}/bin/php" ] && exit 0;
	fi
	service php-fpm-$php_version stop

	chkconfig --del php-fpm-${php_version}
	chkconfig --level 2345 php-fpm-${php_version} off

	rm -rf $php_path/$php_version
	rm -f /etc/init.d/php-fpm-$php_version

}
Download_Conf(){
	if [ ! -f "/www/server/nginx/conf/enable-php-${php_version}.conf" ];then
		wget -O /www/server/nginx/conf/enable-php-${php_version}.conf ${download_Url}/conf/enable-php-${php_version}.conf
	fi
}

Install_Zip_ext(){
	php_setup_path="/www/server/php/${php_version}"
	mkdir -p ${php_setup_path}/src/ext
	cd ${php_setup_path}/src/ext
	wget -O zip${php_version}.tar.gz ${download_Url}/rpm/src/zip${php_version}.tar.gz
	tar -xvf zip${php_version}.tar.gz
	rm -f zip${php_version}.tar.gz
	cd zip
	${php_setup_path}/bin/phpize
	./configure --with-php-config=${php_setup_path}/bin/php-config
	make && make install
	cd ../../

	if [ "${php_version}" == "73" ];then
		extFile="/www/server/php/73/lib/php/extensions/no-debug-non-zts-20180731/zip.so"
	elif [ "${php_version}" == "74" ]; then
		extFile="/www/server/php/74/lib/php/extensions/no-debug-non-zts-20190902/zip.so"
	elif [ "${php_version}" == "80" ]; then
		extFile="/www/server/php/80/lib/php/extensions/no-debug-non-zts-20200930/zip.so"
	fi

	if [ -f "${extFile}" ];then
		echo "extension = zip.so" >> ${php_setup_path}/etc/php.ini
	fi
}


actionType=$1
version=$2

if [ "$actionType" == 'install' ];then
	php_version=`echo $version|sed "s/\.//"`
	rm -f /tmp/php-cgi-${php_version}.sock
	Install_PHP
	Set_Php_INI
	Set_Php_FPM
	Download_Conf
	if [ "${php_version}" -ge "73" ];then
		Install_Zip_ext
	fi
	/etc/init.d/php-fpm-${php_version} reload
	rm -f /tmp/php-cgi-${php_version}.sock
	/etc/init.d/php-fpm-${php_version} start
	Bt_Check
else
	if [ "$actionType" == 'uninstall' ];then
	Uninstall_PHP $version
	fi
fi

