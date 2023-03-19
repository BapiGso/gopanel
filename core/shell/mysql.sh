#!/bin/bash
PATH=/bin:/sbin:/usr/bin:/usr/sbin:/usr/local/bin:/usr/local/sbin:~/bin
export PATH

public_file=/www/server/panel/install/public.sh
publicFileMd5=$(md5sum ${public_file} 2>/dev/null|awk '{print $1}')
md5check="bbc7c9ebfee02b8e5158b509a6ad642d"
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

Root_Path=`cat /var/bt_setupPath.conf`
Setup_Path=$Root_Path/server/mysql
Data_Path=$Root_Path/server/data
Is_64bit=`getconf LONG_BIT`
run_path='/root'
mysql_51='5.1.73'
mysql_55='5.5.62'
mysql_56='5.6.50'
mysql_57='5.7.38'
mysql_80='8.0.24'
mariadb_55='5.5.55'
mysql_mariadb_100='10.0.38'
mysql_mariadb_101='10.1.47'
mysql_mariadb_102='10.2.38'
mysql_mariadb_103='10.3.29'
mysql_mariadb_104='10.4.19'
mysql_mariadb_105='10.5.10'
mysql_mariadb_106='10.6.7'
mysql_mariadb_107='10.7.3'
mysql_mariadb_108='10.8.2'
alisql_version='AliSQL-5.6.32'
Centos7Check=$(cat /etc/redhat-release | grep ' 7.' | grep -iE 'centos')
Centos8Check=$(cat /etc/redhat-release | grep ' 8.' | grep -iE 'centos|Red Hat')
CentosStream8Check=$(cat /etc/redhat-release |grep -i "Centos Stream"|grep 8)

if [ -z "${cpuCore}" ]; then
    cpuCore="1"
fi

MEM_INFO=$(free -m|grep Mem|awk '{printf("%.f",($2)/1024)}')
if [ "${cpuCore}" != "1" ] && [ "${MEM_INFO}" != "0" ];then
    if [ "${cpuCore}" -gt "${MEM_INFO}" ];then
        cpuCore="${MEM_INFO}"
    fi
else
    cpuCore="1"
fi

#检测hosts文件
hostfile=`cat /etc/hosts | grep 127.0.0.1 | grep localhost`
if [ "${hostfile}" = '' ]; then
    echo "127.0.0.1  localhost  localhost.localdomain" >> /etc/hosts
fi

System_Lib(){
    if [ "${PM}" == "yum" ] || [ "${PM}" == "dnf" ] ; then
        Pack="cmake libarchive"
        ${PM} install ${Pack} -y
    elif [ "${PM}" == "apt-get" ]; then
        Pack="cmake"
        ${PM} install ${Pack} -y
    fi
}

gccVersionCheck(){
    gccV=$(gcc -dumpversion|grep ^[789])
    if [ "${gccV}" ]; then
        sed -i "s/field_names\[i\]\[num_fields\*2\].*/field_names\[i\]\[num_fields\*2\]= NULL;/" client/mysql.cc
    fi
}

Service_Add(){
    if [ "${PM}" == "yum" ] || [ "${PM}" == "dnf" ]; then
        chkconfig --add mysqld
        chkconfig --level 2345 mysqld on
    elif [ "${PM}" == "apt-get" ]; then
        update-rc.d mysqld defaults
    fi
}
Service_Del(){
     if [ "${PM}" == "yum" ] || [ "${PM}" == "dnf" ]; then
        chkconfig --del mysqld
        chkconfig --level 2345 mysqld off
    elif [ "${PM}" == "apt-get" ]; then
        update-rc.d mysqld remove
    fi
}

printVersion(){
    if [ "${version}" = "alisql" ];then
        echo "${alisql_version}" > ${Setup_Path}/version.pl
    elif [ -z "${mariadbCheck}" ]; then
        echo "${sqlVersion}" > ${Setup_Path}/version.pl
    else
        echo "mariadb_${sqlVersion}" > ${Setup_Path}/version.pl
    fi
}
Install_Rpcgen(){
    if [ ! -f "/usr/bin/rpcgen" ];then
        wget ${download_Url}/src/rpcsvc-proto-1.4.tar.gz
        tar -xvf rpcsvc-proto-1.4.tar.gz
        cd rpcsvc-proto-1.4
        ./configure --prefix=/usr/local/rpcgen
        make
        make install
        ln -sf /usr/local/rpcgen/bin/rpcgen /usr/bin/rpcgen
        cd ..
        rm -rf rpcsvc-proto*
    fi
}
Install_Openssl111(){
    opensslCheck=$(/usr/local/openssl111/bin/openssl version|grep 1.1.1)
    if [ -z "${opensslCheck}" ]; then
        opensslVersion="1.1.1o"
        cd ${run_path}
        wget ${download_Url}/src/openssl-${opensslVersion}.tar.gz -T 20
        tar -zxf openssl-${opensslVersion}.tar.gz
        rm -f openssl-${opensslVersion}.tar.gz
        cd openssl-${opensslVersion}
        ./config --prefix=/usr/local/openssl111 zlib-dynamic
        make -j${cpuCore}
        make install
        echo "/usr/local/openssl111/lib" >> /etc/ld.so.conf.d/openssl111.conf
        ldconfig
        cd ..
        rm -rf openssl-${opensslVersion}
    fi
    WITH_SSL="-DWITH_SSL=/usr/local/openssl111"
}
Setup_Mysql_PyDb(){
    pyMysql=$1
    pyMysqlVer=$2

    wget -O src.zip ${download_Url}/install/src/${pyMysql}-${pyMysqlVer}.zip -T 20
    unzip src.zip
    mv ${pyMysql}-${pyMysqlVer} src
    cd src
    python setup.py install
    cd ..
    rm -f src.zip
    rm -rf src
    /etc/init.d/bt reload

}
Install_Mysql_PyDb(){
    pip uninstall MySQL-python mysqlclient PyMySQL -y
    pipUrl=$(cat /root/.pip/pip.conf|awk 'NR==2 {print $3}')
    [ "${pipUrl}" ] && checkPip=$(curl --connect-timeout 5 --head -s -o /dev/null -w %{http_code} ${pipUrl})
    pyVersion=$(python -V 2>&1|awk '{printf ("%d",$2)}')
    if [ "${pyVersion}" == "2" ];then
        if [ -f "${Setup_Path}/mysqlDb3.pl" ]; then
            local pyMysql="mysqlclient"
            local pyMysqlVer="1.3.12"
        else
            local pyMysql="MySQL-python"
            local pyMysqlVer="1.2.5"
        fi
        if [ "${checkPip}" = "200" ];then
            pip install ${pyMysql}
        else
            Setup_Mysql_PyDb ${pyMysql} ${pyMysqlVer}
        fi
    fi

    if [ "${checkPip}" = "200" ];then
        pip install PyMySQL
    else
        Setup_Mysql_PyDb "PyMySQL" "0.9.3"
    fi
}

