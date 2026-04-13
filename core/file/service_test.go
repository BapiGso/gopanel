package file

import (
	"strings"
	"testing"

	"github.com/spf13/afero"
)

func TestLocalFileServiceReadEditable(t *testing.T) {
	fs := afero.NewMemMapFs()
	service := NewLocalFileService(fs, "/app")

	if err := afero.WriteFile(fs, "/app/note.txt", []byte("hello"), 0644); err != nil {
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
	fs := afero.NewMemMapFs()
	service := NewLocalFileService(fs, "/app")

	if err := service.CreateFolder("/uploads"); err != nil {
		t.Fatalf("CreateFolder() error = %v", err)
	}
	if err := service.SaveUploadedFile("/uploads", "a.txt", strings.NewReader("payload")); err != nil {
		t.Fatalf("SaveUploadedFile() error = %v", err)
	}

	data, err := afero.ReadFile(fs, "/app/uploads/a.txt")
	if err != nil {
		t.Fatalf("read uploaded file: %v", err)
	}
	if string(data) != "payload" {
		t.Fatalf("uploaded file = %q, want %q", string(data), "payload")
	}
}

func TestLocalFileServiceRejectsPathTraversal(t *testing.T) {
	fs := afero.NewMemMapFs()
	service := NewLocalFileService(fs, "/app")

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

func TestLocalFileServiceRefusesToDeleteRoot(t *testing.T) {
	fs := afero.NewMemMapFs()
	service := NewLocalFileService(fs, "/app")

	if err := service.CreateFolder("/nested"); err != nil {
		t.Fatalf("CreateFolder() error = %v", err)
	}
	if err := service.RemoveAll("/"); err == nil {
		t.Fatal("RemoveAll() should reject deleting the file root")
	}
}
