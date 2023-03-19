#!/bin/sh
public_file=/www/server/panel/install/public.sh
if [ ! -f $public_file ];then
	wget -O $public_file http://download.bt.cn/install/public.sh -T 5;
fi
. $public_file
download_Url=$NODE_URL
OSNAMEVER=
OSNAME=
OSVER=
MARIADBCPUARCH=
##OLS系统版本方法
function check_os
{
    if [ -f /etc/redhat-release ] ; then
        cat /etc/redhat-release | grep " 6." >/dev/null
        if [ $? = 0 ] ; then
            OSNAMEVER=CENTOS6
            OSNAME=centos
            OSVER=6
        else
            cat /etc/redhat-release | grep " 7." >/dev/null
            if [ $? = 0 ] ; then
                OSNAMEVER=CENTOS7
                OSNAME=centos
                OSVER=7

            else
                cat /etc/redhat-release | grep " 8." >/dev/null
                if [ $? = 0 ] ; then
                    OSNAMEVER=CENTOS8
                    OSNAME=centos
                    OSVER=8
                fi
            fi
        fi
    elif [ -f /etc/lsb-release ] ; then
        cat /etc/lsb-release | grep "DISTRIB_RELEASE=14." >/dev/null
        if [ $? = 0 ] ; then
            OSNAMEVER=UBUNTU14
            OSNAME=ubuntu
            OSVER=trusty
            MARIADBCPUARCH="arch=amd64,i386,ppc64el"
        else
            cat /etc/lsb-release | grep "DISTRIB_RELEASE=16." >/dev/null
            if [ $? = 0 ] ; then
                OSNAMEVER=UBUNTU16
                OSNAME=ubuntu
                OSVER=xenial
                MARIADBCPUARCH="arch=amd64,i386,ppc64el"

            else
                cat /etc/lsb-release | grep "DISTRIB_RELEASE=18." >/dev/null
                if [ $? = 0 ] ; then
                    OSNAMEVER=UBUNTU18
                    OSNAME=ubuntu
                    OSVER=bionic
                    MARIADBCPUARCH="arch=amd64"

                else
                    cat /etc/lsb-release | grep "DISTRIB_RELEASE=20." >/dev/null
                    if [ $? = 0 ] ; then
                        OSNAMEVER=UBUNTU20
                        OSNAME=ubuntu
                        OSVER=focal
                        MARIADBCPUARCH="arch=amd64"
                    fi
                fi
            fi
        fi
    elif [ -f /etc/debian_version ] ; then
        cat /etc/debian_version | grep "^7." >/dev/null
        if [ $? = 0 ] ; then
            OSNAMEVER=DEBIAN7
            OSNAME=debian
            OSVER=wheezy
            MARIADBCPUARCH="arch=amd64,i386"
        else
            cat /etc/debian_version | grep "^8." >/dev/null
            if [ $? = 0 ] ; then
                OSNAMEVER=DEBIAN8
                OSNAME=debian
                OSVER=jessie
                MARIADBCPUARCH="arch=amd64,i386"
            else
                cat /etc/debian_version | grep "^9." >/dev/null
                if [ $? = 0 ] ; then
                    OSNAMEVER=DEBIAN9
                    OSNAME=debian
                    OSVER=stretch
                    MARIADBCPUARCH="arch=amd64,i386"
                else
                    cat /etc/debian_version | grep "^10." >/dev/null
                    if [ $? = 0 ] ; then
                        OSNAMEVER=DEBIAN10
                        OSNAME=debian
                        OSVER=buster
                        MARIADBCPUARCH="arch=amd64,i386"
                    fi
                fi
            fi
        fi
    fi

    if [ "x$OSNAMEVER" = "x" ] ; then
        echo "Sorry, currently one click installation only supports Centos(6-8), Debian(7-10) and Ubuntu(14,16,18,20)."
        echo "You can download the source code and build from it."
        echo "The url of the source code is https://github.com/litespeedtech/openlitespeed/releases."
        echo
        exit 1
    else
        if [ "x$OSNAME" = "xcentos" ] ; then
            echo "Current platform is "  "$OSNAME $OSVER."
        else
            export DEBIAN_FRONTEND=noninteractive
            echo "Current platform is "  "$OSNAMEVER $OSNAME $OSVER."
        fi
    fi
}

