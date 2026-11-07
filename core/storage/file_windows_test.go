package storage

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/windows"
)

func TestAtomicWriteLockedWindowsFilePreservesOriginal(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "config")
	if err := AtomicWrite(path, []byte("original")); err != nil {
		t.Fatal(err)
	}
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := windows.CreateFile(name, windows.GENERIC_READ, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(handle)
	if err := AtomicWrite(path, []byte("replacement")); err == nil {
		t.Fatal("replacing a file without delete sharing should fail")
	}
	content, err := os.ReadFile(path)
	if err != nil || string(content) != "original" {
		t.Fatal("failed replacement truncated the original file")
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 {
		t.Fatal("failed replacement left a temporary file")
	}
}