Drop_Test_Databashes(){
    sleep 1
    /etc/init.d/mysqld stop
    pkill -9 mysqld_safe
    pkill -9 mysql
    sleep 1
    /etc/init.d/mysqld start
    sleep 1
    /www/server/mysql/bin/mysql -uroot -p$mysqlpwd -e "drop database test";
    /www/server/mysql/bin/mysql -uroot -p$mysqlpwd -e "delete from mysql.user where user='';"
    /www/server/mysql/bin/mysql -uroot -p$mysqlpwd -e "flush privileges;"

}
#设置软件链
SetLink()
{
    ln -sf ${Setup_Path}/bin/mysql /usr/bin/mysql
    ln -sf ${Setup_Path}/bin/mysqldump /usr/bin/mysqldump
    ln -sf ${Setup_Path}/bin/myisamchk /usr/bin/myisamchk
    ln -sf ${Setup_Path}/bin/mysqld_safe /usr/bin/mysqld_safe
    ln -sf ${Setup_Path}/bin/mysqlcheck /usr/bin/mysqlcheck
    ln -sf ${Setup_Path}/bin/mysql_config /usr/bin/mysql_config

    rm -f /usr/lib/libmysqlclient.so.16
    rm -f /usr/lib64/libmysqlclient.so.16
    rm -f /usr/lib/libmysqlclient.so.18
    rm -f /usr/lib64/libmysqlclient.so.18
    rm -f /usr/lib/libmysqlclient.so.20
    rm -f /usr/lib64/libmysqlclient.so.20
    rm -f /usr/lib/libmysqlclient.so.21
    rm -f /usr/lib64/libmysqlclient.so.21

    if [ -f "${Setup_Path}/lib/libmysqlclient.so.18" ];then
        ln -sf ${Setup_Path}/lib/libmysqlclient.so.18 /usr/lib/libmysqlclient.so.16
        ln -sf ${Setup_Path}/lib/libmysqlclient.so.18 /usr/lib64/libmysqlclient.so.16
        ln -sf ${Setup_Path}/lib/libmysqlclient.so.18 /usr/lib/libmysqlclient.so.18
        ln -sf ${Setup_Path}/lib/libmysqlclient.so.18 /usr/lib64/libmysqlclient.so.18
        ln -sf ${Setup_Path}/lib/libmysqlclient.so.18 /usr/lib/libmysqlclient.so.20
        ln -sf ${Setup_Path}/lib/libmysqlclient.so.18 /usr/lib64/libmysqlclient.so.20
    elif [ -f "${Setup_Path}/lib/mysql/libmysqlclient.so.18" ];then
        ln -sf ${Setup_Path}/lib/mysql/libmysqlclient.so.18 /usr/lib/libmysqlclient.so.16
        ln -sf ${Setup_Path}/lib/mysql/libmysqlclient.so.18 /usr/lib64/libmysqlclient.so.16
        ln -sf ${Setup_Path}/lib/mysql/libmysqlclient.so.18 /usr/lib/libmysqlclient.so.18
        ln -sf ${Setup_Path}/lib/mysql/libmysqlclient.so.18 /usr/lib64/libmysqlclient.so.18
        ln -sf ${Setup_Path}/lib/mysql/libmysqlclient.so.18 /usr/lib/libmysqlclient.so.20
        ln -sf ${Setup_Path}/lib/mysql/libmysqlclient.so.18 /usr/lib64/libmysqlclient.so.20
    elif [ -f "${Setup_Path}/lib/libmysqlclient.so.16" ];then
        ln -sf ${Setup_Path}/lib/libmysqlclient.so.16 /usr/lib/libmysqlclient.so.16
        ln -sf ${Setup_Path}/lib/libmysqlclient.so.16 /usr/lib64/libmysqlclient.so.16
        ln -sf ${Setup_Path}/lib/libmysqlclient.so.16 /usr/lib/libmysqlclient.so.18
        ln -sf ${Setup_Path}/lib/libmysqlclient.so.16 /usr/lib64/libmysqlclient.so.18
        ln -sf ${Setup_Path}/lib/libmysqlclient.so.16 /usr/lib/libmysqlclient.so.20
        ln -sf ${Setup_Path}/lib/libmysqlclient.so.16 /usr/lib64/libmysqlclient.so.20
    elif [ -f "${Setup_Path}/lib/mysql/libmysqlclient.so.16" ];then
        ln -sf ${Setup_Path}/lib/mysql/libmysqlclient.so.16 /usr/lib/libmysqlclient.so.16
        ln -sf ${Setup_Path}/lib/mysql/libmysqlclient.so.16 /usr/lib64/libmysqlclient.so.16
        ln -sf ${Setup_Path}/lib/mysql/libmysqlclient.so.16 /usr/lib/libmysqlclient.so.18
        ln -sf ${Setup_Path}/lib/mysql/libmysqlclient.so.16 /usr/lib64/libmysqlclient.so.18
        ln -sf ${Setup_Path}/lib/mysql/libmysqlclient.so.16 /usr/lib/libmysqlclient.so.20
        ln -sf ${Setup_Path}/lib/mysql/libmysqlclient.so.16 /usr/lib64/libmysqlclient.so.20
    elif [ -f "${Setup_Path}/lib/libmysqlclient_r.so.16" ];then
        ln -sf ${Setup_Path}/lib/libmysqlclient_r.so.16 /usr/lib/libmysqlclient_r.so.16
        ln -sf ${Setup_Path}/lib/libmysqlclient_r.so.16 /usr/lib64/libmysqlclient_r.so.16
    elif [ -f "${Setup_Path}/lib/mysql/libmysqlclient_r.so.16" ];then
        ln -sf ${Setup_Path}/lib/mysql/libmysqlclient_r.so.16 /usr/lib/libmysqlclient_r.so.16
        ln -sf ${Setup_Path}/lib/mysql/libmysqlclient_r.so.16 /usr/lib64/libmysqlclient_r.so.16
    elif [ -f "${Setup_Path}/lib/libmysqlclient.so.20" ];then
        ln -sf ${Setup_Path}/lib/libmysqlclient.so.20 /usr/lib/libmysqlclient.so.16
        ln -sf ${Setup_Path}/lib/libmysqlclient.so.20 /usr/lib64/libmysqlclient.so.16
        ln -sf ${Setup_Path}/lib/libmysqlclient.so.20 /usr/lib/libmysqlclient.so.18
        ln -sf ${Setup_Path}/lib/libmysqlclient.so.20 /usr/lib64/libmysqlclient.so.18
        ln -sf ${Setup_Path}/lib/libmysqlclient.so.20 /usr/lib/libmysqlclient.so.20
        ln -sf ${Setup_Path}/lib/libmysqlclient.so.20 /usr/lib64/libmysqlclient.so.20
    elif [ -f "${Setup_Path}/lib/libmysqlclient.so.21" ];then
        ln -sf ${Setup_Path}/lib/libmysqlclient.so.21 /usr/lib/libmysqlclient.so.16
        ln -sf ${Setup_Path}/lib/libmysqlclient.so.21 /usr/lib64/libmysqlclient.so.16
        ln -sf ${Setup_Path}/lib/libmysqlclient.so.21 /usr/lib/libmysqlclient.so.18
        ln -sf ${Setup_Path}/lib/libmysqlclient.so.21 /usr/lib64/libmysqlclient.so.18
        ln -sf ${Setup_Path}/lib/libmysqlclient.so.21 /usr/lib/libmysqlclient.so.20
        ln -sf ${Setup_Path}/lib/libmysqlclient.so.21 /usr/lib64/libmysqlclient.so.20
        ln -sf ${Setup_Path}/lib/libmysqlclient.so.21 /usr/lib/libmysqlclient.so.21
        ln -sf ${Setup_Path}/lib/libmysqlclient.so.21 /usr/lib64/libmysqlclient.so.21
    elif [ -f "${Setup_Path}/lib/libmariadb.so.3" ]; then
        ln -sf ${Setup_Path}/lib/libmariadb.so.3 /usr/lib/libmysqlclient.so.16
        ln -sf ${Setup_Path}/lib/libmariadb.so.3 /usr/lib64/libmysqlclient.so.16
        ln -sf ${Setup_Path}/lib/libmariadb.so.3 /usr/lib/libmysqlclient.so.18
        ln -sf ${Setup_Path}/lib/libmariadb.so.3 /usr/lib64/libmysqlclient.so.18
        ln -sf ${Setup_Path}/lib/libmariadb.so.3 /usr/lib/libmysqlclient.so.20
        ln -sf ${Setup_Path}/lib/libmariadb.so.3 /usr/lib64/libmysqlclient.so.20
        ln -sf ${Setup_Path}/lib/libmariadb.so.3 /usr/lib/libmysqlclient.so.21
        ln -sf ${Setup_Path}/lib/libmariadb.so.3 /usr/lib64/libmysqlclient.so.21
    elif [ -f "${Setup_Path}/lib/mysql/libmysqlclient.so.20" ];then
        ln -sf ${Setup_Path}/lib/mysql/libmysqlclient.so.20 /usr/lib/libmysqlclient.so.16
        ln -sf ${Setup_Path}/lib/mysql/libmysqlclient.so.20 /usr/lib64/libmysqlclient.so.16
        ln -sf ${Setup_Path}/lib/mysql/libmysqlclient.so.20 /usr/lib/libmysqlclient.so.18
        ln -sf ${Setup_Path}/lib/mysql/libmysqlclient.so.20 /usr/lib64/libmysqlclient.so.18
        ln -sf ${Setup_Path}/lib/mysql/libmysqlclient.so.20 /usr/lib/libmysqlclient.so.20
        ln -sf ${Setup_Path}/lib/mysql/libmysqlclient.so.20 /usr/lib64/libmysqlclient.so.20
    fi
}