Install_OLS(){
  #判断中文版面板版本
  grep "English" /www/server/panel/config/config.json
  if [ "$?" -ne 0 ];then
    panel_version=$(grep -E "^\s*g\.version" /www/server/panel/class/common.py |awk -F "'" '{print $2}'|sed 's/\.//g')
    echo $panel_version
    if [ "$panel_version" -lt 7412 ];then
      echo "抱歉，目前仅支持面板版本为7.4.12或以上的测试版"
      exit
    fi
  fi
  check_os
  if [ -f '/etc/redhat-release' ];then
    rpm -Uvh $download_Url/src/litespeed-repo-1.1-1.el$OSVER.noarch.rpm
    ols_check_repo=$(rpm -qa|grep litespeed)
    if [ $ols_check_repo = '' ];then
      rpm -Uvh http://rpms.litespeedtech.com/centos/litespeed-repo-1.1-1.el$OSVER.noarch.rpm
    fi
  else
    grep -Fq  "http://rpms.litespeedtech.com/debian/" /etc/apt/sources.list.d/lst_debian_repo.list
    if [ $? != 0 ] ; then
        echo "deb http://rpms.litespeedtech.com/debian/ $OSVER main"  > /etc/apt/sources.list.d/lst_debian_repo.list
    fi
    wget -O /etc/apt/trusted.gpg.d/lst_debian_repo.gpg http://rpms.litespeedtech.com/debian/lst_debian_repo.gpg
    wget -O /etc/apt/trusted.gpg.d/lst_repo.gpg http://rpms.litespeedtech.com/debian/lst_repo.gpg
    apt-get -y update
  fi
  init_OLS
  process_OLS_conf
}

update_OLS(){
  \cp /usr/local/lsws/conf/httpd_config.conf /usr/local/lsws/conf/httpd_config.conf_bt
  init_OLS
  \cp /usr/local/lsws/conf/httpd_config.conf_bt /usr/local/lsws/conf/httpd_config.conf
  sed -i "s/user\s*nobody\s*/user www/g" $ols_conf
  sed -i "s/group\s*nobody\s*/group www/g" $ols_conf
  chmod 755 /usr/local/lsws/conf/httpd_config.conf
  sleep 3
  /usr/local/lsws/bin/lswsctrl restart
}

init_OLS() {
  cd /tmp
  grep "English" /www/server/panel/config/config.json
  if [ "$?" -ne 0 ];then
    host=$(cat /etc/hosts|grep -E '103.224.251.67\s*rpms.litespeedtech.com')
    if [ "$host" == "" ];then
      echo '103.224.251.67 rpms.litespeedtech.com www.litespeedtech.com'>> /etc/hosts
    fi
    wget -O openlitespeed-${ols_version}.tgz --no-check-certificate $download_Url/src/openlitespeed-${ols_version}.tgz
  else
    wget -O openlitespeed-${ols_version}.tgz --no-check-certificate https://openlitespeed.org/packages/openlitespeed-${ols_version}.tgz
  fi
  tar -zxvf openlitespeed-${ols_version}.tgz
  chown -R root.root /tmp/openlitespeed
  chmod -R 777 /tmp/openlitespeed
  cd /tmp/openlitespeed
  bash install.sh
}

