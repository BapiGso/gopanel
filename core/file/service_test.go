package file

import (
	"testing"

	"github.com/spf13/afero"
)

// TestWithMemoryFS 使用内存文件系统测试（不需要实际文件）
func TestWithMemoryFS(t *testing.T) {
	// 创建内存文件系统
	memFS := afero.NewMemMapFs()

	// 创建服务
	service := &LocalFileService{
		fs:       memFS,
		basePath: "/app",
		maxSize:  1024 * 1024,
	}

	// 预先创建目录
	_ = memFS.MkdirAll("/app", 0755)

	// 测试创建文件
	t.Run("Create File", func(t *testing.T) {
		err := service.Create("/test.txt", false)
		if err != nil {
			t.Errorf("create file failed: %v", err)
		}
	})

	// 测试写入内容
	t.Run("Write File", func(t *testing.T) {
		err := service.Write("/test.txt", []byte("hello world"))
		if err != nil {
			t.Errorf("write file failed: %v", err)
		}
	})

	// 测试读取
	t.Run("Read File", func(t *testing.T) {
		content, err := service.Read("/test.txt")
		if err != nil {
			t.Errorf("read file failed: %v", err)
		}
		if content.Data != "hello world" {
			t.Errorf("content mismatch: got %s", content.Data)
		}
	})

	// 测试路径遍历保护
	t.Run("Path Traversal Protection", func(t *testing.T) {
		_, err := service.ValidatePath("../../../etc/passwd")
		if err == nil {
			t.Error("path traversal should be blocked")
		}
	})

	// 测试列出目录
	t.Run("List Directory", func(t *testing.T) {
		// 创建一些文件
		_ = service.Create("/file1.txt", false)
		_ = service.Create("/folder1", true)

		files, err := service.List("/")
		if err != nil {
			t.Errorf("list directory failed: %v", err)
		}
		if len(files) == 0 {
			t.Error("expected files in directory")
		}
	})
}

// ExampleNewLocalFileService 示例：如何使用
func ExampleNewLocalFileService() {
	// 创建服务（实际使用）
	service := NewLocalFileService("/var/www")

	// 使用内存文件系统测试
	memService := NewLocalFileService("/app")
	_ = memService

	// 使用
	_ = service
}
