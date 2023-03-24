package cron

import (
	"fmt"
	"github.com/go-co-op/gocron"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func packFolder(s *gocron.Scheduler, path string) error {

	timestamp := time.Now().Unix()

	archiveName := fmt.Sprintf("%s_%d.tar.gz", path, timestamp)

	cmd := fmt.Sprintf("tar czf %s %s", archiveName, path)
	err := exec.Command("/bin/bash", "-c", cmd).Run()

	return err
}

func clearNginxLog(s *gocron.Scheduler) {
	nginxLogCronJob := func() {
		logDir := "/var/log/nginx"

		// 获取符合条件的日志文件列表
		expireTime := time.Now().Add(-30 * time.Hour) // 保留最近 30 小时内的日志文件
		logFiles, _ := filepath.Glob(filepath.Join(logDir, "*.log*"))
		for _, file := range logFiles {
			info, err := os.Stat(file)
			if err != nil {
				continue
			}
			if info.ModTime().Before(expireTime) {
				err := os.Remove(file)
				if err != nil {
					fmt.Printf("Error removing log file %s: %v\n", file, err)
				} else {
					fmt.Printf("Log file %s removed\n", file)
				}
			}
		}

		// 重新打开Nginx日志文件
		cmd := exec.Command("kill", "-USR1", "$(cat /run/nginx.pid)")
		err := cmd.Run()
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println("Nginx log files cleaned up successfully")
	}

	// 每小时执行清理 Nginx 日志文件的任务
	s.Every(1).Hour().Do(nginxLogCronJob)

	// 开始执行定时任务
	s.StartAsync()
}

func backupDB(s *gocron.Scheduler) {
	// 创建函数，备份数据库
	backupCronJob := func() {
		now := time.Now()
		filename := fmt.Sprintf("backup_%s.sql.gz", now.Format("20060102_150405"))

		cmd := exec.Command("mysqldump", "--single-transaction", "-h", "localhost", "-u", "root", "-p", "mydb")
		out, err := cmd.StdoutPipe()
		if err != nil {
			fmt.Println(err)
			return
		}

		gzCmd := exec.Command("gzip", "-c")
		gzFile, err := os.Create(filename)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer gzFile.Close()

		gzCmd.Stderr = cmd.Stderr
		gzCmd.Stdin = out
		gzCmd.Stdout = gzFile

		err = gzCmd.Start()
		if err != nil {
			fmt.Println(err)
			return
		}

		err = cmd.Run()
		if err != nil {
			fmt.Println(err)
			return
		}

		err = gzCmd.Wait()
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Printf("Backup saved to %s\n", filename)
	}

	// 每周 7 天 3:00 执行备份数据库任务
	s.Every(1).Day().At("10:30;08:00").Do(backupCronJob)

	// 开始执行定时任务
	s.StartAsync()
}
