package file

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/labstack/echo/v5"
)

var ErrPathNotAllowed = echo.NewHTTPError(403, "path is outside the allowed root")

// LocalFileService confines all operations to a directory tree using os.Root.
// Traversal and symlink escapes are rejected by the os package and the kernel.
type LocalFileService struct {
	root *os.Root
}

func NewLocalFileService(basePath string) (*LocalFileService, error) {
	root, err := os.OpenRoot(normalizeBasePath(basePath))
	if err != nil {
		return nil, err
	}
	return &LocalFileService{root: root}, nil
}

// Close releases the root directory handle held by the service.
func (s *LocalFileService) Close() error {
	return s.root.Close()
}

func (s *LocalFileService) ReadEditable(path string) (string, error) {
	allowedPath, err := s.resolve(path)
	if err != nil {
		return "", err
	}

	info, err := s.root.Stat(allowedPath)
	if err != nil {
		return "", err
	}
	if info.Size() > 2*1024*1024 {
		return "", echo.NewHTTPError(400, fmt.Sprintf("file too big %d", info.Size()))
	}

	data, err := s.root.ReadFile(allowedPath)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(data) {
		return "", echo.NewHTTPError(400, "not valid UTF-8")
	}
	return string(data), nil
}

func (s *LocalFileService) Open(path string) (*os.File, error) {
	allowedPath, err := s.resolve(path)
	if err != nil {
		return nil, err
	}
	return s.root.Open(allowedPath)
}

func (s *LocalFileService) SaveUploadedFile(dir, filename string, src io.Reader) error {
	if filename == "" || filename == "." || filename == ".." || strings.ContainsAny(filename, "/\\") {
		return ErrPathNotAllowed
	}
	allowedDir, err := s.resolve(dir)
	if err != nil {
		return err
	}
	allowedPath, err := s.resolve(filepath.Join(allowedDir, filename))
	if err != nil {
		return err
	}

	dst, err := s.root.OpenFile(allowedPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
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
	return s.root.Rename(currentPath, nextPath)
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
	return s.root.Chmod(allowedPath, os.FileMode(perm))
}

func (s *LocalFileService) WriteFile(path string, data []byte) error {
	allowedPath, err := s.resolve(path)
	if err != nil {
		return err
	}
	return s.root.WriteFile(allowedPath, data, 0644)
}

func (s *LocalFileService) CreateFile(path string) error {
	allowedPath, err := s.resolve(path)
	if err != nil {
		return err
	}
	file, err := s.root.OpenFile(allowedPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
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
	return s.root.MkdirAll(allowedPath, 0755)
}

func (s *LocalFileService) RemoveAll(path string) error {
	allowedPath, err := s.resolve(path)
	if err != nil {
		return err
	}
	if allowedPath == "." {
		return echo.NewHTTPError(400, "refusing to delete the file root")
	}
	return s.root.RemoveAll(allowedPath)
}

func (s *LocalFileService) ReadDir(path string) ([]os.FileInfo, error) {
	allowedPath, err := s.resolve(path)
	if err != nil {
		return nil, err
	}
	entries, err := fs.ReadDir(s.root.FS(), allowedPath)
	if err != nil {
		return nil, err
	}
	infos := make([]os.FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		infos = append(infos, info)
	}
	return infos, nil
}

func (s *LocalFileService) Stat(path string) (os.FileInfo, error) {
	allowedPath, err := s.resolve(path)
	if err != nil {
		return nil, err
	}
	return s.root.Stat(allowedPath)
}

func (s *LocalFileService) Root() string {
	return "/"
}

// resolve normalizes a logical path to a root-relative path. os.Root applies
// the final containment check, so this only needs to reject obvious escapes.
func (s *LocalFileService) resolve(requested string) (string, error) {
	if requested == "" || strings.IndexByte(requested, 0) >= 0 {
		return "", ErrPathNotAllowed
	}

	slashPath := strings.ReplaceAll(requested, "\\", "/")
	if hasTraversalSegment(slashPath) {
		return "", ErrPathNotAllowed
	}

	cleaned := path.Clean("/" + slashPath)
	relPath := strings.TrimPrefix(cleaned, "/")
	if relPath == "" {
		return ".", nil
	}
	return relPath, nil
}

func hasTraversalSegment(path string) bool {
	for _, segment := range strings.Split(path, "/") {
		if segment == ".." {
			return true
		}
	}
	return false
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

func IsPathNotAllowed(err error) bool {
	var httpErr *echo.HTTPError
	return errors.As(err, &httpErr) && httpErr.Code == 403 && httpErr.Message == ErrPathNotAllowed.Message
}
