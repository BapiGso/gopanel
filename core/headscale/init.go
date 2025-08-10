//go:build linux || darwin || freebsd

package headscale

import (
	"github.com/spf13/viper"
	"log"
	"time"
)

func init() {
	// 在系统启动后延迟启动 Headscale
	go func() {
		// 等待系统初始化完成
		time.Sleep(5 * time.Second)

		// 检查是否启用了 Headscale
		if viper.GetBool("enable.headscale") {
			log.Println("Auto-starting Headscale service...")

			// 尝试启动 Headscale
			if err := StartHeadscale(); err != nil {
				log.Printf("Failed to auto-start Headscale: %v", err)
			} else {
				log.Println("Headscale service started successfully")
			}
		}
	}()
}
