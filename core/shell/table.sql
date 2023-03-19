BEGIN TRANSACTION;
CREATE TABLE IF NOT EXISTS "backup" (
	"id"	INTEGER,
	"type"	INTEGER,
	"name"	TEXT,
	"pid"	INTEGER,
	"filename"	TEXT,
	"size"	INTEGER,
	"addtime"	TEXT,
	PRIMARY KEY("id" AUTOINCREMENT)
);
CREATE TABLE IF NOT EXISTS "binding" (
	"id"	INTEGER,
	"pid"	INTEGER,
	"domain"	TEXT,
	"path"	TEXT,
	"port"	INTEGER,
	"addtime"	TEXT,
	PRIMARY KEY("id" AUTOINCREMENT)
);
CREATE TABLE IF NOT EXISTS "config" (
	"id"	INTEGER,
	"webserver"	TEXT,
	"backup_path"	TEXT,
	"sites_path"	TEXT,
	"status"	INTEGER,
	"mysql_root"	TEXT,
	PRIMARY KEY("id" AUTOINCREMENT)
);
CREATE TABLE IF NOT EXISTS "crontab" (
	"id"	INTEGER,
	"name"	TEXT,
	"type"	TEXT,
	"where1"	TEXT,
	"where_hour"	INTEGER,
	"where_minute"	INTEGER,
	"echo"	TEXT,
	"addtime"	TEXT,
	"status"	INTEGER DEFAULT 1,
	"save"	INTEGER DEFAULT 3,
	"backupTo"	TEXT DEFAULT off,
	"sName"	TEXT,
	"sBody"	TEXT,
	"sType"	TEXT,
	"urladdress"	TEXT,
	PRIMARY KEY("id" AUTOINCREMENT)
);
CREATE TABLE IF NOT EXISTS "databases" (
	"id"	INTEGER,
	"pid"	INTEGER,
	"name"	TEXT,
	"username"	TEXT,
	"password"	TEXT,
	"accept"	TEXT,
	"ps"	TEXT,
	"addtime"	TEXT,
	PRIMARY KEY("id" AUTOINCREMENT)
);
CREATE TABLE IF NOT EXISTS "firewall" (
	"id"	INTEGER,
	"port"	TEXT,
	"ps"	TEXT,
	"addtime"	TEXT,
	PRIMARY KEY("id" AUTOINCREMENT)
);
CREATE TABLE IF NOT EXISTS "ftps" (
	"id"	INTEGER,
	"pid"	INTEGER,
	"name"	TEXT,
	"password"	TEXT,
	"path"	TEXT,
	"status"	TEXT,
	"ps"	TEXT,
	"addtime"	TEXT,
	PRIMARY KEY("id" AUTOINCREMENT)
);
CREATE TABLE IF NOT EXISTS "logs" (
	"id"	INTEGER,
	"type"	TEXT,
	"log"	TEXT,
	"addtime"	TEXT,
	"uid"	integer DEFAULT '1',
	"username"	TEXT DEFAULT 'system',
	PRIMARY KEY("id" AUTOINCREMENT)
);
CREATE TABLE IF NOT EXISTS "sites" (
	"id"	INTEGER,
	"name"	TEXT,
	"path"	TEXT,
	"status"	TEXT,
	"index"	TEXT,
	"ps"	TEXT,
	"addtime"	TEXT,
	"edate"	integer DEFAULT '0000-00-00',
	"type_id"	integer DEFAULT 0,
	PRIMARY KEY("id" AUTOINCREMENT)
);
CREATE TABLE IF NOT EXISTS "domain" (
	"id"	INTEGER,
	"pid"	INTEGER,
	"name"	TEXT,
	"port"	INTEGER,
	"addtime"	TEXT,
	PRIMARY KEY("id" AUTOINCREMENT)
);
CREATE TABLE IF NOT EXISTS "users" (
	"id"	INTEGER,
	"username"	TEXT,
	"password"	TEXT,
	"login_ip"	TEXT,
	"login_time"	TEXT,
	"phone"	TEXT,
	"email"	TEXT,
	"salt"	TEXT,
	PRIMARY KEY("id" AUTOINCREMENT)
);
CREATE TABLE IF NOT EXISTS "tasks" (
	"id"	INTEGER,
	"name"	TEXT,
	"type"	TEXT,
	"status"	TEXT,
	"addtime"	TEXT,
	"start"	INTEGER,
	"end"	INTEGER,
	"execstr"	TEXT,
	PRIMARY KEY("id" AUTOINCREMENT)
);
CREATE TABLE IF NOT EXISTS "task_list" (
	"id"	INTEGER,
	"name"	TEXT,
	"type"	TEXT,
	"status"	INTEGER,
	"shell"	TEXT,
	"other"	TEXT,
	"exectime"	INTEGER,
	"endtime"	INTEGER,
	"addtime"	INTEGER,
	PRIMARY KEY("id" AUTOINCREMENT)
);
CREATE TABLE IF NOT EXISTS "site_types" (
	"id"	INTEGER,
	"name"	REAL,
	"ps"	REAL,
	PRIMARY KEY("id" AUTOINCREMENT)
);
CREATE TABLE IF NOT EXISTS "download_token" (
	"id"	INTEGER,
	"token"	REAL,
	"filename"	REAL,
	"total"	INTEGER DEFAULT 0,
	"expire"	INTEGER,
	"password"	REAL,
	"ps"	REAL,
	"addtime"	INTEGER,
	PRIMARY KEY("id" AUTOINCREMENT)
);
CREATE TABLE IF NOT EXISTS "messages" (
	"id"	INTEGER,
	"level"	TEXT,
	"msg"	TEXT,
	"state"	INTEGER DEFAULT 0,
	"expire"	INTEGER,
	"addtime"	INTEGER,
	PRIMARY KEY("id" AUTOINCREMENT)
);
CREATE TABLE IF NOT EXISTS "temp_login" (
	"id"	INTEGER,
	"token"	REAL,
	"salt"	REAL,
	"state"	INTEGER,
	"login_time"	INTEGER,
	"login_addr"	REAL,
	"logout_time"	INTEGER,
	"expire"	INTEGER,
	"addtime"	INTEGER,
	PRIMARY KEY("id" AUTOINCREMENT)
);
CREATE TABLE IF NOT EXISTS "panel" (
	"id"	INTEGER,
	"title"	TEXT,
	"url"	TEXT,
	"username"	TEXT,
	"password"	TEXT,
	"click"	INTEGER,
	"addtime"	INTEGER,
	PRIMARY KEY("id" AUTOINCREMENT)
);
COMMIT;
