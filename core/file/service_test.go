package file

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeBasePathDefaultsToFilesystemRoot(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}

	got := normalizeBasePath("")
	want := string(os.PathSeparator)
	if volume := filepath.VolumeName(cwd); volume != "" {
		want = volume + string(os.PathSeparator)
	}

	if got != want {
		t.Fatalf("normalizeBasePath(\"\") = %q, want filesystem root %q", got, want)
	}
	if got == cwd {
		t.Fatalf("normalizeBasePath(\"\") = cwd %q, want filesystem root", cwd)
	}
}

func newTestService(t *testing.T) (*LocalFileService, string) {
	t.Helper()
	root := t.TempDir()
	service, err := NewLocalFileService(root)
	if err != nil {
		t.Fatalf("NewLocalFileService() error = %v", err)
	}
	t.Cleanup(func() { _ = service.Close() })
	return service, root
}

func TestLocalFileServiceReadEditable(t *testing.T) {
	service, root := newTestService(t)

	if err := os.WriteFile(filepath.Join(root, "note.txt"), []byte("hello"), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	got, err := service.ReadEditable("/note.txt")
	if err != nil {
		t.Fatalf("ReadEditable() error = %v", err)
	}
	if got != "hello" {
		t.Fatalf("ReadEditable() = %q, want %q", got, "hello")
	}
}

func TestLocalFileServiceSaveUploadedFile(t *testing.T) {
	service, root := newTestService(t)

	if err := service.CreateFolder("/uploads"); err != nil {
		t.Fatalf("CreateFolder() error = %v", err)
	}
	if err := service.SaveUploadedFile("/uploads", "a.txt", strings.NewReader("payload")); err != nil {
		t.Fatalf("SaveUploadedFile() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(root, "uploads", "a.txt"))
	if err != nil {
		t.Fatalf("read uploaded file: %v", err)
	}
	if string(data) != "payload" {
		t.Fatalf("uploaded file = %q, want %q", string(data), "payload")
	}
}

func TestLocalFileServiceRejectsPathTraversal(t *testing.T) {
	service, _ := newTestService(t)

	if _, err := service.ReadEditable("/../../etc/passwd"); !IsPathNotAllowed(err) {
		t.Fatalf("ReadEditable() error = %v, want path not allowed", err)
	}
	if err := service.CreateFolder("/../../escape"); !IsPathNotAllowed(err) {
		t.Fatalf("CreateFolder() error = %v, want path not allowed", err)
	}
	if err := service.Rename("/ok.txt", "/../../tmp/outside.txt"); !IsPathNotAllowed(err) {
		t.Fatalf("Rename() error = %v, want path not allowed", err)
	}
}

func TestLocalFileServiceRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	target := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(target, []byte("secret"), 0644); err != nil {
		t.Fatalf("write outside file: %v", err)
	}
	if err := os.Symlink(target, filepath.Join(root, "link")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	service, err := NewLocalFileService(root)
	if err != nil {
		t.Fatalf("NewLocalFileService() error = %v", err)
	}
	t.Cleanup(func() { _ = service.Close() })
	if _, err := service.ReadEditable("/link"); err == nil {
		t.Fatal("ReadEditable() should reject a symlink escaping the root")
	}
}

func TestLocalFileServiceRefusesToDeleteRoot(t *testing.T) {
	service, _ := newTestService(t)

	if err := service.CreateFolder("/nested"); err != nil {
		t.Fatalf("CreateFolder() error = %v", err)
	}
	if err := service.RemoveAll("/"); err == nil {
		t.Fatal("RemoveAll() should reject deleting the file root")
	}
}
