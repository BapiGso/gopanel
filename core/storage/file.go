package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

var ErrRevisionConflict = errors.New("configuration changed; read it again before saving")

var fileLocks sync.Map

type File struct {
	Path string
}

type Document struct {
	Content  string `json:"content"`
	Revision string `json:"revision"`
}

func Revision(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

func (file *File) Read() (Document, error) {
	path, mutex, err := fileLock(file.Path)
	if err != nil {
		return Document{}, err
	}
	mutex.Lock()
	defer mutex.Unlock()
	return read(path)
}

func (file *File) Ensure(content []byte) error {
	path, mutex, err := fileLock(file.Path)
	if err != nil {
		return err
	}
	mutex.Lock()
	defer mutex.Unlock()
	if _, err := read(path); !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return atomicWrite(path, content)
}

func (file *File) Save(content, revision string) (Document, error) {
	path, mutex, err := fileLock(file.Path)
	if err != nil {
		return Document{}, err
	}
	mutex.Lock()
	defer mutex.Unlock()
	current, err := read(path)
	if err != nil {
		return Document{}, err
	}
	if current.Revision != revision {
		return Document{}, ErrRevisionConflict
	}
	if current.Content == content {
		return current, nil
	}
	if err := atomicWrite(path, []byte(content)); err != nil {
		return Document{}, err
	}
	return Document{Content: content, Revision: Revision([]byte(content))}, nil
}

func AtomicWrite(path string, content []byte) error {
	absolute, mutex, err := fileLock(path)
	if err != nil {
		return err
	}
	mutex.Lock()
	defer mutex.Unlock()
	return atomicWrite(absolute, content)
}

func fileLock(path string) (string, *sync.Mutex, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", nil, err
	}
	key := absolute
	if runtime.GOOS == "windows" {
		key = strings.ToLower(key)
	}
	mutex, _ := fileLocks.LoadOrStore(key, &sync.Mutex{})
	return absolute, mutex.(*sync.Mutex), nil
}

func read(path string) (Document, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Document{}, err
	}
	return Document{Content: string(content), Revision: Revision(content)}, nil
}

func atomicWrite(path string, content []byte) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	defer os.Remove(temporary.Name())
	if err := temporary.Chmod(0600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(content); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return replaceFile(temporary.Name(), path)
}
