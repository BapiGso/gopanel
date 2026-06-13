package file

import (
	"errors"
	"fmt"
	"io"
	"os"
	pathpkg "path"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/labstack/echo/v5"
	"github.com/spf13/afero"
)

var ErrPathNotAllowed = echo.NewHTTPError(403, "path is outside the allowed root")

type LocalFileService struct {
	fs       afero.Fs
	basePath string
}

func NewLocalFileService(fs afero.Fs, basePath string) *LocalFileService {
	root := normalizeBasePath(basePath)
	return &LocalFileService{
		fs:       afero.NewBasePathFs(fs, root),
		basePath: root,
	}
}

func (s *LocalFileService) ReadEditable(path string) (string, error) {
	allowedPath, err := s.resolve(path)
	if err != nil {
		return "", err
	}

	file, err := s.fs.Stat(allowedPath)
	if err != nil {
		return "", err
	}
	if file.Size() > 2*1024*1024 {
		return "", echo.NewHTTPError(400, fmt.Sprintf("file too big %d", file.Size()))
	}

	data, err := afero.ReadFile(s.fs, allowedPath)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(data) {
		return "", echo.NewHTTPError(400, "not valid UTF-8")
	}
	return string(data), nil
}

func (s *LocalFileService) Open(path string) (afero.File, error) {
	allowedPath, err := s.resolve(path)
	if err != nil {
		return nil, err
	}
	return s.fs.Open(allowedPath)
}

func (s *LocalFileService) SaveUploadedFile(dir, filename string, src io.Reader) error {
	allowedDir, err := s.resolve(dir)
	if err != nil {
		return err
	}
	allowedPath, err := s.resolve(pathpkg.Join(allowedDir, filename))
	if err != nil {
		return err
	}

	dst, err := s.fs.OpenFile(allowedPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

func (s *LocalFileService) Rename(path, next string) error {
	currentPath, err := s.resolve(path)
	if err != nil {
		return err
	}
	nextPath, err := s.resolve(next)
	if err != nil {
		return err
	}
	return s.fs.Rename(currentPath, nextPath)
}

func (s *LocalFileService) Chmod(path, permText string) error {
	allowedPath, err := s.resolve(path)
	if err != nil {
		return err
	}
	perm, err := strconv.ParseUint(permText, 8, 32)
	if err != nil {
		return err
	}
	return s.fs.Chmod(allowedPath, os.FileMode(perm))
}

func (s *LocalFileService) WriteFile(path string, data []byte) error {
	allowedPath, err := s.resolve(path)
	if err != nil {
		return err
	}
	return afero.WriteFile(s.fs, allowedPath, data, 0644)
}

func (s *LocalFileService) CreateFile(path string) error {
	allowedPath, err := s.resolve(path)
	if err != nil {
		return err
	}
	file, err := s.fs.Create(allowedPath)
	if err != nil {
		return err
	}
	return file.Close()
}

func (s *LocalFileService) CreateFolder(path string) error {
	allowedPath, err := s.resolve(path)
	if err != nil {
		return err
	}
	return s.fs.MkdirAll(allowedPath, 0755)
}

func (s *LocalFileService) RemoveAll(path string) error {
	allowedPath, err := s.resolve(path)
	if err != nil {
		return err
	}
	if allowedPath == "." {
		return echo.NewHTTPError(400, "refusing to delete the file root")
	}
	return s.fs.RemoveAll(allowedPath)
}

func (s *LocalFileService) ReadDir(path string) ([]os.FileInfo, error) {
	allowedPath, err := s.resolve(path)
	if err != nil {
		return nil, err
	}
	return afero.ReadDir(s.fs, allowedPath)
}

func (s *LocalFileService) Root() string {
	return "/"
}

func (s *LocalFileService) resolve(path string) (string, error) {
	if hasTraversal(path) {
		return "", ErrPathNotAllowed
	}

	cleanPath := normalizeRequestPath(path)
	relPath := strings.TrimPrefix(cleanPath, "/")
	if relPath == "" {
		return ".", nil
	}
	return relPath, nil
}

func normalizeBasePath(path string) string {
	if path == "" {
		return filesystemRoot()
	}
	return path
}

func filesystemRoot() string {
	if cwd, err := os.Getwd(); err == nil {
		if volume := filepath.VolumeName(cwd); volume != "" {
			return volume + string(os.PathSeparator)
		}
	}
	return string(os.PathSeparator)
}

func normalizeRequestPath(pathValue string) string {
	cleaned := strings.ReplaceAll(pathValue, "\\", "/")
	if cleaned == "" {
		return "/"
	}
	if !strings.HasPrefix(cleaned, "/") {
		cleaned = "/" + cleaned
	}
	return pathpkg.Clean(cleaned)
}

func hasTraversal(pathValue string) bool {
	pathValue = strings.ReplaceAll(pathValue, "\\", "/")
	for _, segment := range strings.Split(pathValue, "/") {
		if segment == ".." {
			return true
		}
	}
	return false
}

func IsPathNotAllowed(err error) bool {
	var httpErr *echo.HTTPError
	return errors.As(err, &httpErr) && httpErr.Code == 403 && httpErr.Message == ErrPathNotAllowed.Message
}
