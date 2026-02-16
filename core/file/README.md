### 文件管理代码存放

## 重构后架构

```
┌─────────────────┐
│   Echo Handler  │  ← HTTP 层，参数绑定，响应格式化
│   (handler.go)  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  FileService    │  ← 业务逻辑层，定义接口
│   Interface     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ LocalFileService│  ← 数据访问层，使用 afero 抽象文件系统
│   (service.go)  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   afero.Fs      │  ← 可以是 OsFs(真实) 或 MemMapFs(内存)
└─────────────────┘
```

## 改进点

### 1. 分层架构
- **Handler 层**: 只处理 HTTP 相关逻辑
- **Service 层**: 业务逻辑，可替换实现
- **Repository 层**: 通过 afero 抽象文件系统

### 2. 依赖注入
```go
// 生产环境
handler := file.NewHandler("/var/www")

// 测试环境 - 使用内存文件系统
memFS := afero.NewMemMapFs()
service := &LocalFileService{fs: memFS, basePath: "/app"}
```

### 3. 统一错误处理
使用预定义错误，便于判断：
- `ErrPathNotAllowed` - 路径越界
- `ErrFileNotFound` - 文件不存在
- `ErrFileTooLarge` - 文件超过2MB
- `ErrInvalidUTF8` - 非UTF-8编码
- `ErrInvalidPerm` - 权限格式错误

### 4. 可测试性
```go
func TestUpload(t *testing.T) {
    // 不需要创建真实文件，使用内存文件系统
    service := NewLocalFileServiceWithFS(afero.NewMemMapFs(), "/app")

    // 测试逻辑...
    err := service.Create("/test.txt", false)
    assert.NoError(t, err)
}
```

## 使用方式

### 基本使用
```go
import "gopanel/core/file"

// 自动使用 FILE_BASE_PATH 环境变量
handler := file.DefaultHandler()

// 或指定路径
handler := file.NewHandler("/var/www")

// 注册路由
e.GET("/admin/file", handler.Index)
e.Any("/admin/file/process", handler.Process)
```

### 兼容旧代码
```go
// 旧代码仍然可以工作
e.GET("/admin/file", file.Index)
e.Any("/admin/file/process", file.Process)
```

## 扩展指南

### 添加新功能
在 `service.go` 中添加方法：
```go
func (s *LocalFileService) Copy(src, dst string) error {
    // 实现...
}
```

在 `handler.go` 中处理：
```go
func (h *Handler) handleCopy(c *echo.Context, src, dst string) error {
    if err := h.service.Copy(src, dst); err != nil {
        return h.handleError(c, err)
    }
    return c.JSON(200, map[string]string{"status": "success"})
}
```

### 替换存储后端
可以实现 FileService 接口使用 S3：
```go
type S3FileService struct {
    s3Client *s3.Client
    bucket   string
}

func (s *S3FileService) Read(path string) (*FileContent, error) {
    // S3 实现
}
```
