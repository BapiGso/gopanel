package file

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
)

func newTestHandler(t *testing.T) (*Handler, string) {
	t.Helper()
	root := t.TempDir()
	handler, err := NewHandler(root)
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	t.Cleanup(func() { _ = handler.service.Close() })
	return handler, root
}

func TestHandlerRejectsTraversalInRead(t *testing.T) {
	handler, _ := newTestHandler(t)
	e := echo.New()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/files/../../etc/passwd", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.getEntry(c)
	if err == nil {
		t.Fatal("getEntry() should reject traversal path")
	}
	if !IsPathNotAllowed(err) {
		t.Fatalf("getEntry() error = %v, want path not allowed", err)
	}
}

func TestHandlerFallsBackToRootWhenCookiePathIsInvalid(t *testing.T) {
	handler, root := newTestHandler(t)
	if err := os.WriteFile(filepath.Join(root, "readme.txt"), []byte("ok"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	e := echo.New()
	e.Renderer = testRenderer{}

	req := httptest.NewRequest(http.MethodGet, "/admin/file", nil)
	req.AddCookie(&http.Cookie{Name: "dirHistory", Value: "%2F..%2F..%2Fetc"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.Index(c); err != nil {
		t.Fatalf("Index() error = %v", err)
	}

	found := false
	for _, setCookie := range rec.Header().Values("Set-Cookie") {
		if strings.Contains(setCookie, "dirHistory=%2F") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Set-Cookie headers = %q, want dirHistory to be reset to %%2F", rec.Header().Values("Set-Cookie"))
	}
}

func TestHandlerSupportsChmodMode(t *testing.T) {
	handler, root := newTestHandler(t)
	note := filepath.Join(root, "note.txt")
	if err := os.WriteFile(note, []byte("ok"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	e := echo.New()

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/files/note.txt", strings.NewReader(`{"permissions":"0600"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler.patchEntry(c); err != nil {
		t.Fatalf("patchEntry() error = %v", err)
	}
	info, err := os.Stat(note)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Fatalf("file mode = %v, want 0600", info.Mode().Perm())
	}
	final := fmt.Sprintf("%04o", info.Mode().Perm())
	if !strings.Contains(rec.Body.String(), `"permissions":"`+final+`"`) {
		t.Fatalf("patchEntry() body = %q, want permissions %s in data", rec.Body.String(), final)
	}
}

type testRenderer struct{}

func (testRenderer) Render(_ *echo.Context, w io.Writer, _ string, _ any) error {
	_, err := io.WriteString(w, "ok")
	return err
}
