package storage

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

func TestAtomicWriteReplacesFilesAndCleansTemporaryFiles(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "config")
	for _, content := range []string{"initial", "updated 配置", ""} {
		if err := AtomicWrite(path, []byte(content)); err != nil {
			t.Fatal(err)
		}
		actual, err := os.ReadFile(path)
		if err != nil || string(actual) != content {
			t.Fatalf("content = %q, error = %v", actual, err)
		}
		entries, err := os.ReadDir(directory)
		if err != nil || len(entries) != 1 || entries[0].Name() != "config" {
			t.Fatalf("temporary files remain: %v, error = %v", entries, err)
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Fatalf("permissions = %o, want 0600", info.Mode().Perm())
	}
}

func TestAtomicWriteFailedReplacementPreservesTarget(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "config")
	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(target, "original")
	if err := os.WriteFile(marker, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := AtomicWrite(target, []byte("replacement")); err == nil {
		t.Fatal("replacement of a directory should fail")
	}
	content, err := os.ReadFile(marker)
	if err != nil || string(content) != "keep" {
		t.Fatal("failed replacement changed the original target")
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 1 {
		t.Fatalf("failed replacement left temporary files: %v", err)
	}
}

func TestIndependentFileHandlesRejectConcurrentLostUpdates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")
	first := &File{Path: path}
	second := &File{Path: path}
	if err := first.Ensure([]byte("initial")); err != nil {
		t.Fatal(err)
	}
	initial, err := first.Read()
	if err != nil {
		t.Fatal(err)
	}
	var writers sync.WaitGroup
	failures := make(chan error, 2)
	for index, file := range []*File{first, second} {
		writers.Go(func() {
			_, err := file.Save([]string{"first", "second"}[index], initial.Revision)
			failures <- err
		})
	}
	writers.Wait()
	close(failures)
	successes, conflicts := 0, 0
	for err := range failures {
		if err == nil {
			successes++
		} else if errors.Is(err, ErrRevisionConflict) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes = %d, conflicts = %d", successes, conflicts)
	}
}

func TestNoOpSaveChecksRevisionAndEnsurePreservesExistingContent(t *testing.T) {
	file := &File{Path: filepath.Join(t.TempDir(), "config")}
	if err := file.Ensure([]byte("{broken")); err != nil {
		t.Fatal(err)
	}
	if err := file.Ensure([]byte("defaults")); err != nil {
		t.Fatal(err)
	}
	initial, err := file.Read()
	if err != nil || initial.Content != "{broken" {
		t.Fatal("Ensure overwrote an existing file")
	}
	unchanged, err := file.Save(initial.Content, initial.Revision)
	if err != nil || unchanged != initial {
		t.Fatal("no-op save changed content or revision")
	}
	current, err := file.Save("updated", initial.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Save(current.Content, initial.Revision); !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("stale no-op save = %v, want revision conflict", err)
	}
}