My_Cnf(){
    if [ "${version}" == "5.1" ]; then
        defaultEngine="MyISAM"

    else
        defaultEngine="InnoDB"
    fi
    cat > /etc/my.cnf<<EOF
[client]
#password	= your_password
port		= 3306
socket		= /tmp/mysql.sock

[mysqld]
port		= 3306
socket		= /tmp/mysql.sock
datadir = ${Data_Path}
default_storage_engine = ${defaultEngine}
skip-external-locking
key_buffer_size = 8M
max_allowed_packet = 100G
table_open_cache = 32
sort_buffer_size = 256K
net_buffer_length = 4K
read_buffer_size = 128K
read_rnd_buffer_size = 256K
myisam_sort_buffer_size = 4M
thread_cache_size = 4
query_cache_size = 4M
tmp_table_size = 8M
sql-mode=NO_ENGINE_SUBSTITUTION,STRICT_TRANS_TABLES

#skip-name-resolve
max_connections = 500
max_connect_errors = 100
open_files_limit = 65535

log-bin=mysql-bin
binlog_format=mixed
server-id = 1
slow_query_log=1
slow-query-log-file=${Data_Path}/mysql-slow.log
long_query_time=3
#log_queries_not_using_indexes=on


innodb_data_home_dir = ${Data_Path}
innodb_data_file_path = ibdata1:10M:autoextend
innodb_log_group_home_dir = ${Data_Path}
innodb_buffer_pool_size = 16M
innodb_log_file_size = 5M
innodb_log_buffer_size = 8M
innodb_flush_log_at_trx_commit = 1
innodb_lock_wait_timeout = 50

[mysqldump]
quick
max_allowed_packet = 500M

[mysql]
no-auto-rehash

[myisamchk]
key_buffer_size = 20M
sort_buffer_size = 20M
read_buffer = 2M
write_buffer = 2M

[mysqlhotcopy]
interactive-timeout
EOF

    if [ "${version}" == "8.0" ]; then
        sed -i '/server-id/a\binlog_expire_logs_seconds = 600000' /etc/my.cnf
        sed -i '/tmp_table_size/a\default_authentication_plugin = mysql_native_password' /etc/my.cnf
        sed -i '/default_authentication_plugin/a\lower_case_table_names = 1' /etc/my.cnf
        sed -i '/query_cache_size/d' /etc/my.cnf
    else
        sed -i '/server-id/a\expire_logs_days = 10' /etc/my.cnf
    fi

    if [ "${version}" != "5.5" ];then
        if [ "${version}" != "5.1" ]; then
            sed -i '/skip-external-locking/i\table_definition_cache = 400' /etc/my.cnf
            sed -i '/table_definition_cache/i\performance_schema_max_table_instances = 400' /etc/my.cnf
        fi
    fi

    if [ "${version}" != "5.1" ]; then
        sed -i '/innodb_lock_wait_timeout/a\innodb_max_dirty_pages_pct = 90' /etc/my.cnf
        sed -i '/innodb_max_dirty_pages_pct/a\innodb_read_io_threads = 4' /etc/my.cnf
        sed -i '/innodb_read_io_threads/a\innodb_write_io_threads = 4' /etc/my.cnf
    fi

    [ "${version}" == "5.1" ] || [ "${version}" == "5.5" ] && sed -i '/STRICT_TRANS_TABLES/d' /etc/my.cnf
    [ "${version}" == "5.7" ] || [ "${version}" == "8.0" ] && sed -i '/#log_queries_not_using_indexes/a\early-plugin-load = ""' /etc/my.cnf
    [ "${version}" == "5.6" ] || [ "${version}" == "5.7" ] || [ "${version}" == "8.0" ] && sed -i '/#skip-name-resolve/i\explicit_defaults_for_timestamp = true' /etc/my.cnf
    chmod 644 /etc/my.cnf
}
MySQL_Opt()
{
    cpuInfo=`cat /proc/cpuinfo |grep "processor"|wc -l`
    sed -i 's/innodb_write_io_threads = 4/innodb_write_io_threads = '${cpuInfo}'/g' /etc/my.cnf
    sed -i 's/innodb_read_io_threads = 4/innodb_read_io_threads = '${cpuInfo}'/g' /etc/my.cnf
    MemTotal=`free -m | grep Mem | awk '{print  $2}'`
    if [[ ${MemTotal} -gt 1024 && ${MemTotal} -lt 2048 ]]; then
        sed -i "s#^key_buffer_size.*#key_buffer_size = 32M#" /etc/my.cnf
        sed -i "s#^table_open_cache.*#table_open_cache = 128#" /etc/my.cnf
        sed -i "s#^sort_buffer_size.*#sort_buffer_size = 768K#" /etc/my.cnf
        sed -i "s#^read_buffer_size.*#read_buffer_size = 768K#" /etc/my.cnf
        sed -i "s#^myisam_sort_buffer_size.*#myisam_sort_buffer_size = 8M#" /etc/my.cnf
        sed -i "s#^thread_cache_size.*#thread_cache_size = 16#" /etc/my.cnf
        sed -i "s#^query_cache_size.*#query_cache_size = 16M#" /etc/my.cnf
        sed -i "s#^tmp_table_size.*#tmp_table_size = 32M#" /etc/my.cnf
        sed -i "s#^innodb_buffer_pool_size.*#innodb_buffer_pool_size = 128M#" /etc/my.cnf
        sed -i "s#^innodb_log_file_size.*#innodb_log_file_size = 64M#" /etc/my.cnf
        sed -i "s#^innodb_log_buffer_size.*#innodb_log_buffer_size = 16M#" /etc/my.cnf
    elif [[ ${MemTotal} -ge 2048 && ${MemTotal} -lt 4096 ]]; then
        sed -i "s#^key_buffer_size.*#key_buffer_size = 64M#" /etc/my.cnf
        sed -i "s#^table_open_cache.*#table_open_cache = 256#" /etc/my.cnf
        sed -i "s#^sort_buffer_size.*#sort_buffer_size = 1M#" /etc/my.cnf
        sed -i "s#^read_buffer_size.*#read_buffer_size = 1M#" /etc/my.cnf
        sed -i "s#^myisam_sort_buffer_size.*#myisam_sort_buffer_size = 16M#" /etc/my.cnf
        sed -i "s#^thread_cache_size.*#thread_cache_size = 32#" /etc/my.cnf
        sed -i "s#^query_cache_size.*#query_cache_size = 32M#" /etc/my.cnf
        sed -i "s#^tmp_table_size.*#tmp_table_size = 64M#" /etc/my.cnf
        sed -i "s#^innodb_buffer_pool_size.*#innodb_buffer_pool_size = 256M#" /etc/my.cnf
        sed -i "s#^innodb_log_file_size.*#innodb_log_file_size = 128M#" /etc/my.cnf
        sed -i "s#^innodb_log_buffer_size.*#innodb_log_buffer_size = 32M#" /etc/my.cnf
    elif [[ ${MemTotal} -ge 4096 && ${MemTotal} -lt 8192 ]]; then
        sed -i "s#^key_buffer_size.*#key_buffer_size = 128M#" /etc/my.cnf
        sed -i "s#^table_open_cache.*#table_open_cache = 512#" /etc/my.cnf
        sed -i "s#^sort_buffer_size.*#sort_buffer_size = 2M#" /etc/my.cnf
        sed -i "s#^read_buffer_size.*#read_buffer_size = 2M#" /etc/my.cnf
        sed -i "s#^myisam_sort_buffer_size.*#myisam_sort_buffer_size = 32M#" /etc/my.cnf
        sed -i "s#^thread_cache_size.*#thread_cache_size = 64#" /etc/my.cnf
        sed -i "s#^query_cache_size.*#query_cache_size = 64M#" /etc/my.cnf
        sed -i "s#^tmp_table_size.*#tmp_table_size = 64M#" /etc/my.cnf
        sed -i "s#^innodb_buffer_pool_size.*#innodb_buffer_pool_size = 512M#" /etc/my.cnf
        sed -i "s#^innodb_log_file_size.*#innodb_log_file_size = 256M#" /etc/my.cnf
        sed -i "s#^innodb_log_buffer_size.*#innodb_log_buffer_size = 64M#" /etc/my.cnf
    elif [[ ${MemTotal} -ge 8192 && ${MemTotal} -lt 16384 ]]; then
        sed -i "s#^key_buffer_size.*#key_buffer_size = 256M#" /etc/my.cnf
        sed -i "s#^table_open_cache.*#table_open_cache = 1024#" /etc/my.cnf
        sed -i "s#^sort_buffer_size.*#sort_buffer_size = 4M#" /etc/my.cnf
        sed -i "s#^read_buffer_size.*#read_buffer_size = 4M#" /etc/my.cnf
        sed -i "s#^myisam_sort_buffer_size.*#myisam_sort_buffer_size = 64M#" /etc/my.cnf
        sed -i "s#^thread_cache_size.*#thread_cache_size = 128#" /etc/my.cnf
        sed -i "s#^query_cache_size.*#query_cache_size = 128M#" /etc/my.cnf
        sed -i "s#^tmp_table_size.*#tmp_table_size = 128M#" /etc/my.cnf
        sed -i "s#^innodb_buffer_pool_size.*#innodb_buffer_pool_size = 1024M#" /etc/my.cnf
        sed -i "s#^innodb_log_file_size.*#innodb_log_file_size = 512M#" /etc/my.cnf
        sed -i "s#^innodb_log_buffer_size.*#innodb_log_buffer_size = 128M#" /etc/my.cnf
    elif [[ ${MemTotal} -ge 16384 && ${MemTotal} -lt 32768 ]]; then
        sed -i "s#^key_buffer_size.*#key_buffer_size = 512M#" /etc/my.cnf
        sed -i "s#^table_open_cache.*#table_open_cache = 2048#" /etc/my.cnf
        sed -i "s#^sort_buffer_size.*#sort_buffer_size = 8M#" /etc/my.cnf
        sed -i "s#^read_buffer_size.*#read_buffer_size = 8M#" /etc/my.cnf
        sed -i "s#^myisam_sort_buffer_size.*#myisam_sort_buffer_size = 128M#" /etc/my.cnf
        sed -i "s#^thread_cache_size.*#thread_cache_size = 256#" /etc/my.cnf
        sed -i "s#^query_cache_size.*#query_cache_size = 256M#" /etc/my.cnf
        sed -i "s#^tmp_table_size.*#tmp_table_size = 256M#" /etc/my.cnf
        sed -i "s#^innodb_buffer_pool_size.*#innodb_buffer_pool_size = 2048M#" /etc/my.cnf
        sed -i "s#^innodb_log_file_size.*#innodb_log_file_size = 1024M#" /etc/my.cnf
        sed -i "s#^innodb_log_buffer_size.*#innodb_log_buffer_size = 256M#" /etc/my.cnf
    elif [[ ${MemTotal} -ge 32768 ]]; then
        sed -i "s#^key_buffer_size.*#key_buffer_size = 1024M#" /etc/my.cnf
        sed -i "s#^table_open_cache.*#table_open_cache = 4096#" /etc/my.cnf
        sed -i "s#^sort_buffer_size.*#sort_buffer_size = 16M#" /etc/my.cnf
        sed -i "s#^read_buffer_size.*#read_buffer_size = 16M#" /etc/my.cnf
        sed -i "s#^myisam_sort_buffer_size.*#myisam_sort_buffer_size = 256M#" /etc/my.cnf
        sed -i "s#^thread_cache_size.*#thread_cache_size = 512#" /etc/my.cnf
        sed -i "s#^query_cache_size.*#query_cache_size = 512M#" /etc/my.cnf
        sed -i "s#^tmp_table_size.*#tmp_table_size = 512M#" /etc/my.cnf
        sed -i "s#^innodb_buffer_pool_size.*#innodb_buffer_pool_size = 4096M#" /etc/my.cnf
    if [ "${version}" == "5.5" ];then
            sed -i "s#^innodb_log_file_size.*#innodb_log_file_size = 1024M#" /etc/my.cnf
            sed -i "s#^innodb_log_buffer_size.*#innodb_log_buffer_size = 256M#" /etc/my.cnf
        else
            sed -i "s#^innodb_log_file_size.*#innodb_log_file_size = 2048M#" /etc/my.cnf
            sed -i "s#^innodb_log_buffer_size.*#innodb_log_buffer_size = 512M#" /etc/my.cnf
        fi
    fi
    chmod 644 /etc/my.cnf
}
Install_Ready(){
    Close_MySQL
    cd ${run_path}
    mkdir -p ${Setup_Path}
    rm -rf ${Setup_Path}/*
    groupadd mysql
    useradd -s /sbin/nologin -M -g mysql mysql
}
Download_Src(){
    cd ${Setup_Path}

    mariadbCheck=$(echo ${version}|grep mariadb)
    if [ -z "${mariadbCheck}" ]; then
        sqlName="mysql"
    else
        sqlName="mariadb"
    fi

    if [ "${version}" == "5.7" ];then
        wget -O ${Setup_Path}/src.tar.gz ${download_Url}/src/mysql-boost-${sqlVersion}.tar.gz -T20
    elif [ "${version}" == "8.0" ]; then
        is_8_5=`cat /etc/os-release|grep VERSION=\"8.5\"`
        is_oracle=`cat /etc/os-release|awk '{FS=":"}{print $3}'|grep oracle`
        is_aarch64=$(uname -a|grep aarch64)
        if [ "$is_8_5" == "VERSION=\"8.5\"" ] && [ "$is_oracle" == "oracle" ] && [ "$is_aarch64" != "" ];then
            sqlVersion='8.0.22'
            wget -O ${Setup_Path}/src.tar.gz ${download_Url}/src/mysql-boost-${sqlVersion}.tar.gz -T20
        else
            wget -O ${Setup_Path}/src.tar.gz ${download_Url}/src/mysql-boost-${sqlVersion}.tar.gz -T20
        fi
    else
        wget -O ${Setup_Path}/src.tar.gz ${download_Url}/src/${sqlName}-${sqlVersion}.tar.gz -T20
    fi

    rm -rf ${sqlName}-${sqlVersion}
    rm -rf src
    tar -zxvf src.tar.gz

    armCheck=$(uname -m|grep arm)
    if [ "${version}" == "5.5" ] || [ "${armCheck}" ]; then
        wget -O mysql-5.5-fix-arm-client_plugin.patch ${download_Url}/src/patch/mysql-5.5-fix-arm-client_plugin.patch
        patch -p0 < mysql-5.5-fix-arm-client_plugin.patch
        rm -f mysql-5.5-fix-arm-client_plugin.patch
    fi

    mv ${sqlName}-${sqlVersion} src
    cd src
}
Install_Configure(){

    name=mysql
    i_path=/www/server/panel/install/$name

    i_args=$(cat $i_path/config.pl|xargs)
    i_make_args=""
    for i_name in $i_args
    do
        init_file=$i_path/$i_name/init.sh
        if [ -f $init_file ];then
            bash $init_file
        fi
        args_file=$i_path/$i_name/args.pl
        if [ -f $args_file ];then
            args_string=$(cat $args_file)
            i_make_args="$i_make_args $args_string"
        fi
    done

    cd src ${Setup_Path}/src
    if [ "${version}" == "5.1" ];then
        gccVersionCheck
        ./configure --prefix=${Setup_Path} --sysconfdir=/etc --with-plugins=csv,myisam,myisammrg,heap,innobase --with-extra-charsets=all --with-charset=utf8 --with-collation=utf8_general_ci --with-embedded-server --enable-local-infile --enable-assembler --with-mysqld-ldflags=-all-static --enable-thread-safe-client --with-big-tables --with-readline --with-ssl ${i_make_args}
    elif [ "${version}" == "5.5" ]; then
        gccVersionCheck
        cmake -DCMAKE_INSTALL_PREFIX=${Setup_Path} -DSYSCONFDIR=/etc -DWITH_MYISAM_STORAGE_ENGINE=1 -DWITH_INNOBASE_STORAGE_ENGINE=1 -DWITH_PARTITION_STORAGE_ENGINE=1 -DWITH_FEDERATED_STORAGE_ENGINE=1 -DEXTRA_CHARSETS=all -DDEFAULT_CHARSET=utf8mb4 -DDEFAULT_COLLATION=utf8mb4_general_ci -DWITH_READLINE=1 -DWITH_EMBEDDED_SERVER=1 -DENABLED_LOCAL_INFILE=1 ${i_make_args}
    elif [  "${version}" == "5.6" ]; then
        cmake -DCMAKE_INSTALL_PREFIX=${Setup_Path} -DSYSCONFDIR=/etc -DWITH_MYISAM_STORAGE_ENGINE=1 -DWITH_INNOBASE_STORAGE_ENGINE=1 -DWITH_PARTITION_STORAGE_ENGINE=1 -DWITH_FEDERATED_STORAGE_ENGINE=1 -DEXTRA_CHARSETS=all -DDEFAULT_CHARSET=utf8mb4 -DDEFAULT_COLLATION=utf8mb4_general_ci -DWITH_EMBEDDED_SERVER=1 -DENABLED_LOCAL_INFILE=1 ${WITH_SSL} ${i_make_args}
    elif [ "${version}" == "5.7" ]; then
        mkdir install
        cd install
        cmake .. -DCMAKE_INSTALL_PREFIX=${Setup_Path} -DSYSCONFDIR=/etc -DWITH_MYISAM_STORAGE_ENGINE=1 -DWITH_INNOBASE_STORAGE_ENGINE=1 -DWITH_PARTITION_STORAGE_ENGINE=1 -DWITH_FEDERATED_STORAGE_ENGINE=1 -DEXTRA_CHARSETS=all -DDEFAULT_CHARSET=utf8mb4 -DDEFAULT_COLLATION=utf8mb4_general_ci -DWITH_EMBEDDED_SERVER=1 -DENABLED_LOCAL_INFILE=1 -DWITH_BOOST=../boost ${WITH_SSL} ${i_make_args}
    elif [ "${version}" == "8.0" ]; then
        mkdir install
        cd install
        cmakeV="cmake"
        if [ "${PM}" = "yum" ]; then
            if [ "${Centos7Check}" ];then
                yum install centos-release-scl-rh -y
                yum install devtoolset-7-gcc devtoolset-7-gcc-c++ -y
                yum install cmake3 -y
                cmakeV="cmake3"
                export CC=/opt/rh/devtoolset-7/root/usr/bin/gcc
                export CXX=/opt/rh/devtoolset-7/root/usr/bin/g++
            else
                export CC=/usr/bin/gcc
                export CXX=/usr/bin/g++
            fi
        fi
        ${cmakeV} .. -DCMAKE_INSTALL_PREFIX=${Setup_Path} -DSYSCONFDIR=/etc -DWITH_MYISAM_STORAGE_ENGINE=1 -DWITH_INNOBASE_STORAGE_ENGINE=1 -DWITH_PARTITION_STORAGE_ENGINE=1 -DDEFAULT_CHARSET=utf8mb4 -DDEFAULT_COLLATION=utf8mb4_general_ci -DENABLED_LOCAL_INFILE=1 -DWITH_BOOST=../boost ${WITH_SSL} ${i_make_args}
    elif [ "${version}" == "alisql" ]; then
         cmake -DCMAKE_INSTALL_PREFIX=${Setup_Path} -DMYSQL_UNIX_ADDR=/tmp/mysql.sock -DDEFAULT_CHARSET=utf8   -DDEFAULT_COLLATION=utf8_general_ci -DWITH_INNOBASE_STORAGE_ENGINE=1 -DWITH_ARCHIVE_STORAGE_ENGINE=1 -DWITH_BLACKHOLE_STORAGE_ENGINE=1 -DMYSQL_DATADIR=${Data_Path} -DMYSQL_TCP_PORT=3306 -DENABLE_DOWNLOADS=1 ${i_make_args}
    elif [ "${version}" == "mariadb_10.0" ]; then
        cmake -DCMAKE_INSTALL_PREFIX=${Setup_Path} -DWITH_ARIA_STORAGE_ENGINE=1 -DWITH_XTRADB_STORAGE_ENGINE=1 -DWITH_INNOBASE_STORAGE_ENGINE=1 -DWITH_PARTITION_STORAGE_ENGINE=1 -DWITH_MYISAM_STORAGE_ENGINE=1 -DWITH_FEDERATED_STORAGE_ENGINE=1 -DEXTRA_CHARSETS=all -DDEFAULT_CHARSET=utf8mb4 -DDEFAULT_COLLATION=utf8mb4_general_ci -DWITH_READLINE=1 -DWITH_EMBEDDED_SERVER=1 -DENABLED_LOCAL_INFILE=1 ${i_make_args}
    elif [ "${version}" == "mariadb_10.6" ]; then
        cmakeV="cmake"
        if [ "${PM}" = "yum" ]; then
            if [ "${Centos7Check}" ];then
                yum install centos-release-scl-rh -y
                yum install devtoolset-7-gcc devtoolset-7-gcc-c++ -y
                yum install cmake3 -y
                cmakeV="cmake3"
                export CC=/opt/rh/devtoolset-7/root/usr/bin/gcc
                export CXX=/opt/rh/devtoolset-7/root/usr/bin/g++
            else
                export CC=/usr/bin/gcc
                export CXX=/usr/bin/g++
            fi
            ${cmakeV} -DCMAKE_INSTALL_PREFIX=${Setup_Path} -DWITH_ARIA_STORAGE_ENGINE=1 -DWITH_XTRADB_STORAGE_ENGINE=1 -DWITH_INNOBASE_STORAGE_ENGINE=1 -DWITH_PARTITION_STORAGE_ENGINE=1 -DWITH_MYISAM_STORAGE_ENGINE=1 -DWITH_FEDERATED_STORAGE_ENGINE=1 -DEXTRA_CHARSETS=all -DDEFAULT_CHARSET=utf8mb4 -DDEFAULT_COLLATION=utf8mb4_general_ci -DWITH_READLINE=1 -DWITH_EMBEDDED_SERVER=1 -DENABLED_LOCAL_INFILE=1 -DWITHOUT_TOKUDB=1 ${i_make_args}
        else
            cmake -DCMAKE_INSTALL_PREFIX=${Setup_Path} -DWITH_ARIA_STORAGE_ENGINE=1 -DWITH_XTRADB_STORAGE_ENGINE=1 -DWITH_INNOBASE_STORAGE_ENGINE=1 -DWITH_PARTITION_STORAGE_ENGINE=1 -DWITH_MYISAM_STORAGE_ENGINE=1 -DWITH_FEDERATED_STORAGE_ENGINE=1 -DEXTRA_CHARSETS=all -DDEFAULT_CHARSET=utf8mb4 -DDEFAULT_COLLATION=utf8mb4_general_ci -DWITH_READLINE=1 -DWITH_EMBEDDED_SERVER=1 -DENABLED_LOCAL_INFILE=1 -DWITHOUT_TOKUDB=1 ${i_make_args}
        fi
    elif [ "${version}" == "mariadb_10.7" ] || [ "${version}" == "mariadb_10.8" ]; then
        cmakeV="cmake"
        if [ "${PM}" = "yum" ]; then
            if [ "${Centos7Check}" ];then
                yum install centos-release-scl-rh -y
                yum install devtoolset-7-gcc devtoolset-7-gcc-c++ -y
                yum install cmake3 -y
                cmakeV="cmake3"
                export CC=/usr/bin/gcc
                export CXX=/usr/bin/g++
            fi
            ${cmakeV} -DCMAKE_INSTALL_PREFIX=${Setup_Path} -DWITH_ARIA_STORAGE_ENGINE=1 -DWITH_XTRADB_STORAGE_ENGINE=1 -DWITH_INNOBASE_STORAGE_ENGINE=1 -DWITH_PARTITION_STORAGE_ENGINE=1 -DWITH_MYISAM_STORAGE_ENGINE=1 -DWITH_FEDERATED_STORAGE_ENGINE=1 -DEXTRA_CHARSETS=all -DDEFAULT_CHARSET=utf8mb4 -DDEFAULT_COLLATION=utf8mb4_general_ci -DWITH_READLINE=1 -DWITH_EMBEDDED_SERVER=1 -DENABLED_LOCAL_INFILE=1 -DWITHOUT_TOKUDB=1 ${i_make_args}
        else
            cmake -DCMAKE_INSTALL_PREFIX=${Setup_Path} -DWITH_ARIA_STORAGE_ENGINE=1 -DWITH_XTRADB_STORAGE_ENGINE=1 -DWITH_INNOBASE_STORAGE_ENGINE=1 -DWITH_PARTITION_STORAGE_ENGINE=1 -DWITH_MYISAM_STORAGE_ENGINE=1 -DWITH_FEDERATED_STORAGE_ENGINE=1 -DEXTRA_CHARSETS=all -DDEFAULT_CHARSET=utf8mb4 -DDEFAULT_COLLATION=utf8mb4_general_ci -DWITH_READLINE=1 -DWITH_EMBEDDED_SERVER=1 -DENABLED_LOCAL_INFILE=1 -DWITHOUT_TOKUDB=1 ${i_make_args}
        fi
    else
        cmake -DCMAKE_INSTALL_PREFIX=${Setup_Path} -DWITH_ARIA_STORAGE_ENGINE=1 -DWITH_XTRADB_STORAGE_ENGINE=1 -DWITH_INNOBASE_STORAGE_ENGINE=1 -DWITH_PARTITION_STORAGE_ENGINE=1 -DWITH_MYISAM_STORAGE_ENGINE=1 -DWITH_FEDERATED_STORAGE_ENGINE=1 -DEXTRA_CHARSETS=all -DDEFAULT_CHARSET=utf8mb4 -DDEFAULT_COLLATION=utf8mb4_general_ci -DWITH_READLINE=1 -DWITH_EMBEDDED_SERVER=1 -DENABLED_LOCAL_INFILE=1 -DWITHOUT_TOKUDB=1 ${i_make_args}
    fi
    make -j${cpuCore}

    if [ $? -ne 0 ];then
        echo '========================================================'
        GetSysInfo
        echo -e "ERROR: mysql ${version} installation failed.";
        echo -e "安装失败，请截图以上报错信息发帖至论坛www.bt.cn/bbs求助"
        exit 1;
    fi
}
Install_Mysql(){
    if [ "${actionType}" == "update" ]; then
        /etc/init.d/mysqld stop
        sleep 2
        make install
        sleep 2
        /etc/init.d/mysqld start
        printVersion
        rm -f ${Setup_Path}/version_check.pl
        rm -f ${Setup_Path}/src.tar.gz
        rm -rf ${Setup_Path}/src
        exit 0;
    fi
    make install

    [ "${version}" == "8.0" ] || [ "${version}" == "mariadb_10.2" ] || [ "${version}" == "mariadb_10.3" ] || [ "${version}" == "mariadb_10.4" ]&& echo "True" > ${Setup_Path}/mysqlDb3.pl
}
Mysql_Initialize(){
    if [ -d "${Data_Path}" ]; then
        rm -rf ${Data_Path}/*
    else
        mkdir -p ${Data_Path}
    fi

    chown -R mysql:mysql ${Data_Path}
    chgrp -R mysql ${Setup_Path}/.

    if [ "${version}" == "mariadb_10.4" ] || [ "${version}" == "mariadb_10.5" ]; then
        mkdir -p ${Setup_Path}/lib/plugin/auth_pam_tool_dir/auth_pam_tool_dir
        wget -O ${Setup_Path}/scripts/mysql_install_db ${download_Url}/tools/mysql_install_db
        Authentication_Method="--auth-root-authentication-method=normal"
    fi

    if [ "${version}" == "5.1" ]; then
        ${Setup_Path}/bin/mysql_install_db --defaults-file=/etc/my.cnf --basedir=${Setup_Path} --datadir=${Data_Path} --user=mysql
    elif [ "${version}" == "5.7" ] || [ "${version}" == "8.0" ];then
        ${Setup_Path}/bin/mysqld --initialize-insecure --basedir=${Setup_Path} --datadir=${Data_Path} --user=mysql
    else
        ${Setup_Path}/scripts/mysql_install_db --defaults-file=/etc/my.cnf --basedir=${Setup_Path} --datadir=${Data_Path} --user=mysql ${Authentication_Method}
    fi

    \cp support-files/mysql.server /etc/init.d/mysqld
    chmod 755 /etc/init.d/mysqld

    if [ "${version}" != "5.7" ];then
        if [ "${version}" != "8.0" ]; then
            sed -i "s#\"\$\*\"#--sql-mode=\"NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION\"#" /etc/init.d/mysqld
        fi
    fi

    sed -i 's/$bindir\/mysqld_safe /&--defaults-file="\/etc\/my.cnf" /' /etc/init.d/mysqld
    sed -i '/case "$mode" in/i\ulimit -s unlimited' /etc/init.d/mysqld

    cat > /etc/ld.so.conf.d/mysql.conf<<EOF
${Setup_Path}/lib
EOF
    ldconfig
    ln -sf ${Setup_Path}/lib/mysql /usr/lib/mysql
    ln -sf ${Setup_Path}/include/mysql /usr/include/mysql
    /etc/init.d/mysqld start

    ${Setup_Path}/bin/mysqladmin -u root password "${mysqlpwd}"

    cd ${Setup_Path}
    rm -f src.tar.gz
    rm -rf src
    /etc/init.d/mysqld start

}

Bt_Check(){
    checkFile="/www/server/panel/install/check.sh"
    wget -O ${checkFile} ${download_Url}/tools/check.sh
    . ${checkFile}
}

Close_MySQL()
{
    [ -f "/etc/init.d/mysqld" ] && /etc/init.d/mysqld stop

    if [ "${PM}" = "yum" ];then
        mysqlVersion=`rpm -qa |grep bt-mysql-`
        mariadbVersion=`rpm -qa |grep bt-mariadb-`
        [ "${mysqlVersion}" ] && rpm -e $mysqlVersion --nodeps
        [ "${mariadbVersion}" ] && rpm -e $mariadbVersion --nodeps
        [ -f "${Setup_Path}/rpm.pl" ] && yum remove $(cat ${Setup_Path}/rpm.pl) -y
    elif [ "${PM}" = "apt-get" ]; then
        [ -f "${Setup_Path}/deb.pl" ] && apt-get remove $(cat ${Setup_Path}/deb.pl) -y
    fi

    if [ -f "${Setup_Path}/bin/mysql" ];then
        Service_Del
        rm -f /etc/init.d/mysqld
        rm -rf ${Setup_Path}
        mkdir -p /www/backup
        [ -d "/www/backup/oldData" ] && rm -rf /www/backup/oldData
        mv -f $Data_Path  /www/backup/oldData
        rm -rf $Data_Path
        rm -f /usr/bin/mysql*
        rm -f /usr/lib/libmysql*
        rm -f /usr/lib64/libmysql*
    fi
}

actionType=$1
version=$2

if [ "${actionType}" == 'install' ] || [ "${actionType}" == "update" ];then
    if [ -z "${version}" ]; then
        exit
    fi
    mysqlpwd=`cat /dev/urandom | head -n 16 | md5sum | head -c 16`
    case "$version" in
        '5.1')
            sqlVersion=${mysql_51}
            ;;
        '5.5')
            sqlVersion=${mysql_55}
            ;;
        '5.6')
            sqlVersion=${mysql_56}
            ;;
        '5.7')
            sqlVersion=${mysql_57}
            ;;
        '8.0')
            sqlVersion=${mysql_80}
            ;;
        'alisql')
            sqlVersion=${alisql_version}
            ;;
        'mariadb_10.0')
            sqlVersion=${mysql_mariadb_100}
            ;;
        'mariadb_10.1')
            sqlVersion=${mysql_mariadb_101}
            ;;
        'mariadb_10.2')
            sqlVersion=${mysql_mariadb_102}
            ;;
        'mariadb_10.3')
            sqlVersion=${mysql_mariadb_103}
            ;;
        'mariadb_10.4')
            sqlVersion=${mysql_mariadb_104}
            ;;
        'mariadb_10.5')
            sqlVersion=${mysql_mariadb_105}
            ;;
        'mariadb_10.6')
            sqlVersion=${mysql_mariadb_106}
            ;;
        'mariadb_10.7')
            sqlVersion=${mysql_mariadb_107}
            ;;
        'mariadb_10.8')
            sqlVersion=${mysql_mariadb_108}
            ;;
    esac
    System_Lib
    if [ "${actionType}" == "install" ]; then
        Install_Ready
    fi

    if [ -z "${Centos7Check}" ] && [ "${PM}" == "yum" ];then
        yum install libtirpc libtirpc-devel -y
    fi

    OPENSSL_30_VER=$(openssl version|grep '3.0')
    if [ "${OPENSSL_30_VER}" ];then
        Install_Openssl111
        if [ "${version}" == "8.0" ];then
            if [ "${PM}" == "yum" ];then
                yum install openldap-devel patchelf -y
            elif [ "${PM}" == "apt-get" ]; then
                apt-get install patchelf -y
            fi
        fi
    fi

    Install_Rpcgen
    Download_Src
    Install_Configure
    Install_Mysql
    My_Cnf
    MySQL_Opt
    Mysql_Initialize
    SetLink
    Service_Add
    printVersion
    Install_Mysql_PyDb
    if [ -f '/www/server/panel/tools.py' ];then
        python /www/server/panel/tools.py root $mysqlpwd
    else
        python /www/server/panel/tools.pyc root $mysqlpwd
    fi
    Drop_Test_Databashes
elif [ "$actionType" == 'uninstall' ];then
    Close_MySQL del
fi


