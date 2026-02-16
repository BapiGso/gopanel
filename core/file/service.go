package file

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/spf13/afero"
)

// 预定义错误
var (
	ErrFileTooLarge   = errors.New("file too large (max 2MB)")
	ErrInvalidUTF8    = errors.New("file is not valid UTF-8")
	ErrPathNotAllowed = errors.New("path not allowed")
	ErrFileNotFound   = errors.New("file not found")
	ErrInvalidPerm    = errors.New("invalid permission format")
)

// FileService 文件服务接口
type FileService interface {
	List(dir string) ([]os.FileInfo, error)
	Read(path string) (*FileContent, error)
	Download(path string) (io.ReadCloser, error)
	Create(path string, isDir bool) error
	Write(path string, data []byte) error
	Rename(oldPath, newPath string) error
	Chmod(path string, perm string) error
	Delete(path string) error
	ValidatePath(userPath string) (string, error)
}

// FileContent 文件内容结构
type FileContent struct {
	Type string `json:"type"`
	Data string `json:"data"`
	Size int64  `json:"size"`
}

// LocalFileService 本地文件服务实现
type LocalFileService struct {
	fs       afero.Fs
	basePath string
	maxSize  int64
}

// NewLocalFileService 创建本地文件服务
func NewLocalFileService(basePath string) *LocalFileService {
	return &LocalFileService{
		fs:       afero.NewOsFs(),
		basePath: filepath.Clean(basePath),
		maxSize:  2 * 1024 * 1024, // 2MB
	}
}

// ValidatePath 验证并清理路径
func (s *LocalFileService) ValidatePath(userPath string) (string, error) {
	cleanPath := filepath.Clean(userPath)
	cleanPath = strings.TrimPrefix(cleanPath, "/")
	cleanPath = strings.TrimPrefix(cleanPath, "\\")

	fullPath := filepath.Join(s.basePath, cleanPath)
	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("%w: invalid path", ErrPathNotAllowed)
	}

	absBasePath, _ := filepath.Abs(s.basePath)

	sep := string(filepath.Separator)
	if !strings.HasPrefix(absPath, absBasePath+sep) && absPath != absBasePath {
		return "", fmt.Errorf("%w: path outside base directory", ErrPathNotAllowed)
	}

	return absPath, nil
}

// List 列出目录内容
func (s *LocalFileService) List(dir string) ([]os.FileInfo, error) {
	path, err := s.ValidatePath(dir)
	if err != nil {
		return nil, err
	}

	info, err := s.fs.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFileNotFound
		}
		return nil, err
	}

	if !info.IsDir() {
		return nil, errors.New("path is not a directory")
	}

	files, err := afero.ReadDir(s.fs, path)
	if err != nil {
		return nil, err
	}

	// 文件夹排在前面
	result := make([]os.FileInfo, len(files))
	idx := 0
	for _, f := range files {
		if f.IsDir() {
			result[idx] = f
			idx++
		}
	}
	for _, f := range files {
		if !f.IsDir() {
			result[idx] = f
			idx++
		}
	}

	return result, nil
}

// Read 读取文件内容（用于编辑）
func (s *LocalFileService) Read(path string) (*FileContent, error) {
	absPath, err := s.ValidatePath(path)
	if err != nil {
		return nil, err
	}

	info, err := s.fs.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFileNotFound
		}
		return nil, err
	}

	if info.IsDir() {
		return nil, errors.New("cannot read directory as file")
	}

	if info.Size() > s.maxSize {
		return nil, ErrFileTooLarge
	}

	data, err := afero.ReadFile(s.fs, absPath)
	if err != nil {
		return nil, err
	}

	if !utf8.Valid(data) {
		return nil, ErrInvalidUTF8
	}

	return &FileContent{
		Type: filepath.Ext(info.Name()),
		Data: string(data),
		Size: info.Size(),
	}, nil
}

// Download 获取文件用于下载
func (s *LocalFileService) Download(path string) (io.ReadCloser, error) {
	absPath, err := s.ValidatePath(path)
	if err != nil {
		return nil, err
	}

	info, err := s.fs.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFileNotFound
		}
		return nil, err
	}

	if info.IsDir() {
		return nil, errors.New("cannot download directory")
	}

	return s.fs.Open(absPath)
}

// Create 创建文件或目录
func (s *LocalFileService) Create(path string, isDir bool) error {
	absPath, err := s.ValidatePath(path)
	if err != nil {
		return err
	}

	if isDir {
		return s.fs.MkdirAll(absPath, 0755)
	}

	file, err := s.fs.Create(absPath)
	if err != nil {
		return err
	}
	return file.Close()
}

// Write 写入文件
func (s *LocalFileService) Write(path string, data []byte) error {
	absPath, err := s.ValidatePath(path)
	if err != nil {
		return err
	}

	return afero.WriteFile(s.fs, absPath, data, 0644)
}

// Rename 重命名/移动
func (s *LocalFileService) Rename(oldPath, newPath string) error {
	absOldPath, err := s.ValidatePath(oldPath)
	if err != nil {
		return err
	}

	absNewPath, err := s.ValidatePath(newPath)
	if err != nil {
		return err
	}

	return s.fs.Rename(absOldPath, absNewPath)
}

// Chmod 修改权限
func (s *LocalFileService) Chmod(path string, permStr string) error {
	absPath, err := s.ValidatePath(path)
	if err != nil {
		return err
	}

	perm, err := strconv.ParseUint(permStr, 8, 32)
	if err != nil {
		return ErrInvalidPerm
	}

	// afero 对权限的支持有限，对于 OsFs 需要底层操作
	if osFS, ok := s.fs.(*afero.OsFs); ok {
		_ = osFS
		return os.Chmod(absPath, os.FileMode(perm))
	}

	return s.fs.Chmod(absPath, os.FileMode(perm))
}

// Delete 删除文件或目录
func (s *LocalFileService) Delete(path string) error {
	absPath, err := s.ValidatePath(path)
	if err != nil {
		return err
	}

	return s.fs.RemoveAll(absPath)
}

// GetBasePath 获取基础路径
func (s *LocalFileService) GetBasePath() string {
	return s.basePath
}