process_OLS_conf () {
  #ols安装脚本
#  wget --no-check-certificate https://raw.githubusercontent.com/litespeedtech/ols1clk/master/ols1clk.sh && bash ols1clk.sh --quiet

  #修改主配置文件
  ols_conf='/usr/local/lsws/conf/httpd_config.conf'
  if [ -f $ols_conf ];then
    sed -i "s/address\s*\*:80/address *:81/g" $ols_conf
    sed -i "s/address\s*\*:443/address *:4433/g" $ols_conf
    sed -i "s/user\s*nobody\s*/user www/g" $ols_conf
    sed -i "s/group\s*nobody\s*/group www/g" $ols_conf
  fi
  echo "include /www/server/panel/vhost/openlitespeed/*.conf" >> $ols_conf
  echo "include /www/server/panel/vhost/openlitespeed/listen/*.conf" >> $ols_conf
#  设置默认安装的php版本
  if [ -f '/etc/redhat-release' ];then
    rm -f /usr/local/lsws/lsphp74/etc/php.ini_old
    mv /usr/local/lsws/lsphp74/etc/php.ini /usr/local/lsws/lsphp74/etc/php.ini_old
    wget -O /usr/local/lsws/lsphp74/etc/php.ini $download_Url/install/ols/php/centos/php74.ini
    if [ ! -f /usr/local/lsws/lsphp74/etc/php.ini ];then
      mv /usr/local/lsws/lsphp74/etc/php.ini_old /usr/local/lsws/lsphp74/etc/php.ini
      sed -i 's/disable_functions =.*/disable_functions = passthru,exec,system,putenv,chroot,chgrp,chown,shell_exec,popen,proc_open,pcntl_exec,ini_alter,ini_restore,dl,openlog,syslog,readlink,symlink,popepassthru,pcntl_alarm,pcntl_fork,pcntl_waitpid,pcntl_wait,pcntl_wifexited,pcntl_wifstopped,pcntl_wifsignaled,pcntl_wifcontinued,pcntl_wexitstatus,pcntl_wtermsig,pcntl_wstopsig,pcntl_signal,pcntl_signal_dispatch,pcntl_get_last_error,pcntl_strerror,pcntl_sigprocmask,pcntl_sigwaitinfo,pcntl_sigtimedwait,pcntl_exec,pcntl_getpriority,pcntl_setpriority,imap_open,apache_setenv/g' /usr/local/lsws/lsphp74/etc/php.ini
      sed -i 's/mysqli.default_socket.*/mysqli.default_socket = \/tmp\/mysql.sock/g' /usr/local/lsws/lsphp74/etc/php.ini
      sed -i 's/mysql.default_socket.*/mysql.default_socket = \/tmp\/mysql.sock/g' /usr/local/lsws/lsphp74/etc/php.ini
      sed -i 's/session.save_path.*/session.save_path = "\/tmp"/g' /usr/local/lsws/lsphp74/etc/php.ini
    fi
  else
    rm -f /usr/local/lsws/lsphp74/etc/php/7.4/litespeed/php.ini_old
    mv /usr/local/lsws/lsphp74/etc/php/7.4/litespeed/php.ini /usr/local/lsws/lsphp74/etc/php/7.4/litespeed/php.ini_old
    wget -O /usr/local/lsws/lsphp74/etc/php/7.4/litespeed/php.ini $download_Url/install/ols/php/ubuntu/php74.ini
    if [ ! -f "/usr/local/lsws/lsphp74/etc/php/7.4/litespeed/php.ini" ];then
      mv /usr/local/lsws/lsphp74/etc/php/7.4/litespeed/php.ini_old /usr/local/lsws/lsphp74/etc/php/7.4/litespeed/php.ini
      sed -i 's/disable_functions =.*/disable_functions = passthru,exec,system,putenv,chroot,chgrp,chown,shell_exec,popen,proc_open,pcntl_exec,ini_alter,ini_restore,dl,openlog,syslog,readlink,symlink,popepassthru,pcntl_alarm,pcntl_fork,pcntl_waitpid,pcntl_wait,pcntl_wifexited,pcntl_wifstopped,pcntl_wifsignaled,pcntl_wifcontinued,pcntl_wexitstatus,pcntl_wtermsig,pcntl_wstopsig,pcntl_signal,pcntl_signal_dispatch,pcntl_get_last_error,pcntl_strerror,pcntl_sigprocmask,pcntl_sigwaitinfo,pcntl_sigtimedwait,pcntl_exec,pcntl_getpriority,pcntl_setpriority,imap_open,apache_setenv/g' /usr/local/lsws/lsphp74/etc/php/7.4/litespeed/php.ini
      sed -i 's/mysqli.default_socket.*/mysqli.default_socket = \/tmp\/mysql.sock/g' /usr/local/lsws/lsphp74/etc/php/7.4/litespeed/php.ini
      sed -i 's/mysql.default_socket.*/mysql.default_socket = \/tmp\/mysql.sock/g' /usr/local/lsws/lsphp74/etc/php/7.4/litespeed/php.ini
      sed -i 's/session.save_path.*/session.save_path = "\/tmp"/g' /usr/local/lsws/lsphp74/etc/php/7.4/litespeed/php.ini
    fi
  fi
  #检查已经安装了什么版本的php并安装对应的lsphp
  php_dir='/www/server/php/'
  phpv=''
  if [ -d $phpdir ];then
          phpv=`ls $php_dir`
  fi
  echo $php_v
  array=''
  if [ '$php_v' != '' ];then
          array=(${phpv// /})
  fi
  for v in ${array[@]}
    do
    #centos安装
    if [ -f /etc/redhat-release ];then
      centos_install_ols
      if [ ! -f /usr/local/lsws/lsphp${v}/etc/php.ini ];then
			  centos_install_ols
		  fi
		  if [ ! -f /usr/local/lsws/lsphp${v}/bin/php ];then
			  centos_install_ols
		  fi
    #debian安装
    else
      debian_install_ols
      if [ ! -f /usr/local/lsws/lsphp$v/etc/php/${v:0:1}.${v:1:2}/litespeed/php.ini ];then
			  debian_install_ols
		  fi
		  if [ ! -f /usr/local/lsws/lsphp${v}/bin/php ];then
			  debian_install_ols
		  fi
    fi
    if [ -f '/etc/redhat-release' ];then
      rm -f /usr/local/lsws/lsphp$v/etc/php.ini_old
      mv /usr/local/lsws/lsphp$v/etc/php.ini /usr/local/lsws/lsphp$v/etc/php.ini_old
      wget -O /usr/local/lsws/lsphp$v/etc/php.ini $download_Url/install/ols/php/centos/php$v.ini
      if [ ! -f /usr/local/lsws/lsphp$v/etc/php.ini ];then
        mv /usr/local/lsws/lsphp$v/etc/php.ini_old /usr/local/lsws/lsphp$v/etc/php.ini
        sed -i 's/disable_functions =.*/disable_functions = passthru,exec,system,putenv,chroot,chgrp,chown,shell_exec,popen,proc_open,pcntl_exec,ini_alter,ini_restore,dl,openlog,syslog,readlink,symlink,popepassthru,pcntl_alarm,pcntl_fork,pcntl_waitpid,pcntl_wait,pcntl_wifexited,pcntl_wifstopped,pcntl_wifsignaled,pcntl_wifcontinued,pcntl_wexitstatus,pcntl_wtermsig,pcntl_wstopsig,pcntl_signal,pcntl_signal_dispatch,pcntl_get_last_error,pcntl_strerror,pcntl_sigprocmask,pcntl_sigwaitinfo,pcntl_sigtimedwait,pcntl_exec,pcntl_getpriority,pcntl_setpriority,imap_open,apache_setenv/g' /usr/local/lsws/lsphp$v/etc/php.ini
        sed -i 's/mysqli.default_socket.*/mysqli.default_socket = \/tmp\/mysql.sock/g' /usr/local/lsws/lsphp$v/etc/php.ini
        sed -i 's/mysql.default_socket.*/mysql.default_socket = \/tmp\/mysql.sock/g' /usr/local/lsws/lsphp$v/etc/php.ini
        sed -i 's/session.save_path.*/session.save_path = "\/tmp"/g' /usr/local/lsws/lsphp$v/etc/php.ini
      fi
    else
      rm -f /usr/local/lsws/lsphp$v/etc/php/${v:0:1}.${v:1:2}/litespeed/php.ini_old
      mv /usr/local/lsws/lsphp$v/etc/php/${v:0:1}.${v:1:2}/litespeed/php.ini /usr/local/lsws/lsphp$v/etc/php/${v:0:1}.${v:1:2}/litespeed/php.ini_old
      wget -O /usr/local/lsws/lsphp$v/etc/php/${v:0:1}.${v:1:2}/litespeed/php.ini $download_Url/install/ols/php/ubuntu/php$v.ini
      if [ ! -f "/usr/local/lsws/lsphp$v/etc/php/${v:0:1}.${v:1:2}/litespeed/php.ini" ];then
		    mv /usr/local/lsws/lsphp$v/etc/php/${v:0:1}.${v:1:2}/litespeed/php.ini_old /usr/local/lsws/lsphp$v/etc/php/${v:0:1}.${v:1:2}/litespeed/php.ini
        sed -i 's/disable_functions =.*/disable_functions = passthru,exec,system,putenv,chroot,chgrp,chown,shell_exec,popen,proc_open,pcntl_exec,ini_alter,ini_restore,dl,openlog,syslog,readlink,symlink,popepassthru,pcntl_alarm,pcntl_fork,pcntl_waitpid,pcntl_wait,pcntl_wifexited,pcntl_wifstopped,pcntl_wifsignaled,pcntl_wifcontinued,pcntl_wexitstatus,pcntl_wtermsig,pcntl_wstopsig,pcntl_signal,pcntl_signal_dispatch,pcntl_get_last_error,pcntl_strerror,pcntl_sigprocmask,pcntl_sigwaitinfo,pcntl_sigtimedwait,pcntl_exec,pcntl_getpriority,pcntl_setpriority,imap_open,apache_setenv/g' /usr/local/lsws/lsphp$v/etc/php/${v:0:1}.${v:1:2}/litespeed/php.ini
        sed -i 's/mysqli.default_socket.*/mysqli.default_socket = \/tmp\/mysql.sock/g' /usr/local/lsws/lsphp$v/etc/php/${v:0:1}.${v:1:2}/litespeed/php.ini
        sed -i 's/mysql.default_socket.*/mysql.default_socket = \/tmp\/mysql.sock/g' /usr/local/lsws/lsphp$v/etc/php/${v:0:1}.${v:1:2}/litespeed/php.ini
        sed -i 's/session.save_path.*/session.save_path = "\/tmp"/g' /usr/local/lsws/lsphp$v/etc/php/${v:0:1}.${v:1:2}/litespeed/php.ini
	    fi
    fi
  done
  if [ ! -f "/www/server/panel/vhost/openlitespeed/listen/80.conf" ];then
    mkdir -p /www/server/panel/vhost/openlitespeed/listen
    mkdir -p /www/server/panel/vhost/openlitespeed/detail
    if [ ! -f "/www/server/panel/vhost/openlitespeed/default.conf" ];then
      write_default
    fi
    write_80
  else
    write_default
    grep "map default" /www/server/panel/vhost/openlitespeed/listen/80.conf
    if [ "$?" != "0" ];then
      sed -i 's/secure\s*0/secure 0\n    map default \*/' /www/server/panel/vhost/openlitespeed/listen/80.conf
    fi
  fi
  if [ ! -f "/www/server/panel/vhost/openlitespeed/listen/888.conf" ];then
    write_phpmyadmin
  fi
  write_phpmyadmin_ssl
  ##处理mysqlsock位置可能无法修改的问题
  rm -f /var/run/mysqld/mysqld.sock
  mkdir -p /var/run/mysqld/
  ln -s /tmp/mysql.sock /var/run/mysqld/mysqld.sock
  systemctl enable lsws
  /usr/local/lsws/bin/lswsctrl restart
}

centos_install_ols(){
      JSON=
      if [ "$v" = "70" ] || [ "$v" = "71" ] || [ "$v" = "72" ] || [ "$v" = "73" ] || [ "$v" = "74" ] ; then
        JSON=lsphp$v-json
      fi
      for i in lsphp$v lsphp$v-common lsphp$v-gd lsphp$v-process lsphp$v-mbstring lsphp$v-xml lsphp$v-mcrypt lsphp$v-pdo lsphp$v-imap lsphp$v-mysqlnd lsphp$v-sqlite3 lsphp$v-zip lsphp$v-curl $JSON
      do
      yum -y install $i
      done
}

debian_install_ols(){
      for i in lsphp$v lsphp$v-mysql lsphp$v-imap lsphp$v-curl lsphp$v-sqlite3 lsphp$v-zip
      do
      apt-get -y install $i
      done
      if [ "$v" != "70" ] && [ "$v" != "71" ] && [ "$v" != "72" ]  && [ "$v" != "73" ] && [ "$v" != "74" ]; then
        apt-get -y install lsphp$v-gd lsphp$v-mcrypt
      else
         apt-get -y install lsphp$v-common lsphp$v-json
      fi
}

check_panel_version () {
  tmp=`cat /www/server/panel/class/common.py|grep "g.version = "|awk -F "'" '{print $2}'`
  version=`echo ${tmp//./}`
  if [ $version -lt 666 ];then
        echo "Your panel version is less than 6.6.6, this environment is not supported, please update the panel before installing"
        exit
  fi
}

write_default () {
  touch /www/server/panel/vhost/openlitespeed/default.conf
  cat > /www/server/panel/vhost/openlitespeed/default.conf <<- EOF
#VHOST default START
virtualhost default {
vhRoot /www/server/panel/data/
configFile /www/server/panel/vhost/openlitespeed/detail/default.conf
allowSymbolLink 1
enableScript 1
restrained 1
setUIDMode 0
}
#VHOST default END
EOF
  touch /www/server/panel/vhost/openlitespeed/detail/default.conf
  cat > /www/server/panel/vhost/openlitespeed/detail/default.conf <<- EOF
docRoot                   \$VH_ROOT
vhDomain                  \$VH_NAME
adminEmails               example@example.com
enableGzip                1
enableIpGeo               1

index  {
  useServer               0
  indexFiles empty.html
}

errorlog /www/wwwlogs/\$VH_NAME_ols.error_log {
  useServer               0
  logLevel                ERROR
  rollingSize             10M
}

accesslog /www/wwwlogs/\$VH_NAME_ols.access_log {
  useServer               0
  logFormat               "%v %h %l %u %t "%r" %>s %b"
  logHeaders              5
  rollingSize             10M
  keepDays                10  compressArchive         1
}


phpIniOverride  {
php_admin_value open_basedir "/tmp:$VH_ROOT"
}
EOF
}

write_80 () {
  touch /www/server/panel/vhost/openlitespeed/listen/80.conf
  cat > /www/server/panel/vhost/openlitespeed/listen/80.conf <<- EOF
listener Default80{
    address *:80
    secure 0
    map default *
}
EOF
}

write_phpmyadmin () {
  last_v=`ls /www/server/php/|awk 'END {print}'`
  if [ "$last_v" == '' ];then
    last_v='73'
  fi
  touch /www/server/panel/vhost/openlitespeed/phpmyadmin.conf
  cat > /www/server/panel/vhost/openlitespeed/phpmyadmin.conf <<- EOF
#VHOST phpmyadmin START
virtualhost phpmyadmin {
vhRoot /www/server/phpmyadmin/
configFile /www/server/panel/vhost/openlitespeed/detail/phpmyadmin.conf
allowSymbolLink 1
enableScript 1
restrained 1
setUIDMode 0
}
#VHOST phpmyadmin END
EOF
  touch /www/server/panel/vhost/openlitespeed/detail/phpmyadmin.conf
  cat > /www/server/panel/vhost/openlitespeed/detail/phpmyadmin.conf <<- EOF
docRoot                   \$VH_ROOT
vhDomain                  \$VH_NAME
adminEmails               example@example.com
enableGzip                1
enableIpGeo               1

index  {
  useServer               0
  indexFiles index.php,index.html
}

errorlog /www/wwwlogs/\$VH_NAME_ols.error_log {
  useServer               0
  logLevel                ERROR
  rollingSize             10M
}

accesslog /www/wwwlogs/\$VH_NAME_ols.access_log {
  useServer               0
  logFormat               "%v %h %l %u %t "%r" %>s %b"
  logHeaders              5
  rollingSize             10M
  keepDays                10  compressArchive         1
}

scripthandler  {
  add                     lsapi:phpmyadmin php
}

extprocessor phpmyadmin {
  type                    lsapi
  address                 UDS://tmp/lshttpd/phpmyadmin.sock
  maxConns                10
  env                     LSAPI_CHILDREN=10
  initTimeout             600
  retryTimeout            0
  persistConn             1
  pcKeepAliveTimeout      1
  respBuffer              0
  autoStart               1
  path                    /usr/local/lsws/lsphp${last_v}/bin/lsphp
  extUser                 www
  extGroup                www
  memSoftLimit            2047M
  memHardLimit            2047M
  procSoftLimit           400
  procHardLimit           500
}

phpIniOverride  {
php_admin_value open_basedir "/tmp:\$VH_ROOT"
}

expires {
    enableExpires           1
    expiresByType           image/*=A43200,text/css=A43200,application/x-javascript=A43200,application/javascript=A43200,font/*=A43200,application/x-font-ttf=A43200
}

EOF
  touch /www/server/panel/vhost/openlitespeed/listen/888.conf
  cat > /www/server/panel/vhost/openlitespeed/listen/888.conf <<- EOF
listener Default888{
    address *:888
    secure 0
    map phpmyadmin *
}
EOF

}

write_phpmyadmin_ssl() {
  phpmyadmin_ssl_port=$(grep 'listen' /www/server/panel/vhost/nginx/phpmyadmin.conf |awk '{print $2}')
  if [ ! -f "/www/server/panel/vhost/openlitespeed/listen/887.conf" ];then
    cat > /www/server/panel/vhost/openlitespeed/listen/887.conf <<- EOF
listener SSL887 {
  map phpmyadmin *
  address                 *:${phpmyadmin_ssl_port}
  secure                  1
  keyFile                 /www/server/panel/ssl/privateKey.pem
  certFile                /www/server/panel/ssl/certificate.pem
  certChain               1
  sslProtocol             24
  ciphers                 EECDH+AESGCM:EDH+AESGCM:AES256+EECDH:AES256+EDH:ECDHE-RSA-AES128-GCM-SHA384:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA128:DHE-RSA-AES128-GCM-SHA384:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES128-GCM-SHA128:ECDHE-RSA-AES128-SHA384:ECDHE-RSA-AES128-SHA128:ECDHE-RSA-AES128-SHA:ECDHE-RSA-AES128-SHA:DHE-RSA-AES128-SHA128:DHE-RSA-AES128-SHA128:DHE-RSA-AES128-SHA:DHE-RSA-AES128-SHA:ECDHE-RSA-DES-CBC3-SHA:EDH-RSA-DES-CBC3-SHA:AES128-GCM-SHA384:AES128-GCM-SHA128:AES128-SHA128:AES128-SHA128:AES128-SHA:AES128-SHA:DES-CBC3-SHA:HIGH:!aNULL:!eNULL:!EXPORT:!DES:!MD5:!PSK:!RC4
  enableECDHE             1
  renegProtection         1
  sslSessionCache         1
  enableSpdy              15
  enableStapling           1
  ocspRespMaxAge           86400
}
EOF
  fi
}

Uninstall_OLS () {
  /usr/local/lsws/bin/lswsctrl stop
  if [ -f "/etc/redhat-release" ];then
    yum remove openlitespeed -y
    rm -rf /usr/local/lsws/bin
    rm -rf /usr/local/lsws/conf_bt
    mv /usr/local/lsws/conf /usr/local/lsws/conf_bt
  else
    apt remove openlitespeed -y
    rm -rf /usr/local/lsws/bin
    rm -rf /usr/local/lsws/conf_bt
    mv /usr/local/lsws/conf /usr/local/lsws/conf_bt
  fi

}


actionType=$1
ols_version=$2
if [ "$ols_version" == "1.6" ];then
  ols_version=1.6.21
fi

if [ "${actionType}" == "uninstall" ]; then
	Uninstall_OLS
else
  check_panel_version
	if [ "${actionType}" == "install" ];then
		if [ -f "/usr/local/lsws/bin/lswsctrl" ]; then
			Uninstall_OLS
		fi
    Install_OLS
  else
    update_OLS
	fi
fi
rm -rf /tmp/openlitespeed
rm -f /tmp/openlitespeed-${ols_version}.tgz